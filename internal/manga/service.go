package manga

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"

	"mangahub/internal/mangadex"
	"mangahub/pkg/models"
)

// Service contains core manga data management logic.
type Service struct {
	DB          *sql.DB
	MangaDex    *mangadex.Client
	UseMangaDex bool // Whether to fetch from MangaDex when local DB has few results
}

func NewService(db *sql.DB) *Service {
	useMangaDex := os.Getenv("MANGAHUB_USE_MANGADEX")
	return &Service{
		DB:          db,
		MangaDex:    mangadex.NewClient(),
		UseMangaDex: useMangaDex != "false", // Default to true unless explicitly disabled
	}
}

// SearchParams holds search and filter parameters.
type SearchParams struct {
	Query  string
	Genre  string
	Status string
	Page   int
	Limit  int
}

// SearchResult contains paginated manga results.
type SearchResult struct {
	Data       []models.Manga
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

// SearchManga implements UC-003: search manga with filters and pagination.
// If local DB has few results and UseMangaDex is true, fetches from MangaDex and caches results.
func (s *Service) SearchManga(params SearchParams) (*SearchResult, error) {
	// Validate and set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	// If filtering by genre and MangaDex is enabled, always use MangaDex + client-side genre filter
	// This ensures we see the full MangaDex catalog for that genre, not just the small local DB.
	if params.Genre != "" && s.UseMangaDex {
		log.Printf("[Manga] Genre filter active (%s) - using MangaDex with client-side genre filtering", params.Genre)
		return s.searchMangaDexAndCache(params)
	}

	// Strategy:
	// 1. If no query/filter (browsing all): Always fetch from MangaDex for full catalog (80,000+)
	// 2. If query/filter: Check local DB first, fallback to MangaDex if no results
	// 3. Cache all results from MangaDex for future queries

	// For "all/trending" view (no query/filter), always use MangaDex to show full catalog
	if params.Query == "" && params.Genre == "" && params.Status == "" {
		if s.UseMangaDex {
			log.Printf("[Manga] Browsing all manga - fetching from MangaDex for full catalog")
			return s.searchMangaDexAndCache(params)
		}
		// If MangaDex disabled, use local DB
		return s.searchLocalDB(params)
	}

	// For specific queries/filters: try local DB first, then MangaDex
	localResult, err := s.searchLocalDB(params)
	if err != nil {
		log.Printf("Error searching local DB: %v", err)
	}

	// If local DB has results for this query, use them
	if localResult != nil && localResult.Total > 0 {
		log.Printf("[Manga] Found %d results in local DB for query/filter", localResult.Total)
		return localResult, nil
	}

	// No local results - try MangaDex
	if s.UseMangaDex {
		log.Printf("[Manga] No local results for query/filter, fetching from MangaDex")
		return s.searchMangaDexAndCache(params)
	}

	// No results and MangaDex disabled
	if localResult == nil {
		return &SearchResult{
			Data:       []models.Manga{},
			Total:      0,
			Page:       params.Page,
			Limit:      params.Limit,
			TotalPages: 0,
		}, nil
	}

	return localResult, nil
}

// searchLocalDB searches only the local database
func (s *Service) searchLocalDB(params SearchParams) (*SearchResult, error) {
	// Build WHERE clause
	var whereConditions []string
	var args []interface{}

	if params.Query != "" {
		searchPattern := "%" + strings.ToLower(params.Query) + "%"
		whereConditions = append(whereConditions, "(LOWER(title) LIKE ? OR LOWER(author) LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	if params.Genre != "" {
		whereConditions = append(whereConditions, "genres LIKE ?")
		args = append(args, "%\""+params.Genre+"\"%")
	}

	if params.Status != "" {
		whereConditions = append(whereConditions, "status = ?")
		args = append(args, params.Status)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM manga " + whereClause
	var total int
	err := s.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get paginated results
	offset := (params.Page - 1) * params.Limit
	querySQL := `SELECT id, title, author, genres, status, total_chapters, description, cover_url 
		FROM manga ` + whereClause + ` ORDER BY title LIMIT ? OFFSET ?`
	args = append(args, params.Limit, offset)

	rows, err := s.DB.Query(querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresJSON string
		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL); err != nil {
			log.Printf("Error scanning manga row: %v", err)
			continue
		}
		m.Genres = parseGenres(genresJSON)
		result = append(result, m)
	}

	totalPages := (total + params.Limit - 1) / params.Limit
	return &SearchResult{
		Data:       result,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchMangaDexAndCache fetches from MangaDex and caches results in local DB
// For genre filtering, fetches larger batches and filters client-side (MangaDex requires tag IDs, not names)
func (s *Service) searchMangaDexAndCache(params SearchParams) (*SearchResult, error) {
	offset := (params.Page - 1) * params.Limit

	log.Printf("[Manga] Fetching from MangaDex: query=%s, genre=%s, status=%s, limit=%d, offset=%d", 
		params.Query, params.Genre, params.Status, params.Limit, offset)

	// If filtering by genre, we need to fetch larger batches and filter client-side
	// because MangaDex doesn't accept genre names, only tag IDs
	hasGenreFilter := params.Genre != ""
	
	var allMangas []models.Manga
	var mangadexTotal int
	batchSize := 100
	maxBatches := 10 // Safety limit
	currentOffset := 0
	totalFetched := 0
	totalFiltered := 0

	if hasGenreFilter {
		// Fetch multiple batches to get enough filtered results
		// Increase max batches for rare genres - fetch up to 50 batches (5000 manga) if needed
		maxBatches = 50
		targetCount := offset + params.Limit
		
		// Also try partial matching (contains) in addition to exact match
		genreLower := strings.ToLower(params.Genre)
		
		for len(allMangas) < targetCount && currentOffset < maxBatches*batchSize {
			mdResp, err := s.MangaDex.SearchManga(params.Query, "", params.Status, batchSize, currentOffset)
			if err != nil {
				log.Printf("[Manga] Error fetching batch from MangaDex: %v", err)
				break
			}

			if mangadexTotal == 0 && mdResp.Total > 0 {
				mangadexTotal = mdResp.Total
			}

			// Transform and filter by genre
			batchFiltered := 0
			for _, mdManga := range mdResp.Data {
				manga := mangadex.TransformMangaDexToManga(&mdManga)
				if manga != nil {
					totalFetched++
					// Filter by genre (case-insensitive, also try partial match)
					genreMatch := false
					for _, g := range manga.Genres {
						gLower := strings.ToLower(g)
						// Exact match or contains match
						if strings.EqualFold(g, params.Genre) || strings.Contains(gLower, genreLower) || strings.Contains(genreLower, gLower) {
							genreMatch = true
							break
						}
					}
					if genreMatch {
						allMangas = append(allMangas, *manga)
						totalFiltered++
						batchFiltered++
						// Cache in local DB (async)
						go s.cacheManga(manga)
					}
				}
			}

			// Log progress every 5 batches
			if (currentOffset/batchSize+1)%5 == 0 {
				log.Printf("[Manga] Genre filter progress: fetched %d batches (%d manga), found %d matching '%s'", 
					currentOffset/batchSize+1, totalFetched, totalFiltered, params.Genre)
			}

			// If we've fetched enough batches but still no matches, log sample genres
			if currentOffset >= 500 && totalFiltered == 0 {
				// Sample a few manga to see what genres we're getting
				if len(mdResp.Data) > 0 {
					sample := mangadex.TransformMangaDexToManga(&mdResp.Data[0])
					if sample != nil && len(sample.Genres) > 0 {
						sampleCount := 3
						if len(sample.Genres) < sampleCount {
							sampleCount = len(sample.Genres)
						}
						log.Printf("[Manga] Sample genres found (looking for '%s'): %v", params.Genre, sample.Genres[:sampleCount])
					}
				}
			}

			if len(mdResp.Data) < batchSize {
				break // No more data
			}
			currentOffset += batchSize
			
			// If we've found enough for pagination and have a good sample, we can stop early
			if len(allMangas) >= targetCount && totalFetched >= 1000 {
				break
			}
		}
		
		log.Printf("[Manga] Genre filter complete: fetched %d manga, found %d matching '%s'", totalFetched, totalFiltered, params.Genre)

		// Calculate estimated total based on filter ratio
		var estimatedTotal int
		if mangadexTotal > 0 && totalFetched > 0 {
			filterRatio := float64(totalFiltered) / float64(totalFetched)
			estimatedTotal = int(float64(mangadexTotal) * filterRatio)
		} else {
			estimatedTotal = len(allMangas)
		}

		// Paginate filtered results
		var paginatedMangas []models.Manga
		if offset < len(allMangas) {
			end := offset + params.Limit
			if end > len(allMangas) {
				end = len(allMangas)
			}
			paginatedMangas = allMangas[offset:end]
		} else {
			paginatedMangas = []models.Manga{}
		}
		
		// If no results found, log a warning
		if len(paginatedMangas) == 0 && totalFiltered == 0 {
			log.Printf("[Manga] Warning: No manga found matching genre '%s' after fetching %d manga", params.Genre, totalFetched)
		}

		totalPages := (estimatedTotal + params.Limit - 1) / params.Limit
		return &SearchResult{
			Data:       paginatedMangas,
			Total:      estimatedTotal,
			Page:       params.Page,
			Limit:      params.Limit,
			TotalPages: totalPages,
		}, nil
	}

	// No genre filter - direct fetch
	mdResp, err := s.MangaDex.SearchManga(params.Query, "", params.Status, params.Limit, offset)
	if err != nil {
		log.Printf("[Manga] Error fetching from MangaDex: %v, falling back to local DB", err)
		return s.searchLocalDB(params)
	}

	log.Printf("[Manga] MangaDex returned %d results, total: %d", len(mdResp.Data), mdResp.Total)

	// Transform and cache results
	var mangas []models.Manga
	for _, mdManga := range mdResp.Data {
		manga := mangadex.TransformMangaDexToManga(&mdManga)
		if manga != nil {
			mangas = append(mangas, *manga)
			// Cache in local DB (async, don't block response)
			go s.cacheManga(manga)
		}
	}

	total := mdResp.Total
	if total == 0 {
		total = len(mangas) // Fallback if MangaDex doesn't provide total
	}

	totalPages := (total + params.Limit - 1) / params.Limit
	return &SearchResult{
		Data:       mangas,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// cacheManga stores a manga in the local database for future queries
func (s *Service) cacheManga(m *models.Manga) {
	genresJSON, err := json.Marshal(m.Genres)
	if err != nil {
		log.Printf("Error marshaling genres for caching: %v", err)
		return
	}

	_, err = s.DB.Exec(
		`INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Title, m.Author, string(genresJSON), m.Status, m.TotalChapters, m.Description, m.CoverURL,
	)
	if err != nil {
		log.Printf("Error caching manga %s: %v", m.ID, err)
	}
}

// GetMangaByID retrieves a single manga by ID.
// If not found in local DB and ID starts with "mangadex-", tries to fetch from MangaDex.
func (s *Service) GetMangaByID(id string) (*models.Manga, error) {
	// Try local database first
	var m models.Manga
	var genresJSON string
	err := s.DB.QueryRow(
		`SELECT id, title, author, genres, status, total_chapters, description, cover_url 
		FROM manga WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
	
	if err == nil {
		// Found in local DB
		m.Genres = parseGenres(genresJSON)
		return &m, nil
	}

	if err != sql.ErrNoRows {
		log.Printf("Error querying manga: %v", err)
		return nil, errors.New("failed to query manga")
	}

	// Not found in local DB
	// If ID starts with "mangadex-", try fetching from MangaDex
	if strings.HasPrefix(id, "mangadex-") && s.UseMangaDex {
		mangaDexID := strings.TrimPrefix(id, "mangadex-")
		// For now, we'd need a GetMangaByID in MangaDex client
		// This is a placeholder - you can implement it if needed
		log.Printf("Manga %s not in local DB, would fetch from MangaDex (not implemented)", mangaDexID)
	}

	return nil, errors.New("not_found")
}

// MangaWithProgress combines manga details with user progress.
type MangaWithProgress struct {
	Manga        *models.Manga
	UserProgress *models.UserProgress `json:"user_progress"`
}

// GetMangaByIDWithProgress retrieves a single manga by ID and includes user progress if userID is provided.
func (s *Service) GetMangaByIDWithProgress(mangaID, userID string) (*MangaWithProgress, error) {
	manga, err := s.GetMangaByID(mangaID)
	if err != nil {
		return nil, err
	}

	result := &MangaWithProgress{
		Manga:        manga,
		UserProgress: nil,
	}

	// If userID is provided, fetch user progress
	if userID != "" {
		var progress models.UserProgress
		err := s.DB.QueryRow(
			`SELECT user_id, manga_id, current_chapter, status, updated_at 
			FROM user_progress 
			WHERE user_id = ? AND manga_id = ?`,
			userID, mangaID,
		).Scan(&progress.UserID, &progress.MangaID, &progress.CurrentChapter, &progress.Status, &progress.UpdatedAt)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Error querying user progress: %v", err)
			}
			// User progress not found is OK, just leave it as nil
		} else {
			result.UserProgress = &progress
		}
	}

	return result, nil
}

// parseGenres parses JSON genres string, with fallback for comma-separated values.
func parseGenres(genresJSON string) []string {
	if genresJSON == "" {
		return []string{}
	}

	var genres []string
	if err := json.Unmarshal([]byte(genresJSON), &genres); err != nil {
		// Fallback: try comma-separated
		if strings.Contains(genresJSON, ",") {
			parts := strings.Split(genresJSON, ",")
			genres = make([]string, 0, len(parts))
			for _, p := range parts {
				genres = append(genres, strings.TrimSpace(p))
			}
		} else if genresJSON != "" {
			genres = []string{strings.TrimSpace(genresJSON)}
		} else {
			genres = []string{}
		}
	}
	return genres
}

