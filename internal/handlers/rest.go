package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"plivo/internal/pubsub"

	"github.com/gorilla/mux"
)

// RESTHandler handles REST API endpoints
type RESTHandler struct {
	hub *pubsub.Hub
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(hub *pubsub.Hub) *RESTHandler {
	return &RESTHandler{hub: hub}
}

// CreateTopicRequest represents the request body for creating a topic
type CreateTopicRequest struct {
	Name string `json:"name"`
}

// CreateTopic creates a new topic
func (h *RESTHandler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Topic name is required", http.StatusBadRequest)
		return
	}

	if err := h.hub.CreateTopic(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"topic":  req.Name,
	})
}

// ListTopics returns all topics
func (h *RESTHandler) ListTopics(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topics := h.hub.GetTopics()

	// Convert to the required format
	topicList := make([]map[string]interface{}, 0, len(topics))
	for _, topic := range topics {
		topicList = append(topicList, map[string]interface{}{
			"name":        topic.Name,
			"subscribers": topic.SubscriberCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics": topicList,
	})
}

// DeleteTopic deletes a topic
func (h *RESTHandler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	topicName := vars["topic"]

	if err := h.hub.DeleteTopic(topicName); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"topic":  topicName,
	})
}

// Health returns system health status
func (h *RESTHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Health endpoint doesn't require authentication
	stats := h.hub.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uptime_sec":  int(stats.Uptime.Seconds()),
		"topics":      stats.TotalTopics,
		"subscribers": stats.TotalClients,
	})
}

// Stats returns system statistics
func (h *RESTHandler) Stats(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !h.authenticateRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topics := h.hub.GetTopics()

	// Convert to the required format
	topicStats := make(map[string]map[string]interface{})
	for name, topic := range topics {
		topicStats[name] = map[string]interface{}{
			"messages":    topic.MessageCount,
			"subscribers": topic.SubscriberCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics": topicStats,
	})
}

// authenticateRequest checks X-API-Key header
func (h *RESTHandler) authenticateRequest(r *http.Request) bool {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		// No API key set, allow all requests
		return true
	}

	providedKey := r.Header.Get("X-API-Key")
	return providedKey == apiKey
}
