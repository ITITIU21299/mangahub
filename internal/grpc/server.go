package grpc

import (
	"log"
	"net"

	"mangahub/internal/manga"
	"mangahub/internal/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "mangahub/proto"
)

// Server wraps the gRPC server and services
type Server struct {
	grpcServer   *grpc.Server
	mangaService *manga.Service
	userService  *user.Service
}

// NewServer creates a new gRPC server with the provided services
func NewServer(mangaSvc *manga.Service, userSvc *user.Service) *Server {
	grpcServer := grpc.NewServer()

	// Create the manga service implementation
	mangaSvcImpl := NewMangaServiceServer(mangaSvc, userSvc)

	// Register the service using generated protobuf code
	pb.RegisterMangaServiceServer(grpcServer, mangaSvcImpl)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	return &Server{
		grpcServer:   grpcServer,
		mangaService: mangaSvc,
		userService:  userSvc,
	}
}

// Start starts the gRPC server on the specified address
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Printf("gRPC server listening on %s", addr)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
