package main

// This file intentionally keeps the gRPC server minimal.
// You can add .proto definitions under a proto/ directory and generate
// Go code with protoc, then implement MangaService here.

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// TODO: register your generated MangaService server implementation here.

	fmt.Println("gRPC server listening on :9092")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}


