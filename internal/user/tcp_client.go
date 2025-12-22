package user

import (
	"encoding/json"
	"log"
	"net"
	"time"
)

// ProgressUpdate represents a progress update to broadcast via TCP.
type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

// BroadcastProgress sends a progress update to the TCP server.
// The TCP server protocol expects:
// 1. First line: user ID (as string ending with '\n')
// 2. Subsequent lines: JSON progress updates (ending with '\n')
// Returns error if TCP server is unavailable (for queuing/retry logic).
func BroadcastProgress(tcpAddr string, update ProgressUpdate) error {
	if tcpAddr == "" {
		tcpAddr = "localhost:9090" // Default TCP server port
	}

	conn, err := net.DialTimeout("tcp", tcpAddr, 2*time.Second)
	if err != nil {
		log.Printf("TCP server unavailable at %s: %v", tcpAddr, err)
		return err // Return error so caller can queue/retry
	}
	defer conn.Close()

	// Step 1: Send user ID first (TCP server protocol requirement)
	userIDLine := update.UserID + "\n"
	if _, err := conn.Write([]byte(userIDLine)); err != nil {
		log.Printf("Failed to send user ID to TCP server: %v", err)
		return err
	}

	// Step 2: Send JSON progress update
	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("Failed to marshal progress update: %v", err)
		return err
	}

	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		log.Printf("Failed to send progress update to TCP server: %v", err)
		return err
	}

	log.Printf("Progress update broadcasted: user=%s, manga=%s, chapter=%d", update.UserID, update.MangaID, update.Chapter)
	return nil
}

