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
func (s *Service) SearchManga(params SearchParams) (*SearchResult, error) {
	// Validate and set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	if !s.UseMangaDex {
		return nil, errors.New("mangadex integration disabled")
	}

	// Always use MangaDex; if it fails, bubble the error to the caller.
	return s.searchMangaDexAndCache(params)
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
				manga := mangadex.TransformMangaDexToManga(&mdManga, s.MangaDex, false) // false = don't use aggregate for list views
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
					sample := mangadex.TransformMangaDexToManga(&mdResp.Data[0], s.MangaDex, false) // false = don't use aggregate
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
		log.Printf("[Manga] Error fetching from MangaDex: %v", err)
		return nil, err
	}

	log.Printf("[Manga] MangaDex returned %d results, total: %d", len(mdResp.Data), mdResp.Total)

	// Transform and cache results
	var mangas []models.Manga
	for _, mdManga := range mdResp.Data {
		manga := mangadex.TransformMangaDexToManga(&mdManga, s.MangaDex, false) // false = don't use aggregate for list views
		if manga != nil {
			mangas = append(mangas, *manga)
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

// isUUID checks if a string looks like a UUID (MangaDex ID format)
func isUUID(s string) bool {
	// MangaDex IDs are UUIDs: 8-4-4-4-12 format (36 chars total with hyphens)
	// Example: "9e7def13-2ce9-424b-a178-1f907023ffea"
	if len(s) != 36 {
		return false
	}
	// Check for hyphens at positions 8, 13, 18, 23
	if len(s) >= 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	return false
}

// GetMangaByID retrieves a single manga by ID.
// Supports both prefixed ("mangadex-{id}") and raw UUID formats.
// If not found in local DB, tries to fetch from MangaDex if it's a MangaDex ID.
func (s *Service) GetMangaByID(id string) (*models.Manga, error) {
	// Try local database first with the ID as-is
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

	// Not found with the ID as-is
	// If it's a raw UUID (MangaDex format), try with "mangadex-" prefix
	if isUUID(id) {
		prefixedID := "mangadex-" + id
		err := s.DB.QueryRow(
			`SELECT id, title, author, genres, status, total_chapters, description, cover_url 
			FROM manga WHERE id = ?`,
			prefixedID,
		).Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)

		if err == nil {
			// Found with prefixed ID
			m.Genres = parseGenres(genresJSON)
			return &m, nil
		}

		if err != sql.ErrNoRows {
			log.Printf("Error querying manga with prefixed ID: %v", err)
			return nil, errors.New("failed to query manga")
		}

		// Still not found - try fetching from MangaDex
		if s.UseMangaDex {
			log.Printf("Manga %s not in local DB, fetching from MangaDex...", id)
			mdManga, err := s.MangaDex.GetMangaByID(id)
			if err != nil {
				log.Printf("Failed to fetch from MangaDex: %v", err)
				return nil, errors.New("not_found")
			}

			// Transform and cache the manga (use aggregate for detail page)
			manga := mangadex.TransformMangaDexToManga(mdManga, s.MangaDex, true) // true = use aggregate for detail pages
			if manga != nil {
				s.cacheManga(manga)
				return manga, nil
			}
		}
	} else if strings.HasPrefix(id, "mangadex-") && s.UseMangaDex {
		// ID has "mangadex-" prefix, extract the actual MangaDex ID
		mangaDexID := strings.TrimPrefix(id, "mangadex-")
		log.Printf("Manga %s not in local DB, fetching from MangaDex...", mangaDexID)
		mdManga, err := s.MangaDex.GetMangaByID(mangaDexID)
		if err != nil {
			log.Printf("Failed to fetch from MangaDex: %v", err)
			return nil, errors.New("not_found")
		}

		// Transform and cache the manga (use aggregate for detail page)
		manga := mangadex.TransformMangaDexToManga(mdManga, s.MangaDex, true) // true = use aggregate for detail pages
		if manga != nil {
			s.cacheManga(manga)
			return manga, nil
		}
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
