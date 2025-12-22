package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ChatPayload is the message format exchanged over WebSocket.
type ChatPayload struct {
	Type      string `json:"type"`       // "chat", "typing", "join", "leave", "friend_request", "friend_response", "system"
	UserID    string `json:"user_id"`    // optional
	Username  string `json:"username"`   // required for display
	Message   string `json:"message"`    // chat text or system text
	Timestamp int64  `json:"timestamp"`  // unix seconds
	Room      string `json:"room"`       // logical chat room, e.g. "general", "friends", "group-123", "dm:username"
	Target    string `json:"target"`     // for friend requests/responses
}

type Client struct {
	Conn     *websocket.Conn
	Username string
}

type ChatHub struct {
	Clients    map[*websocket.Conn]string
	Broadcast  chan ChatPayload
	Register   chan Client
	Unregister chan *websocket.Conn
	mu         sync.RWMutex
	history    []ChatPayload // simple in-memory history (recent 50)
}

func newHub() *ChatHub {
	return &ChatHub{
		Clients:    make(map[*websocket.Conn]string),
		Broadcast:  make(chan ChatPayload, 128),
		Register:   make(chan Client),
		Unregister: make(chan *websocket.Conn),
		history:    make([]ChatPayload, 0, 50),
	}
}

func (h *ChatHub) addHistory(msg ChatPayload) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.history) >= 50 {
		h.history = h.history[1:]
	}
	h.history = append(h.history, msg)
}

func (h *ChatHub) sendHistory(conn *websocket.Conn) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, msg := range h.history {
		_ = conn.WriteJSON(msg)
	}
}

func (h *ChatHub) run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			h.Clients[c.Conn] = c.Username
			h.mu.Unlock()
			// send history to newcomer
			h.sendHistory(c.Conn)
			// broadcast join
			h.Broadcast <- ChatPayload{
				Type:      "join",
				Username:  c.Username,
				Message:   "",
				Timestamp: time.Now().Unix(),
				Room:      "general",
			}
		case conn := <-h.Unregister:
			h.mu.Lock()
			username := h.Clients[conn]
			delete(h.Clients, conn)
			h.mu.Unlock()
			conn.Close()
			h.Broadcast <- ChatPayload{
				Type:      "leave",
				Username:  username,
				Message:   "",
				Timestamp: time.Now().Unix(),
				Room:      "general",
			}
		case msg := <-h.Broadcast:
			// keep history only for chat messages
			if msg.Type == "chat" {
				h.addHistory(msg)
			}
			h.mu.RLock()
			for conn := range h.Clients {
				if err := conn.WriteJSON(msg); err != nil {
					log.Println("ws write error:", err)
				}
			}
			h.mu.RUnlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	hub := newHub()
	go hub.run()

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		username := c.Query("username")
		if username == "" {
			username = "guest"
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("upgrade:", err)
			return
		}
		hub.Register <- Client{Conn: conn, Username: username}

		go func() {
			defer func() {
				hub.Unregister <- conn
			}()
			for {
				var payload ChatPayload
				if err := conn.ReadJSON(&payload); err != nil {
					break
				}
				if payload.Timestamp == 0 {
					payload.Timestamp = time.Now().Unix()
				}
				if payload.Username == "" {
					payload.Username = username
				}
				if payload.Room == "" {
					payload.Room = "general"
				}
				if payload.Type == "" {
					payload.Type = "chat"
				}
				hub.Broadcast <- payload
			}
		}()
	})

	fmt.Println("WebSocket chat server on :9093/ws")
	if err := r.Run(":9093"); err != nil {
		log.Fatal(err)
	}
}


