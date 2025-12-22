package main

import (
	"log"

	"mangahub/internal/udp"
)

// cmd/udp-server: thin wrapper; all UC-009/UC-010 logic lives in internal/udp.
func main() {
	srv := udp.FromEnv()
	if err := srv.Start(); err != nil {
		log.Println("udp server error:", err)
	}
}

