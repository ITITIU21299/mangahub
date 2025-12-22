package main

import (
	"log"
	"os"
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

	log.Println("HTTP API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
