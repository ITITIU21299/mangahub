package main

import (
	"log"
	"os"

	"mangahub/internal/database"
	"mangahub/internal/grpc"
	"mangahub/internal/manga"
	"mangahub/internal/user"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("MANGAHUB_DB_PATH")
	if dbPath == "" {
		dbPath = "mangahub.db"
	}

	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal("init db:", err)
	}
	defer db.Close()

	// Initialize services
	mangaSvc := manga.NewService(db)
	userSvc := user.NewService(db)
	userSvc.SetMangaService(mangaSvc) // Set manga service for validation

	// Create and start gRPC server
	grpcAddr := os.Getenv("MANGAHUB_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":9092"
	}

	server := grpc.NewServer(mangaSvc, userSvc)
	log.Fatal(server.Start(grpcAddr))
}


