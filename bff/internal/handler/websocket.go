package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	gorillaws "github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/online-judge/bff/internal/websocket"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Restrict in production
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub   *websocket.Hub
	redis *redis.Client
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, redis *redis.Client) *WebSocketHandler {
	return &WebSocketHandler{
		hub:   hub,
		redis: redis,
	}
}

// HandleWebSocket handles WebSocket connection upgrade
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := websocket.NewClient(h.hub, conn, userID)

	h.hub.RegisterClient(client)

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ws", h.HandleWebSocket)
}