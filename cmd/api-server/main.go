package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"mangahub/internal/auth"
	"mangahub/internal/database"
	"mangahub/internal/manga"
	"mangahub/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	jwtSecret := []byte(os.Getenv("MANGAHUB_JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-me")
	}

	// Use same database path as seed script
	dbPath := os.Getenv("MANGAHUB_DB_PATH")
	if dbPath == "" {
		// Try to find mangahub.db in project root
		if _, err := os.Stat("../mangahub.db"); err == nil {
			dbPath = "../mangahub.db"
		} else {
			dbPath = "mangahub.db"
		}
	}
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal("init db:", err)
	}
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS configuration - allow requests from frontend
	// In production, specify exact origins. For development, allow all origins.
	corsOrigins := os.Getenv("MANGAHUB_CORS_ORIGINS")
	var allowedOrigins []string
	if corsOrigins != "" {
		// Parse comma-separated origins from env
		allowedOrigins = []string{}
		for _, origin := range strings.Split(corsOrigins, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
		}
	} else {
		// Default: allow localhost and common development origins
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://25.17.216.66:3000", // Add your IP if needed
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public proxy endpoint for MangaDex cover images to avoid hotlinking placeholders
	r.GET("/proxy/cover", func(c *gin.Context) {
		url := c.Query("url")
		if url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing url parameter"})
			return
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("proxy cover error: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch image"})
			return
		}
		defer resp.Body.Close()

		// Pass through content type
		c.Header("Content-Type", resp.Header.Get("Content-Type"))
		c.Status(resp.StatusCode)
		if _, err := io.Copy(c.Writer, resp.Body); err != nil {
			log.Printf("proxy cover write error: %v", err)
		}
	})

	// Initialize services
	authSvc := auth.NewService(db)
	mangaSvc := manga.NewService(db)
	userSvc := user.NewService(db)
	userSvc.SetMangaService(mangaSvc) // Set manga service for UC-006 validation

	// Register auth routes and get middleware
	authMiddleware := auth.RegisterRoutes(r, authSvc, jwtSecret)

	// Register protected routes
	auth := r.Group("/")
	auth.Use(authMiddleware)
	{
		manga.RegisterRoutes(auth, mangaSvc)
		user.RegisterRoutes(auth, userSvc)
	}

	// Bind to all interfaces (0.0.0.0) to allow network access
	bindAddr := os.Getenv("MANGAHUB_API_ADDR")
	if bindAddr == "" {
		bindAddr = "0.0.0.0:8080" // Listen on all interfaces
	}
	log.Printf("HTTP API listening on %s", bindAddr)
	if err := r.Run(bindAddr); err != nil {
		log.Fatal(err)
	}
}
