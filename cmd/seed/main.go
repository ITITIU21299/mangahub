package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"mangahub/internal/database"
)

type MangaData struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
}

func main() {
	// Open database - use same path as servers
	dbPath := os.Getenv("MANGAHUB_DB_PATH")
	if dbPath == "" {
		dbPath = "mangahub.db" // In mangahub root directory
	}
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Read JSON file
	jsonData, err := os.ReadFile("data/manga.json")
	if err != nil {
		log.Fatal("Failed to read manga.json:", err)
	}

	// Parse JSON
	var mangas []MangaData
	if err := json.Unmarshal(jsonData, &mangas); err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	// Prepare insert statement
	stmt, err := db.Prepare(`
		INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal("Failed to prepare statement:", err)
	}
	defer stmt.Close()

	// Insert each manga
	inserted := 0
	for _, manga := range mangas {
		genresJSON, err := json.Marshal(manga.Genres)
		if err != nil {
			log.Printf("Warning: Failed to marshal genres for %s: %v", manga.Title, err)
			continue
		}

		_, err = stmt.Exec(
			manga.ID,
			manga.Title,
			manga.Author,
			string(genresJSON),
			manga.Status,
			manga.TotalChapters,
			manga.Description,
			manga.CoverURL,
		)
		if err != nil {
			log.Printf("Warning: Failed to insert %s: %v", manga.Title, err)
			continue
		}
		inserted++
	}

	fmt.Printf("Successfully inserted %d manga entries into the database.\n", inserted)
}
