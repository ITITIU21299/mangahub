package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Notification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type NotificationServer struct {
	Port    string
	clients []net.UDPAddr
	mu      sync.RWMutex
}

func NewNotificationServer(port string) *NotificationServer {
	return &NotificationServer{
		Port: port,
	}
}

func (s *NotificationServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ":"+s.Port)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("UDP notification server listening on :" + s.Port)

	buf := make([]byte, 2048)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		msg := string(buf[:n])
		if msg == "REGISTER" {
			s.registerClient(*clientAddr)
			log.Printf("registered UDP client %v\n", clientAddr)
			continue
		}

		var notif Notification
		if err := json.Unmarshal([]byte(msg), &notif); err != nil {
			continue
		}
		if notif.Timestamp == 0 {
			notif.Timestamp = time.Now().Unix()
		}
		s.broadcast(conn, notif)
	}
}

func (s *NotificationServer) registerClient(addr net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, addr)
}

func (s *NotificationServer) broadcast(conn *net.UDPConn, n Notification) {
	data, err := json.Marshal(n)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		if _, err := conn.WriteToUDP(data, &c); err != nil {
			log.Println("udp send error:", err)
		}
	}
}

func main() {
	srv := NewNotificationServer("9091")
	if err := srv.Start(); err != nil {
		fmt.Println("udp server error:", err)
	}
}


