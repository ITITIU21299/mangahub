package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "mangahub/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:9092", "gRPC server address")
	action := flag.String("action", "get", "Action: get, search, or update")
	mangaID := flag.String("manga-id", "", "Manga ID (for get/update)")
	userID := flag.String("user-id", "", "User ID (for update)")
	chapter := flag.Int("chapter", 0, "Current chapter (for update)")
	query := flag.String("query", "", "Search query")
	genre := flag.String("genre", "", "Genre filter")
	page := flag.Int("page", 1, "Page number")
	flag.Parse()

	// Connect to gRPC server
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMangaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *action {
	case "get":
		if *mangaID == "" {
			log.Fatal("manga-id is required for get action")
		}
		resp, err := client.GetManga(ctx, &pb.GetMangaRequest{MangaId: *mangaID})
		if err != nil {
			log.Fatalf("GetManga failed: %v", err)
		}
		fmt.Printf("✅ Retrieved manga via gRPC:\n")
		fmt.Printf("   ID: %s\n", resp.Manga.Id)
		fmt.Printf("   Title: %s\n", resp.Manga.Title)
		fmt.Printf("   Author: %s\n", resp.Manga.Author)
		fmt.Printf("   Status: %s\n", resp.Manga.Status)
		fmt.Printf("   Chapters: %d\n", resp.Manga.TotalChapters)

	case "search":
		req := &pb.SearchMangaRequest{
			Query:  *query,
			Genre:  *genre,
			Page:   int32(*page),
			Limit:  20,
		}
		resp, err := client.SearchManga(ctx, req)
		if err != nil {
			log.Fatalf("SearchManga failed: %v", err)
		}
		fmt.Printf("✅ Search results via gRPC:\n")
		fmt.Printf("   Total: %d\n", resp.Total)
		fmt.Printf("   Page: %d/%d\n", resp.Page, resp.TotalPages)
		fmt.Printf("   Results: %d\n\n", len(resp.Data))
		for i, m := range resp.Data {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(resp.Data)-5)
				break
			}
			fmt.Printf("   %d. %s by %s\n", i+1, m.Title, m.Author)
		}

	case "update":
		if *userID == "" || *mangaID == "" {
			log.Fatal("user-id and manga-id are required for update action")
		}
		req := &pb.UpdateProgressRequest{
			UserId:         *userID,
			MangaId:        *mangaID,
			CurrentChapter: int32(*chapter),
			Status:         "reading",
		}
		resp, err := client.UpdateProgress(ctx, req)
		if err != nil {
			log.Fatalf("UpdateProgress failed: %v", err)
		}
		fmt.Printf("✅ Progress updated via gRPC:\n")
		fmt.Printf("   Success: %v\n", resp.Success)
		fmt.Printf("   Message: %s\n", resp.Message)
		fmt.Printf("   Broadcast sent: %v\n", resp.BroadcastSent)

	default:
		log.Fatalf("Unknown action: %s (use: get, search, or update)", *action)
	}
}

