package main

import (
	"log"

	"mangahub/internal/tcp"
)

// cmd/tcp-server: thin wrapper; all UC-007/UC-008 logic lives in internal/tcp.
func main() {
	srv := tcp.FromEnv()
	if err := srv.Start(); err != nil {
		log.Println("tcp server error:", err)
	}
}

