package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"mangahub/internal/auth"
	"mangahub/internal/database"
	"mangahub/internal/grpc"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ChatPayload and WebSocket server code (simplified from websocket-server)
type ChatPayload struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Room      string `json:"room"`
	Target    string `json:"target"`
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
	history    []ChatPayload
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

func (h *ChatHub) run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			h.Clients[c.Conn] = c.Username
			h.mu.Unlock()
			h.Broadcast <- ChatPayload{
				Type:      "join",
				Username:  c.Username,
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
				Timestamp: time.Now().Unix(),
				Room:      "general",
			}
		case msg := <-h.Broadcast:
			if msg.Type == "chat" {
				h.mu.Lock()
				if len(h.history) >= 50 {
					h.history = h.history[1:]
				}
				h.history = append(h.history, msg)
				h.mu.Unlock()
			}
			h.mu.RLock()
			for conn := range h.Clients {
				_ = conn.WriteJSON(msg)
			}
			h.mu.RUnlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	// Initialize database - use absolute path or path relative to mangahub root
	dbPath := os.Getenv("MANGAHUB_DB_PATH")
	if dbPath == "" {
		// Try to use mangahub.db in the project root (where seed script puts it)
		// If running from cmd/all-servers, go up two levels
		if _, err := os.Stat("../../mangahub.db"); err == nil {
			dbPath = "../../mangahub.db"
		} else if _, err := os.Stat("../mangahub.db"); err == nil {
			dbPath = "../mangahub.db"
		} else {
			dbPath = "mangahub.db"
		}
	}

	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatal("init db:", err)
	}
	defer db.Close()

	// Initialize services
	jwtSecret := []byte(os.Getenv("MANGAHUB_JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-me")
	}

	authSvc := auth.NewService(db)
	mangaSvc := manga.NewService(db)
	userSvc := user.NewService(db)
	userSvc.SetMangaService(mangaSvc)

	var wg sync.WaitGroup

	// Store server references for graceful shutdown
	var httpServer *http.Server
	var grpcServer *grpc.Server
	var wsServer *http.Server

	// 1. HTTP API Server (port 8080)
	wg.Add(1)
	go func() {
		defer wg.Done()
		gin.SetMode(gin.ReleaseMode)
		r := gin.Default()

		// CORS configuration - allow requests from frontend
		corsOrigins := os.Getenv("MANGAHUB_CORS_ORIGINS")
		var allowedOrigins []string
		if corsOrigins != "" {
			allowedOrigins = []string{}
			for _, origin := range strings.Split(corsOrigins, ",") {
				allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
			}
		} else {
			// Default: allow localhost and common development origins
			allowedOrigins = []string{
				"http://localhost:3000",
				"http://127.0.0.1:3000",
				"http://25.17.216.66:3000",
				"http://25.19.136.155:3000",
				"http://0.0.0.0:3000",
			}
		}

		// Function to check if origin is allowed (for local network IPs)
		allowOriginFunc := func(origin string) bool {
			// Check if origin is in the explicit allowed list
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			// Allow localhost variants
			if strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "http://0.0.0.0:") {
				return true
			}
			// Allow local network IPs:
			// - 10.x.x.x (private class A)
			// - 172.16-31.x.x (private class B)
			// - 192.168.x.x (private class C)
			// - 25.x.x.x (your network range)
			if strings.HasPrefix(origin, "http://10.") ||
				strings.HasPrefix(origin, "http://192.168.") ||
				strings.HasPrefix(origin, "http://25.") {
				return true
			}
			// Check for 172.16-31.x.x range (private class B)
			// Match patterns like "http://172.16.", "http://172.17.", ..., "http://172.31."
			if strings.HasPrefix(origin, "http://172.") {
				// Extract the second octet to verify it's in 16-31 range
				// "http://172." = 11 chars, so check the next part
				if len(origin) >= 14 {
					secondOctetStart := origin[11:14] // Get "16.", "17.", "20.", "31.", etc.
					// Check if second octet is 16-31
					if (strings.HasPrefix(secondOctetStart, "16") ||
						strings.HasPrefix(secondOctetStart, "17") ||
						strings.HasPrefix(secondOctetStart, "18") ||
						strings.HasPrefix(secondOctetStart, "19") ||
						strings.HasPrefix(secondOctetStart, "20") ||
						strings.HasPrefix(secondOctetStart, "21") ||
						strings.HasPrefix(secondOctetStart, "22") ||
						strings.HasPrefix(secondOctetStart, "23") ||
						strings.HasPrefix(secondOctetStart, "24") ||
						strings.HasPrefix(secondOctetStart, "25") ||
						strings.HasPrefix(secondOctetStart, "26") ||
						strings.HasPrefix(secondOctetStart, "27") ||
						strings.HasPrefix(secondOctetStart, "28") ||
						strings.HasPrefix(secondOctetStart, "29") ||
						strings.HasPrefix(secondOctetStart, "30") ||
						strings.HasPrefix(secondOctetStart, "31")) &&
						len(secondOctetStart) >= 3 && secondOctetStart[2] == '.' {
						return true
					}
				}
			}
			return false
		}

		r.Use(cors.New(cors.Config{
			AllowOriginFunc:  allowOriginFunc,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))

		// Public proxy endpoint for MangaDex cover images to avoid hotlinking placeholders
		r.GET("/proxy/cover", func(c *gin.Context) {
			url := c.Query("url")
			if url == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing url parameter"})
				return
			}

			resp, err := http.Get(url)
			if err != nil {
				log.Printf("proxy cover error: %v", err)
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch image"})
				return
			}
			defer resp.Body.Close()

			c.Header("Content-Type", resp.Header.Get("Content-Type"))
			c.Status(resp.StatusCode)
			if _, err := io.Copy(c.Writer, resp.Body); err != nil {
				log.Printf("proxy cover write error: %v", err)
			}
		})

		authMiddleware := auth.RegisterRoutes(r, authSvc, jwtSecret)
		authGroup := r.Group("/")
		authGroup.Use(authMiddleware)
		{
			manga.RegisterRoutes(authGroup, mangaSvc)
			user.RegisterRoutes(authGroup, userSvc)
		}

		// Bind to all interfaces (0.0.0.0) to allow network access
		bindAddr := os.Getenv("MANGAHUB_API_ADDR")
		if bindAddr == "" {
			bindAddr = "0.0.0.0:8080"
		}
		httpServer = &http.Server{
			Addr:    bindAddr,
			Handler: r,
		}
		log.Printf("✅ HTTP API server listening on %s", bindAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP API server error: %v", err)
		}
	}()

	// 2. gRPC Server (port 9092)
	wg.Add(1)
	go func() {
		defer wg.Done()
		grpcServer = grpc.NewServer(mangaSvc, userSvc)
		log.Println("✅ gRPC server listening on :9092")
		if err := grpcServer.Start(":9092"); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// 3. TCP Server (port 9090)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpSrv := tcp.FromEnv()
		log.Println("✅ TCP server listening on :9090")
		// TCP server will stop when context is cancelled (listener will close)
		// For now, just run it - it will stop when process exits
		if err := tcpSrv.Start(); err != nil {
			log.Printf("TCP server error: %v", err)
		}
	}()

	// 4. UDP Server (port 9091)
	wg.Add(1)
	go func() {
		defer wg.Done()
		udpSrv := udp.FromEnv()
		log.Println("✅ UDP server listening on :9091")
		if err := udpSrv.Start(); err != nil {
			log.Printf("UDP server error: %v", err)
		}
	}()

	// 5. WebSocket Server (port 9093)
	wg.Add(1)
	go func() {
		defer wg.Done()
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
				return
			}
			hub.Register <- Client{Conn: conn, Username: username}

			go func() {
				defer func() { hub.Unregister <- conn }()
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

		wsServer = &http.Server{
			Addr:    ":9093",
			Handler: r,
		}
		log.Println("✅ WebSocket server listening on :9093/ws")
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	log.Println("🚀 All MangaHub servers started!")
	log.Println("   - HTTP API:    http://localhost:8080")
	log.Println("   - gRPC:        localhost:9092")
	log.Println("   - TCP:         localhost:9090")
	log.Println("   - UDP:         localhost:9091")
	log.Println("   - WebSocket:   ws://localhost:9093/ws")
	log.Println("\nPress Ctrl+C to stop all servers...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down all servers...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP API server
	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP API server shutdown error: %v", err)
		}
	}

	// Shutdown WebSocket server
	if wsServer != nil {
		if err := wsServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("WebSocket server shutdown error: %v", err)
		}
	}

	// Stop gRPC server
	if grpcServer != nil {
		grpcServer.Stop()
	}

	// Note: TCP and UDP servers will stop when the process exits
	// They don't have graceful shutdown methods, but closing listeners will cause them to exit

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for shutdown or timeout
	select {
	case <-done:
		log.Println("✅ All servers stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("⚠️  Shutdown timeout - some servers may still be running")
	}
}
