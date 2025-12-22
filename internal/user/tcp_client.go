package user

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"
)

// ProgressUpdate represents a progress update to broadcast via TCP.
// This matches the wire format expected by internal/tcp.Server.
type ProgressUpdate struct {
	Type      string `json:"type"` // always "progress"
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

// authRequest is sent immediately after connecting to authenticate and register.
type authRequest struct {
	Type   string `json:"type"`            // "auth"
	UserID string `json:"user_id"`         // required
	Token  string `json:"token,omitempty"` // reserved for future JWT-based auth
}

// authResponse is returned by the TCP server after auth.
type authResponse struct {
	Type   string `json:"type"`   // "auth_response"
	Status string `json:"status"` // "ok" or "error"
	Error  string `json:"error,omitempty"`
}

// BroadcastProgress sends a progress update to the TCP server following UC-007:
// 1) Client initiates TCP connection.
// 2) Client sends authentication message with user ID.
// 3) Server validates and registers connection.
// 4) Server confirms registration.
// 5) Client sends progress update for broadcast.
//
// Returns an error if the TCP server is unavailable or authentication fails.
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

	reader := bufio.NewReader(conn)

	// Step 1: Send auth message.
	auth := authRequest{
		Type:   "auth",
		UserID: update.UserID,
	}
	if auth.UserID == "" {
		return errors.New("missing user ID for TCP auth")
	}

	if err := writeJSONLine(conn, auth); err != nil {
		log.Printf("Failed to send TCP auth message: %v", err)
		return err
	}

	// Step 2: Read auth response.
	var resp authResponse
	if err := readJSONLine(reader, &resp); err != nil {
		log.Printf("Failed to read TCP auth response: %v", err)
		return err
	}
	if resp.Type != "auth_response" {
		return errors.New("unexpected TCP auth response type")
	}
	if resp.Status != "ok" {
		if resp.Error == "" {
			resp.Error = "authentication_failed"
		}
		log.Printf("TCP auth failed: %s", resp.Error)
		return errors.New(resp.Error)
	}

	// Step 3: Send progress update.
	update.Type = "progress"
	if err := writeJSONLine(conn, update); err != nil {
		log.Printf("Failed to send progress update to TCP server: %v", err)
		return err
	}

	log.Printf("Progress update broadcast requested: user=%s, manga=%s, chapter=%d", update.UserID, update.MangaID, update.Chapter)
	return nil
}

// writeJSONLine marshals v to JSON and writes it followed by '\n'.
func writeJSONLine(conn net.Conn, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

// readJSONLine reads a '\n'-terminated line and unmarshals JSON into v.
func readJSONLine(r *bufio.Reader, v interface{}) error {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal(line, v)
}

