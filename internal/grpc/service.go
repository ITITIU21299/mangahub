package grpc

import (
	"context"
	"log"

	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/models"
	pb "mangahub/proto"
)

// ServiceServer implements the gRPC MangaService
type ServiceServer struct {
	pb.UnimplementedMangaServiceServer // For forward compatibility
	mangaService                       *manga.Service
	userService                        *user.Service
}

// NewMangaServiceServer creates a new gRPC manga service server
func NewMangaServiceServer(mangaSvc *manga.Service, userSvc *user.Service) *ServiceServer {
	return &ServiceServer{
		mangaService: mangaSvc,
		userService:  userSvc,
	}
}

// GetManga implements UC-014: Retrieve Manga via gRPC
func (s *ServiceServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.GetMangaResponse, error) {
	if req.MangaId == "" {
		return nil, ErrInvalidRequest("manga_id is required")
	}

	// Query database for manga information
	m, err := s.mangaService.GetMangaByID(req.MangaId)
	if err != nil {
		if err.Error() == "not_found" {
			return nil, ErrNotFound("manga not found")
		}
		log.Printf("Error querying manga: %v", err)
		return nil, ErrInternal("failed to query manga")
	}

	// Construct protobuf response message
	return &pb.GetMangaResponse{
		Manga: toProtoManga(m),
	}, nil
}

// SearchManga implements UC-015: Search Manga via gRPC
func (s *ServiceServer) SearchManga(ctx context.Context, req *pb.SearchMangaRequest) (*pb.SearchMangaResponse, error) {
	// Process search parameters
	params := manga.SearchParams{
		Query:  req.Query,
		Genre:  req.Genre,
		Status: req.Status,
		Page:   int(req.Page),
		Limit:  int(req.Limit),
	}

	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	// Execute database query with filters
	result, err := s.mangaService.SearchManga(params)
	if err != nil {
		log.Printf("Error searching manga: %v", err)
		return nil, ErrInternal("failed to search manga")
	}

	// Construct response with result list
	protoMangas := make([]*pb.Manga, 0, len(result.Data))
	for _, m := range result.Data {
		protoMangas = append(protoMangas, toProtoManga(&m))
	}

	// Return paginated results
	return &pb.SearchMangaResponse{
		Data:       protoMangas,
		Total:      int32(result.Total),
		Page:       int32(result.Page),
		Limit:      int32(result.Limit),
		TotalPages: int32(result.TotalPages),
	}, nil
}

// UpdateProgress implements UC-016: Update Progress via gRPC
func (s *ServiceServer) UpdateProgress(ctx context.Context, req *pb.UpdateProgressRequest) (*pb.UpdateProgressResponse, error) {
	// Validate request parameters
	if req.UserId == "" {
		return nil, ErrInvalidRequest("user_id is required")
	}
	if req.MangaId == "" {
		return nil, ErrInvalidRequest("manga_id is required")
	}
	if req.CurrentChapter < 0 {
		return nil, ErrInvalidRequest("current_chapter cannot be negative")
	}

	// Update user_progress table
	updateReq := user.UpdateProgressRequest{
		MangaID:        req.MangaId,
		CurrentChapter: int(req.CurrentChapter),
		Status:         req.Status,
	}

	result, err := s.userService.UpdateProgress(req.UserId, updateReq)
	if err != nil {
		errorMsg := err.Error()
		if errorMsg == "validation_error: manga is not in user's library" {
			return nil, ErrInvalidRequest("manga is not in user's library")
		}
		if errorMsg == "validation_error: chapter number cannot be negative" ||
			errorMsg == "validation_error: chapter number exceeds total chapters" {
			return nil, ErrInvalidRequest(errorMsg)
		}
		log.Printf("Error updating progress: %v", err)
		return nil, ErrInternal("failed to update progress")
	}

	// Trigger TCP broadcast for real-time sync (already done in UpdateProgress)
	// Return success confirmation
	return &pb.UpdateProgressResponse{
		Success:       true,
		Message:       "Progress updated successfully",
		BroadcastSent: result.BroadcastSent,
	}, nil
}

// Helper function to convert models.Manga to proto Manga
func toProtoManga(m *models.Manga) *pb.Manga {
	return &pb.Manga{
		Id:            m.ID,
		Title:         m.Title,
		Author:        m.Author,
		Genres:        m.Genres,
		Status:        m.Status,
		TotalChapters: int32(m.TotalChapters),
		Description:   m.Description,
		CoverUrl:      m.CoverURL,
	}
}
