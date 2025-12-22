package udp

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// RegisterMessage is sent by UDP clients to register for notifications (UC-009).
type RegisterMessage struct {
	Type        string   `json:"type"`                  // "register"
	UserID      string   `json:"user_id,omitempty"`     // user identifier
	MangaIDs    []string `json:"manga_ids,omitempty"`   // optional list of manga IDs
	Preferences []string `json:"preferences,omitempty"` // optional tags/genres
	ClientLabel string   `json:"client_label,omitempty"`
}

// UnregisterMessage is sent by UDP clients to unregister notifications.
type UnregisterMessage struct {
	Type     string   `json:"type"`              // "unregister"
	UserID   string   `json:"user_id,omitempty"` // user identifier
	MangaIDs []string `json:"manga_ids,omitempty"`
}

// RegisterResponse confirms registration.
type RegisterResponse struct {
	Type    string `json:"type"`              // "register_response"
	Status  string `json:"status"`            // "ok" or "error"
	Message string `json:"message,omitempty"` // human-friendly message
}

// Notification represents a chapter release notification (UC-010).
type Notification struct {
	Type      string `json:"type"` // "chapter_release"
	MangaID   string `json:"manga_id"`
	Title     string `json:"title"`
	Chapter   int    `json:"chapter"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// clientInfo stores registration information for a UDP client.
type clientInfo struct {
	Addr        net.UDPAddr
	UserID      string
	MangaIDs    []string
	Preferences []string
	Label       string
}

// Server is the UDP notification server implementation.
type Server struct {
	Port string

	mu      sync.RWMutex
	clients []clientInfo
}

// NewServer creates a new UDP server.
func NewServer(port string) *Server {
	if port == "" {
		port = "9091"
	}
	return &Server{
		Port: port,
	}
}

// FromEnv constructs a Server using environment variable MANGAHUB_UDP_PORT.
func FromEnv() *Server {
	port := os.Getenv("MANGAHUB_UDP_PORT")
	return NewServer(port)
}

// Start listens for UDP packets and handles registration and notifications.
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ":"+s.Port)
	if err != nil {
		return fmt.Errorf("udp resolve error: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("udp listen error: %w", err)
	}
	defer conn.Close()

	log.Println("UDP notification server listening on :" + s.Port)

	buf := make([]byte, 4096)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("udp read error:", err)
			continue
		}
		data := buf[:n]

		// Minimal envelope to inspect the message type.
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			log.Println("udp invalid json:", err)
			continue
		}

		switch envelope.Type {
		case "register":
			s.handleRegister(conn, clientAddr, data)
		case "unregister":
			s.handleUnregister(conn, clientAddr, data)
		case "chapter_release":
			s.handleNotification(conn, data)
		default:
			log.Println("udp unknown message type:", envelope.Type)
		}
	}
}

// handleRegister processes a registration message from a UDP client (UC-009).
func (s *Server) handleRegister(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	var msg RegisterMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Println("udp register unmarshal error:", err)
		_ = s.sendRegisterResponse(conn, addr, "error", "invalid_register_payload")
		return
	}

	if msg.UserID == "" {
		_ = s.sendRegisterResponse(conn, addr, "error", "missing_user_id")
		return
	}

	s.registerClient(clientInfo{
		Addr:        *addr,
		UserID:      msg.UserID,
		MangaIDs:    msg.MangaIDs,
		Preferences: msg.Preferences,
		Label:       msg.ClientLabel,
	})

	log.Printf("UDP: registered client %s at %v (manga_ids=%v)\n", msg.UserID, addr, msg.MangaIDs)
	_ = s.sendRegisterResponse(conn, addr, "ok", "registered")
}

// handleUnregister removes client registrations.
func (s *Server) handleUnregister(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	var msg UnregisterMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Println("udp unregister unmarshal error:", err)
		return
	}
	if msg.UserID == "" {
		return
	}

	s.unregisterClient(msg.UserID, msg.MangaIDs, addr)
	log.Printf("UDP: unregistered client %s at %v (manga_ids=%v)\n", msg.UserID, addr, msg.MangaIDs)
	_ = s.sendRegisterResponse(conn, addr, "ok", "unregistered")
}

func (s *Server) sendRegisterResponse(conn *net.UDPConn, addr *net.UDPAddr, status, message string) error {
	resp := RegisterResponse{
		Type:    "register_response",
		Status:  status,
		Message: message,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(data, addr)
	return err
}

// handleNotification processes a chapter release notification (UC-010).
func (s *Server) handleNotification(conn *net.UDPConn, data []byte) {
	var notif Notification
	if err := json.Unmarshal(data, &notif); err != nil {
		log.Println("udp notification unmarshal error:", err)
		return
	}

	if notif.Timestamp == 0 {
		notif.Timestamp = time.Now().Unix()
	}

	log.Printf("UDP: broadcasting chapter release notification manga=%s chapter=%d\n",
		notif.MangaID, notif.Chapter)

	s.broadcast(conn, notif)
}

// registerClient adds a client to the notification list.
func (s *Server) registerClient(info clientInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove any existing entries for this user (any address) to avoid duplication.
	var filtered []clientInfo
	for _, c := range s.clients {
		if c.UserID == info.UserID {
			continue
		}
		filtered = append(filtered, c)
	}
	filtered = append(filtered, info)
	s.clients = filtered
}

// unregisterClient removes entries for a user. If mangaIDs provided, remove matching those manga; otherwise remove all for the user (and address, if provided).
func (s *Server) unregisterClient(userID string, mangaIDs []string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []clientInfo
	for _, c := range s.clients {
		if c.UserID != userID {
			filtered = append(filtered, c)
			continue
		}
		// If no mangaIDs specified, drop all for this user (any addr)
		if len(mangaIDs) == 0 {
			continue
		}
		// Drop if there is overlap with requested mangaIDs (or stored entry has no mangaIDs)
		if overlap(c.MangaIDs, mangaIDs) || len(c.MangaIDs) == 0 {
			continue
		}
		// Otherwise keep
		filtered = append(filtered, c)
	}
	s.clients = filtered
}

func overlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// broadcast sends the notification to all registered clients (UC-010).
func (s *Server) broadcast(conn *net.UDPConn, n Notification) {
	data, err := json.Marshal(n)
	if err != nil {
		log.Println("udp marshal notification error:", err)
		return
	}

	s.mu.RLock()
	clients := make([]clientInfo, len(s.clients))
	copy(clients, s.clients)
	s.mu.RUnlock()

	for _, c := range clients {
		if _, err := conn.WriteToUDP(data, &c.Addr); err != nil {
			// A1/A2: client unreachable / network error - log and continue
			log.Printf("udp send error to %v (user=%s): %v\n", c.Addr, c.UserID, err)
			continue
		}
	}

	log.Printf("UDP: broadcasted notification to %d client(s) for manga=%s chapter=%d\n",
		len(clients), n.MangaID, n.Chapter)
}

// Helper to build a notification from CLI/admin args (optional convenience).
func BuildNotification(mangaID, title, message string, chapter int) Notification {
	return Notification{
		Type:      "chapter_release",
		MangaID:   mangaID,
		Title:     title,
		Chapter:   chapter,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
}

// ParseChapterFromEnv is a small helper to parse chapter from env var.
func ParseChapterFromEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}


