package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"mangahub/internal/database"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	Router    *gin.Engine
	Database  *sql.DB
	JWTSecret []byte
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	s := &APIServer{
		Router:    r,
		Database:  db,
		JWTSecret: jwtSecret,
	}
	s.registerRoutes()

	log.Println("HTTP API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func (s *APIServer) registerRoutes() {
	s.Router.POST("/auth/register", s.handleRegister)
	s.Router.POST("/auth/login", s.handleLogin)

	auth := s.Router.Group("/")
	auth.Use(s.jwtMiddleware)

	auth.GET("/manga", s.handleListManga)
	auth.GET("/manga/:id", s.handleGetManga)
	auth.POST("/users/library", s.handleAddToLibrary)
	auth.GET("/users/library", s.handleGetLibrary)
	auth.PUT("/users/progress", s.handleUpdateProgress)
}

func (s *APIServer) handleRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	_, err = s.Database.Exec(
		`INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
		"user_"+req.Username, req.Username, string(hash),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

func (s *APIServer) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id, hash string
	if err := s.Database.QueryRow(
		`SELECT id, password_hash FROM users WHERE username = ?`, req.Username,
	).Scan(&id, &hash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id,
		"usr": req.Username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString(s.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed})
}

func (s *APIServer) jwtMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	raw := authHeader[7:]
	token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	userID, _ := claims["sub"].(string)
	c.Set("user_id", userID)
	c.Next()
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
