package user

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"mangahub/pkg/models"
)

// Service contains user library management logic.
type Service struct {
	DB       *sql.DB
	TCPAddr  string      // TCP server address for broadcasting progress updates
	MangaSvc MangaService // Interface to get manga metadata for validation
}

// MangaService interface for getting manga metadata.
type MangaService interface {
	GetMangaByID(mangaID string) (*models.Manga, error)
}

func NewService(db *sql.DB) *Service {
	tcpAddr := os.Getenv("MANGAHUB_TCP_ADDR")
	if tcpAddr == "" {
		tcpAddr = "localhost:9090" // Default TCP server port
	}
	return &Service{
		DB:      db,
		TCPAddr: tcpAddr,
	}
}

// SetMangaService sets the manga service for validation.
func (s *Service) SetMangaService(mangaSvc MangaService) {
	s.MangaSvc = mangaSvc
}

// -------------------- Notification Subscriptions (UC-009) --------------------

// SubscribeToMangaNotifications subscribes the user to notifications for a manga
// and registers them with the UDP notification server.
func (s *Service) SubscribeToMangaNotifications(userID, mangaID string) error {
	if userID == "" || mangaID == "" {
		return errors.New("validation_error: user_id and manga_id are required")
	}

	// Store subscription in DB (idempotent thanks to PRIMARY KEY)
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO user_notifications (user_id, manga_id) VALUES (?, ?)`,
		userID, mangaID,
	)
	if err != nil {
		log.Printf("Error saving notification subscription: %v", err)
		return errors.New("database_error: failed to save notification subscription")
	}

	// Register client with UDP notification server based on userID and mangaID.
	// For this project we use localhost:9091; this can be moved to config if needed.
	_, udpErr := RegisterForUDPNotifications(UDPRegisterOptions{
		ServerAddr: "localhost:9091",
		UserID:     userID,
		MangaIDs:   []string{mangaID},
	})
	if udpErr != nil {
		log.Printf("UDP registration failed for user %s manga %s: %v", userID, mangaID, udpErr)
		// We still treat subscription as successful in DB; frontend can show a warning if needed.
	}

	return nil
}

// IsSubscribedToMangaNotifications checks if the user is subscribed for a manga.
func (s *Service) IsSubscribedToMangaNotifications(userID, mangaID string) (bool, error) {
	if userID == "" || mangaID == "" {
		return false, errors.New("validation_error: user_id and manga_id are required")
	}

	var exists bool
	err := s.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_notifications WHERE user_id = ? AND manga_id = ?)`,
		userID, mangaID,
	).Scan(&exists)
	if err != nil {
		log.Printf("Error checking notification subscription: %v", err)
		return false, errors.New("database_error: failed to check notification subscription")
	}
	return exists, nil
}

// ReRegisterNotification is a best-effort UDP registration, useful when the UDP server was down.
func (s *Service) ReRegisterNotification(userID, mangaID string) {
	if userID == "" || mangaID == "" {
		return
	}
	if _, err := RegisterForUDPNotifications(UDPRegisterOptions{
		ServerAddr: "localhost:9091",
		UserID:     userID,
		MangaIDs:   []string{mangaID},
	}); err != nil {
		log.Printf("UDP re-register failed for user %s manga %s: %v", userID, mangaID, err)
	}
}

// UnsubscribeFromMangaNotifications removes a user's subscription for a manga.
func (s *Service) UnsubscribeFromMangaNotifications(userID, mangaID string) error {
	if userID == "" || mangaID == "" {
		return errors.New("validation_error: user_id and manga_id are required")
	}

	_, err := s.DB.Exec(
		`DELETE FROM user_notifications WHERE user_id = ? AND manga_id = ?`,
		userID, mangaID,
	)
	if err != nil {
		log.Printf("Error deleting notification subscription: %v", err)
		return errors.New("database_error: failed to delete notification subscription")
	}

	// Send unregister to UDP server (best effort)
	if udpErr := UnregisterForUDPNotifications(UDPUnregisterOptions{
		ServerAddr: "localhost:9091",
		UserID:     userID,
		MangaIDs:   []string{mangaID},
	}); udpErr != nil {
		log.Printf("UDP unregister failed for user %s manga %s: %v", userID, mangaID, udpErr)
	}

	return nil
}

// AddToLibraryRequest represents a request to add manga to user's library.
type AddToLibraryRequest struct {
	MangaID        string
	CurrentChapter int
	Status         string
}

