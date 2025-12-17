package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"mangahub/internal/auth"
	"mangahub/internal/database"
	"mangahub/pkg/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Router    *gin.Engine
	Database  *sql.DB
}

type libraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status" binding:"required"`
}

type progressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"gte=0"`
	Status         string `json:"status"`
}

func main() {
	jwtSecret := []byte(os.Getenv("MANGAHUB_JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-me")
	}

	db, err := database.Init("mangahub.db")
	if err != nil {
		log.Fatal("init db:", err)
	}
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Allow the Next.js dev server (http://localhost:3000) to call this API.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	authSvc := auth.NewService(db)
	authMiddleware := auth.RegisterRoutes(r, authSvc, jwtSecret)

	s := &APIServer{
		Router:    r,
		Database:  db,
	}
	s.registerRoutes(authMiddleware)

	log.Println("HTTP API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func (s *APIServer) registerRoutes(authMiddleware gin.HandlerFunc) {
	auth := s.Router.Group("/")
	auth.Use(authMiddleware)

	auth.GET("/manga", s.handleListManga)
	auth.GET("/manga/:id", s.handleGetManga)
	auth.POST("/users/library", s.handleAddToLibrary)
	auth.GET("/users/library", s.handleGetLibrary)
	auth.PUT("/users/progress", s.handleUpdateProgress)
}

func (s *APIServer) handleListManga(c *gin.Context) {
	rows, err := s.Database.Query(`SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query manga"})
		return
	}
	defer rows.Close()

	var result []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresJSON string
		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL); err != nil {
			continue
		}
		// For now keep genres as raw text; you can unmarshal JSON later.
		m.Genres = []string{}
		result = append(result, m)
	}
	c.JSON(http.StatusOK, result)
}

func (s *APIServer) handleGetManga(c *gin.Context) {
	id := c.Param("id")
	var m models.Manga
	var genresJSON string
	err := s.Database.QueryRow(
		`SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query manga"})
		return
	}
	m.Genres = []string{}
	c.JSON(http.StatusOK, m)
}

func (s *APIServer) handleAddToLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	var req libraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.Database.Exec(
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status) 
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, manga_id) DO UPDATE SET current_chapter = excluded.current_chapter, status = excluded.status, updated_at = CURRENT_TIMESTAMP`,
		userID, req.MangaID, req.CurrentChapter, req.Status,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save library entry"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func (s *APIServer) handleGetLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := s.Database.Query(
		`SELECT manga_id, current_chapter, status, updated_at FROM user_progress WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query library"})
		return
	}
	defer rows.Close()

	var items []models.UserProgress
	for rows.Next() {
		var p models.UserProgress
		p.UserID = userID
		if err := rows.Scan(&p.MangaID, &p.CurrentChapter, &p.Status, &p.UpdatedAt); err != nil {
			continue
		}
		items = append(items, p)
	}
	c.JSON(http.StatusOK, items)
}

func (s *APIServer) handleUpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	var req progressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := s.Database.Exec(
		`UPDATE user_progress SET current_chapter = ?, status = COALESCE(NULLIF(?, ''), status), updated_at = CURRENT_TIMESTAMP 
		 WHERE user_id = ? AND manga_id = ?`,
		req.CurrentChapter, req.Status, userID, req.MangaID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
