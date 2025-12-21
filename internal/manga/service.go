package manga

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"mangahub/pkg/models"
)

// Service contains core manga data management logic.
type Service struct {
	DB *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
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
	offset := (params.Page - 1) * params.Limit

	// Build WHERE clause
	var whereConditions []string
	var args []interface{}

	if params.Query != "" {
		searchPattern := "%" + strings.ToLower(params.Query) + "%"
		whereConditions = append(whereConditions, "(LOWER(title) LIKE ? OR LOWER(author) LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	if params.Genre != "" {
		// SQLite JSON: check if genre exists in JSON array
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

	// Get total count for pagination
	countQuery := "SELECT COUNT(*) FROM manga " + whereClause
	var total int
	err := s.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Error counting manga: %v", err)
		return nil, errors.New("failed to count manga")
	}

	// Get paginated results
	querySQL := `SELECT id, title, author, genres, status, total_chapters, description, cover_url 
		FROM manga ` + whereClause + ` ORDER BY title LIMIT ? OFFSET ?`
	args = append(args, params.Limit, offset)

	rows, err := s.DB.Query(querySQL, args...)
	if err != nil {
		log.Printf("Error querying manga: %v", err)
		return nil, errors.New("failed to query manga")
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

		// Parse genres JSON
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

// GetMangaByID retrieves a single manga by ID.
func (s *Service) GetMangaByID(id string) (*models.Manga, error) {
	var m models.Manga
	var genresJSON string
	err := s.DB.QueryRow(
		`SELECT id, title, author, genres, status, total_chapters, description, cover_url 
		FROM manga WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("not_found")
		}
		log.Printf("Error querying manga: %v", err)
		return nil, errors.New("failed to query manga")
	}

	m.Genres = parseGenres(genresJSON)
	return &m, nil
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

