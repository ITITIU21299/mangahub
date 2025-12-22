package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

type ProgressUpdate struct {
	UserID   string `json:"user_id"`
	MangaID  string `json:"manga_id"`
	Chapter  int    `json:"chapter"`
	Timestamp int64 `json:"timestamp"`
}

type ProgressSyncServer struct {
	Port        string
	connections map[string]net.Conn
	mu          sync.RWMutex
	Broadcast   chan ProgressUpdate
}

func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		Port:        port,
		connections: make(map[string]net.Conn),
		Broadcast:   make(chan ProgressUpdate, 64),
	}
}

func (s *ProgressSyncServer) Start() error {
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		return err
	}
	log.Println("TCP progress sync listening on :" + s.Port)

	go s.broadcastLoop()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *ProgressSyncServer) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	// First line is a simple user ID for demo purposes.
	userID, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	userID = userID[:len(userID)-1]

	s.mu.Lock()
	s.connections[userID] = conn
	s.mu.Unlock()
	log.Printf("user %s connected\n", userID)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		var upd ProgressUpdate
		if err := json.Unmarshal(line, &upd); err != nil {
			log.Printf("Failed to unmarshal progress update from %s: %v", userID, err)
			continue
		}
		log.Printf("Received progress update from %s: manga=%s, chapter=%d", userID, upd.MangaID, upd.Chapter)
		s.Broadcast <- upd
	}

	s.mu.Lock()
	delete(s.connections, userID)
	s.mu.Unlock()
	log.Printf("user %s disconnected\n", userID)
}

func (s *ProgressSyncServer) broadcastLoop() {
	for upd := range s.Broadcast {
		data, err := json.Marshal(upd)
		if err != nil {
			log.Printf("Failed to marshal broadcast update: %v", err)
			continue
		}
		data = append(data, '\n')

		s.mu.RLock()
		clientCount := len(s.connections)
		for userID, conn := range s.connections {
			if _, err := conn.Write(data); err != nil {
				log.Printf("failed to send to %s: %v", userID, err)
			}
		}
		s.mu.RUnlock()
		
		if clientCount > 0 {
			log.Printf("Broadcasted progress update: manga=%s, chapter=%d to %d client(s)", upd.MangaID, upd.Chapter, clientCount)
		} else {
			log.Printf("Broadcasted progress update: manga=%s, chapter=%d (no connected clients)", upd.MangaID, upd.Chapter)
		}
	}
}

func main() {
	srv := NewProgressSyncServer("9090")
	if err := srv.Start(); err != nil {
		fmt.Println("tcp server error:", err)
	}
}


