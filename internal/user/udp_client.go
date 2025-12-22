package user

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// UDPRegisterOptions defines parameters for registering for UDP notifications.
type UDPRegisterOptions struct {
	ServerAddr  string   // e.g. "localhost:9091"
	UserID      string
	MangaIDs    []string
	Preferences []string
	ClientLabel string
}

// UDPNotification represents a chapter release notification to send via UDP.
type UDPNotification struct {
	ServerAddr string
	MangaID    string
	Title      string
	Chapter    int
	Message    string
}

// Unregister options for UDP.
type UDPUnregisterOptions struct {
	ServerAddr string // e.g. "localhost:9091"
	UserID     string
	MangaIDs   []string // optional: if empty, remove all for user
}

// internal types mirror internal/udp structures.
type udpRegisterMessage struct {
	Type        string   `json:"type"`
	UserID      string   `json:"user_id"`
	MangaIDs    []string `json:"manga_ids,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
	ClientLabel string   `json:"client_label,omitempty"`
}

type udpRegisterResponse struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type udpNotification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Title     string `json:"title"`
	Chapter   int    `json:"chapter"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// RegisterForUDPNotifications implements UC-009 for a given user ID.
func RegisterForUDPNotifications(opts UDPRegisterOptions) (string, error) {
	if opts.ServerAddr == "" {
		opts.ServerAddr = "localhost:9091"
	}
	if opts.UserID == "" {
		return "", fmt.Errorf("user ID is required for UDP registration")
	}

	serverAddr, err := net.ResolveUDPAddr("udp", opts.ServerAddr)
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	msg := udpRegisterMessage{
		Type:        "register",
		UserID:      opts.UserID,
		MangaIDs:    opts.MangaIDs,
		Preferences: opts.Preferences,
		ClientLabel: opts.ClientLabel,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	// Send register message
	if _, err := conn.Write(data); err != nil {
		return "", err
	}

	// Read confirmation (with simple timeout)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return "", err
	}

	var resp udpRegisterResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return "", err
	}

	if resp.Status != "ok" {
		return resp.Message, fmt.Errorf("udp register error: %s", resp.Message)
	}
	return resp.Message, nil
}

// SendUDPNotification implements UC-010 notification trigger (admin usage).
func SendUDPNotification(n UDPNotification) error {
	addr := n.ServerAddr
	if addr == "" {
		addr = "localhost:9091"
	}

	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	notif := udpNotification{
		Type:      "chapter_release",
		MangaID:   n.MangaID,
		Title:     n.Title,
		Chapter:   n.Chapter,
		Message:   n.Message,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

// UnregisterForUDPNotifications sends an unregister message to the UDP server.
func UnregisterForUDPNotifications(opts UDPUnregisterOptions) error {
	addr := opts.ServerAddr
	if addr == "" {
		addr = "localhost:9091"
	}
	if opts.UserID == "" {
		return fmt.Errorf("user ID is required for UDP unregister")
	}

	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := struct {
		Type     string   `json:"type"`
		UserID   string   `json:"user_id"`
		MangaIDs []string `json:"manga_ids,omitempty"`
	}{
		Type:     "unregister",
		UserID:   opts.UserID,
		MangaIDs: opts.MangaIDs,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}



