package tcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

// AuthMessage is sent by the TCP client immediately after connecting.
// It authenticates the client and registers it for progress updates.
type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token,omitempty"` // Optional JWT token (for future use)
	// For this project we accept a plain user ID for simplicity.
	UserID string `json:"user_id,omitempty"`
}

// AuthResponse is sent by the server to confirm or reject registration.
type AuthResponse struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ProgressUpdate represents a progress update to broadcast via TCP.
type ProgressUpdate struct {
	Type      string `json:"type"` // always "progress"
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

// Server is the TCP sync server implementation for UC-007.
type Server struct {
	Port       string
	MaxClients int

	mu          sync.RWMutex
	connections map[string]net.Conn
	Broadcast   chan ProgressUpdate
}
// Commit
// NewServer creates a new TCP sync server with sane defaults.
func NewServer(port string, maxClients int) *Server {
	if port == "" {
		port = "9090"
	}
	if maxClients <= 0 {
		maxClients = 100
	}

	return &Server{
		Port:       port,
		MaxClients: maxClients,
		connections: make(map[string]net.Conn),
		Broadcast:   make(chan ProgressUpdate, 64),
	}
}

// FromEnv constructs a Server using environment variables:
// - MANGAHUB_TCP_PORT
// - MANGAHUB_TCP_MAX_CLIENTS
func FromEnv() *Server {
	port := os.Getenv("MANGAHUB_TCP_PORT")
	maxClients := 0
	if v := os.Getenv("MANGAHUB_TCP_MAX_CLIENTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxClients = n
		}
	}
	return NewServer(port, maxClients)
}

// Start begins listening on the configured port and handling connections.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		return fmt.Errorf("tcp listen error: %w", err)
	}
	log.Println("TCP progress sync listening on :" + s.Port)

	go s.broadcastLoop()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("tcp accept error:", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn implements UC-007 connection and registration flow.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// 1) Read authentication message from client
	line, err := reader.ReadBytes('\n')
	if err != nil {
		log.Println("failed to read auth message:", err)
		return
	}

	var authMsg AuthMessage
	if err := json.Unmarshal(line, &authMsg); err != nil {
		log.Println("invalid auth message:", err)
		_ = sendAuthResponse(conn, "error", "invalid_auth_message")
		return
	}

	if authMsg.Type != "auth" {
		_ = sendAuthResponse(conn, "error", "expected_auth_message")
		return
	}

	// Simple validation for this project: require non-empty user ID.
	userID := authMsg.UserID
	if userID == "" {
		_ = sendAuthResponse(conn, "error", "missing_user_id")
		return
	}

	// A2: Server at capacity.
	if err := s.registerClient(userID, conn); err != nil {
		if errors.Is(err, ErrServerAtCapacity) {
			_ = sendAuthResponse(conn, "error", "server_at_capacity")
		} else {
			_ = sendAuthResponse(conn, "error", "registration_failed")
		}
		return
	}

	log.Printf("TCP: user %s connected\n", userID)

	// Acknowledge successful registration to client.
	if err := sendAuthResponse(conn, "ok", ""); err != nil {
		log.Println("failed to send auth_ok:", err)
		s.unregisterClient(userID)
		return
	}

	// 2) Main loop: receive progress updates from this client.
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		var upd ProgressUpdate
		if err := json.Unmarshal(line, &upd); err != nil {
			log.Printf("TCP: failed to unmarshal progress from %s: %v\n", userID, err)
			continue
		}
		if upd.Type != "progress" {
			// Ignore unknown message types for now.
			continue
		}

		// Ensure the user ID on the update matches the authenticated user.
		if upd.UserID == "" {
			upd.UserID = userID
		}

		log.Printf("TCP: received progress from %s: manga=%s, chapter=%d\n",
			userID, upd.MangaID, upd.Chapter)

		s.Broadcast <- upd
	}

	s.unregisterClient(userID)
	log.Printf("TCP: user %s disconnected\n", userID)
}

// ErrServerAtCapacity indicates the server cannot accept more clients.
var ErrServerAtCapacity = errors.New("server_at_capacity")

func (s *Server) registerClient(userID string, conn net.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.connections) >= s.MaxClients {
		return ErrServerAtCapacity
	}

	// If a connection with this user already exists, close it and replace.
	if old, ok := s.connections[userID]; ok {
		_ = old.Close()
	}
	s.connections[userID] = conn
	return nil
}

func (s *Server) unregisterClient(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn, ok := s.connections[userID]; ok {
		_ = conn.Close()
		delete(s.connections, userID)
	}
}

// broadcastLoop sends progress updates to all connected and registered clients.
func (s *Server) broadcastLoop() {
	for upd := range s.Broadcast {
		data, err := json.Marshal(upd)
		if err != nil {
			log.Printf("TCP: failed to marshal broadcast update: %v\n", err)
			continue
		}
		data = append(data, '\n')

		s.mu.RLock()
		clientCount := len(s.connections)
		for userID, conn := range s.connections {
			if _, err := conn.Write(data); err != nil {
				log.Printf("TCP: failed to send to %s: %v\n", userID, err)
			}
		}
		s.mu.RUnlock()

		if clientCount > 0 {
			log.Printf("TCP: broadcasted progress update: manga=%s, chapter=%d to %d client(s)\n",
				upd.MangaID, upd.Chapter, clientCount)
		} else {
			log.Printf("TCP: broadcasted progress update: manga=%s, chapter=%d (no clients)\n",
				upd.MangaID, upd.Chapter)
		}
	}
}

func sendAuthResponse(conn net.Conn, status, errMsg string) error {
	resp := AuthResponse{
		Type:   "auth_response",
		Status: status,
		Error:  errMsg,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}


