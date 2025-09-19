package handlers

import (
	"log"
	"net/http"
	"os"
	"plivo/internal/pubsub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *pubsub.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *pubsub.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check authentication if API key is set
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := uuid.New().String()
	client := pubsub.NewClient(h.hub, conn, clientID)
	h.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

// authenticateRequest checks X-API-Key header
func (h *WebSocketHandler) authenticateRequest(r *http.Request) bool {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		// No API key set, allow all requests
		return true
	}

	providedKey := r.Header.Get("X-API-Key")
	return providedKey == apiKey
}
