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
	Type   string `json:"type"`             // "auth"
	UserID string `json:"user_id,omitempty"` // user identifier
	Token  string `json:"token,omitempty"` // optional JWT token (reserved)
}

// AuthResponse is sent by the server to confirm or reject registration.
type AuthResponse struct {
	Type   string `json:"type"`   // "auth_response"
	Status string `json:"status"` // "ok" or "error"
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

// Server is the TCP sync server implementation for UC-007/UC-008.
type Server struct {
	Port       string
	MaxClients int

	mu          sync.RWMutex
	connections map[string]map[net.Conn]struct{} // userID -> set of conns
	Broadcast   chan ProgressUpdate
}

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
		connections: make(map[string]map[net.Conn]struct{}),
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
		s.unregisterClient(userID, conn)
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

	s.unregisterClient(userID, conn)
	log.Printf("TCP: user %s disconnected\n", userID)
}

// ErrServerAtCapacity indicates the server cannot accept more clients.
var ErrServerAtCapacity = errors.New("server_at_capacity")

func (s *Server) registerClient(userID string, conn net.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Count total connections
	total := 0
	for _, conns := range s.connections {
		total += len(conns)
	}
	if total >= s.MaxClients {
		return ErrServerAtCapacity
	}

	if s.connections[userID] == nil {
		s.connections[userID] = make(map[net.Conn]struct{})
	}
	s.connections[userID][conn] = struct{}{}
	return nil
}

func (s *Server) unregisterClient(userID string, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conns, ok := s.connections[userID]; ok {
		if _, exists := conns[conn]; exists {
			delete(conns, conn)
			_ = conn.Close()
		}
		if len(conns) == 0 {
			delete(s.connections, userID)
		}
	}
}

// broadcastLoop sends progress updates to all relevant clients (same user).
func (s *Server) broadcastLoop() {
	for upd := range s.Broadcast {
		data, err := json.Marshal(upd)
		if err != nil {
			log.Printf("TCP: failed to marshal broadcast update: %v\n", err)
			continue
		}
		data = append(data, '\n')

		s.mu.RLock()
		targetUser := upd.UserID
		conns := s.connections[targetUser]
		clientCount := len(conns)
		for conn := range conns {
			if _, err := conn.Write(data); err != nil {
				log.Printf("TCP: failed to send to %s: %v\n", targetUser, err)
			}
		}
		s.mu.RUnlock()

		if clientCount > 0 {
			log.Printf("TCP: broadcasted progress update to user %s: manga=%s, chapter=%d to %d connection(s)\n",
				targetUser, upd.MangaID, upd.Chapter, clientCount)
		} else {
			log.Printf("TCP: broadcasted progress update for user %s: manga=%s, chapter=%d (no active connections)\n",
				targetUser, upd.MangaID, upd.Chapter)
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