// AddToLibraryResult represents the result of adding manga to library (UC-005).
type AddToLibraryResult struct {
	Status string // "newly_added" or "already_exists"
}

// AddToLibrary adds a manga to user's library or updates if it exists (UC-005).
// Returns AddToLibraryResult indicating whether manga was newly added or already existed.
func (s *Service) AddToLibrary(userID string, req AddToLibraryRequest) (*AddToLibraryResult, error) {
	// UC-005 Alternative Flow A1: Check if manga already exists in user's library
	var alreadyInLibrary bool
	err := s.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM user_progress WHERE user_id = ? AND manga_id = ?)",
		userID, req.MangaID,
	).Scan(&alreadyInLibrary)
	if err != nil {
		log.Printf("Error checking library entry: %v", err)
		return nil, errors.New("database_error: failed to check library entry")
	}

	// UC-005 Main Success Scenario: Create user_progress record
	// Works for any manga_id (local DB manga or MangaDex manga)
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
		return nil, errors.New("database_error: failed to save library entry")
	}

	// Return status for UC-005
	result := &AddToLibraryResult{}
	if alreadyInLibrary {
		result.Status = "already_exists"
	} else {
		result.Status = "newly_added"
	}

	return result, nil
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

// UpdateProgressResult represents the result of updating progress (UC-006).
type UpdateProgressResult struct {
	BroadcastSent bool   // Whether TCP broadcast was successfully sent
	BroadcastError string // Error message if broadcast failed (for queuing)
}

// UpdateProgress updates user's reading progress for a manga (UC-006).
// Precondition: Manga must be in user's library.
// Validates chapter number against manga metadata.
// Updates progress with timestamp and broadcasts via TCP.
func (s *Service) UpdateProgress(userID string, req UpdateProgressRequest) (*UpdateProgressResult, error) {
	// UC-006 Precondition: Check if manga is in user's library
	var existsInLibrary bool
	err := s.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM user_progress WHERE user_id = ? AND manga_id = ?)",
		userID, req.MangaID,
	).Scan(&existsInLibrary)
	if err != nil {
		log.Printf("Error checking library entry: %v", err)
		return nil, errors.New("database_error: failed to check library entry")
	}
	if !existsInLibrary {
		return nil, errors.New("validation_error: manga is not in user's library")
	}

	// UC-006 Main Success Scenario Step 2: Validate chapter number against manga metadata
	if s.MangaSvc != nil {
		manga, err := s.MangaSvc.GetMangaByID(req.MangaID)
		if err == nil && manga != nil {
			// Validate chapter number
			if req.CurrentChapter < 0 {
				return nil, errors.New("validation_error: chapter number cannot be negative")
			}
			if manga.TotalChapters > 0 && req.CurrentChapter > manga.TotalChapters {
				return nil, errors.New("validation_error: chapter number exceeds total chapters")
			}
		}
		// If manga not found in local DB (e.g., MangaDex manga), allow update
		// The frontend dropdown already limits selection to valid range
	}

	// UC-006 Main Success Scenario Step 3: Update user_progress record with timestamp
	result, err := s.DB.Exec(
		`UPDATE user_progress 
		SET current_chapter = ?, 
			status = COALESCE(NULLIF(?, ''), status), 
			updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = ? AND manga_id = ?`,
		req.CurrentChapter, req.Status, userID, req.MangaID,
	)
	if err != nil {
		log.Printf("Error updating progress: %v", err)
		return nil, errors.New("database_error: failed to update progress")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
	} else if rowsAffected == 0 {
		return nil, errors.New("validation_error: manga is not in user's library")
	}

	// UC-006 Main Success Scenario Step 4: Trigger TCP broadcast to connected clients
	update := ProgressUpdate{
		UserID:    userID,
		MangaID:   req.MangaID,
		Chapter:   req.CurrentChapter,
		Timestamp: time.Now().Unix(),
	}

	broadcastResult := &UpdateProgressResult{
		BroadcastSent: false,
		BroadcastError: "",
	}

	err = BroadcastProgress(s.TCPAddr, update)
	if err != nil {
		// UC-006 Alternative Flow A2: TCP server unavailable - update locally, queue broadcast
		log.Printf("TCP broadcast failed (will be queued/retried): %v", err)
		broadcastResult.BroadcastError = err.Error()
		// Still return success since local update succeeded
	} else {
		broadcastResult.BroadcastSent = true
	}

	return broadcastResult, nil
}
