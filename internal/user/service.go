package user

import (
	"database/sql"
	"errors"
	"log"

	"mangahub/pkg/models"
)

// Service contains user library management logic.
type Service struct {
	DB *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
}

// AddToLibraryRequest represents a request to add manga to user's library.
type AddToLibraryRequest struct {
	MangaID        string
	CurrentChapter int
	Status         string
}

// AddToLibrary adds a manga to user's library or updates if it exists.
func (s *Service) AddToLibrary(userID string, req AddToLibraryRequest) error {
	// Validate manga exists
	var exists bool
	err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM manga WHERE id = ?)", req.MangaID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking manga existence: %v", err)
		return errors.New("failed to validate manga")
	}
	if !exists {
		return errors.New("manga_not_found")
	}

	_, err = s.DB.Exec(
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status) 
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, manga_id) DO UPDATE SET 
			current_chapter = excluded.current_chapter, 
			status = excluded.status, 
			updated_at = CURRENT_TIMESTAMP`,
		userID, req.MangaID, req.CurrentChapter, req.Status,
	)
	if err != nil {
		log.Printf("Error adding to library: %v", err)
		return errors.New("failed to save library entry")
	}
	return nil
}

// GetLibrary retrieves all manga in user's library.
func (s *Service) GetLibrary(userID string) ([]models.UserProgress, error) {
	rows, err := s.DB.Query(
		`SELECT manga_id, current_chapter, status, updated_at 
		FROM user_progress 
		WHERE user_id = ? 
		ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		log.Printf("Error querying library: %v", err)
		return nil, errors.New("failed to query library")
	}
	defer rows.Close()

	var items []models.UserProgress
	for rows.Next() {
		var p models.UserProgress
		p.UserID = userID
		if err := rows.Scan(&p.MangaID, &p.CurrentChapter, &p.Status, &p.UpdatedAt); err != nil {
			log.Printf("Error scanning library row: %v", err)
			continue
		}
		items = append(items, p)
	}
	return items, nil
}

// UpdateProgressRequest represents a request to update reading progress.
type UpdateProgressRequest struct {
	MangaID        string
	CurrentChapter int
	Status         string
}

// UpdateProgress updates user's reading progress for a manga.
func (s *Service) UpdateProgress(userID string, req UpdateProgressRequest) error {
	_, err := s.DB.Exec(
		`UPDATE user_progress 
		SET current_chapter = ?, 
			status = COALESCE(NULLIF(?, ''), status), 
			updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = ? AND manga_id = ?`,
		req.CurrentChapter, req.Status, userID, req.MangaID,
	)
	if err != nil {
		log.Printf("Error updating progress: %v", err)
		return errors.New("failed to update progress")
	}
	return nil
}

