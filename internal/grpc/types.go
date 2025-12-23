package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error helpers for gRPC status codes

func ErrInvalidRequest(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func ErrNotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

func ErrInternal(msg string) error {
	return status.Error(codes.Internal, msg)
}

// GetMangaRequest, GetMangaResponse, SearchMangaRequest, etc. are defined here
// to match the proto structure without requiring protoc generation

type GetMangaRequest struct {
	MangaId string
}

type GetMangaResponse struct {
	Manga *Manga
}

type SearchMangaRequest struct {
	Query  string
	Genre  string
	Status string
	Page   int32
	Limit  int32
}

type SearchMangaResponse struct {
	Data       []*Manga
	Total      int32
	Page       int32
	Limit      int32
	TotalPages int32
}

type UpdateProgressRequest struct {
	UserId         string
	MangaId        string
	CurrentChapter int32
	Status         string
}

type UpdateProgressResponse struct {
	Success       bool
	Message       string
	BroadcastSent bool
}

type Manga struct {
	Id            string
	Title         string
	Author        string
	Genres        []string
	Status        string
	TotalChapters int32
	Description   string
	CoverUrl      string
}

// MangaServiceServer interface defines the gRPC service methods
type MangaServiceServer interface {
	GetManga(ctx context.Context, req *GetMangaRequest) (*GetMangaResponse, error)
	SearchManga(ctx context.Context, req *SearchMangaRequest) (*SearchMangaResponse, error)
	UpdateProgress(ctx context.Context, req *UpdateProgressRequest) (*UpdateProgressResponse, error)
}

// UnimplementedMangaServiceServer provides default implementations for forward compatibility
type UnimplementedMangaServiceServer struct{}

func (UnimplementedMangaServiceServer) GetManga(ctx context.Context, req *GetMangaRequest) (*GetMangaResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetManga not implemented")
}

func (UnimplementedMangaServiceServer) SearchManga(ctx context.Context, req *SearchMangaRequest) (*SearchMangaResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method SearchManga not implemented")
}

func (UnimplementedMangaServiceServer) UpdateProgress(ctx context.Context, req *UpdateProgressRequest) (*UpdateProgressResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateProgress not implemented")
}

