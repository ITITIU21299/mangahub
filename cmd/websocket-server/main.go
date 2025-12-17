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

type ChatMessage struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Message  string `json:"message"`
	Timestamp int64 `json:"timestamp"`
}

type ClientConnection struct {
	Conn     *websocket.Conn
	Username string
}

type ChatHub struct {
	Clients    map[*websocket.Conn]string
	Broadcast  chan ChatMessage
	Register   chan ClientConnection
	Unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func newHub() *ChatHub {
	return &ChatHub{
		Clients:    make(map[*websocket.Conn]string),
		Broadcast:  make(chan ChatMessage, 64),
		Register:   make(chan ClientConnection),
		Unregister: make(chan *websocket.Conn),
	}
}

func (h *ChatHub) run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			h.Clients[c.Conn] = c.Username
			h.mu.Unlock()
		case conn := <-h.Unregister:
			h.mu.Lock()
			delete(h.Clients, conn)
			h.mu.Unlock()
			conn.Close()
		case msg := <-h.Broadcast:
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
		hub.Register <- ClientConnection{Conn: conn, Username: username}

		go func() {
			defer func() {
				hub.Unregister <- conn
			}()
			for {
				var msg ChatMessage
				if err := conn.ReadJSON(&msg); err != nil {
					break
				}
				if msg.Timestamp == 0 {
					msg.Timestamp = time.Now().Unix()
				}
				if msg.Username == "" {
					msg.Username = username
				}
				hub.Broadcast <- msg
			}
		}()
	})

	fmt.Println("WebSocket chat server on :9093/ws")
	if err := r.Run(":9093"); err != nil {
		log.Fatal(err)
	}
}


