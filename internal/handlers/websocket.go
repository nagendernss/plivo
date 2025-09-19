package handlers

import (
	"log"
	"net/http"
	"plivo/internal/config"
	"plivo/internal/pubsub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *pubsub.Hub
	cfg *config.Config
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *pubsub.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
		cfg: cfg,
	}
}

// getUpgrader returns a websocket upgrader with CORS configuration
func (h *WebSocketHandler) getUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if !h.cfg.Security.EnableCORS {
				return true // Allow all origins when CORS is disabled
			}
			// TODO: Implement proper origin checking based on AllowedOrigins
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check authentication if API key is set
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := h.getUpgrader()
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
	apiKey := h.cfg.Security.APIKey
	if apiKey == "" {
		// No API key set, allow all requests
		return true
	}

	providedKey := r.Header.Get("X-API-Key")
	return providedKey == apiKey
}
