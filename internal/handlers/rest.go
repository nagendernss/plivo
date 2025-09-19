package handlers

import (
	"encoding/json"
	"net/http"
	"plivo/internal/config"
	"plivo/internal/pubsub"

	"github.com/gorilla/mux"
)

// RESTHandler handles REST API endpoints
type RESTHandler struct {
	hub *pubsub.Hub
	cfg *config.Config
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(hub *pubsub.Hub, cfg *config.Config) *RESTHandler {
	return &RESTHandler{
		hub: hub,
		cfg: cfg,
	}
}

// CreateTopicRequest represents the request body for creating a topic
type CreateTopicRequest struct {
	Name string `json:"name"`
}

// CreateTopic creates a new topic
// @Summary Create a new topic
// @Description Create a new pub/sub topic for message publishing and subscription
// @Tags topics
// @Accept json
// @Produce json
// @Param request body CreateTopicRequest true "Topic creation request"
// @Success 201 {object} map[string]string "Topic created successfully"
// @Failure 400 {string} string "Bad request - invalid JSON or missing topic name"
// @Failure 401 {string} string "Unauthorized - invalid or missing API key"
// @Failure 409 {string} string "Conflict - topic already exists"
// @Security ApiKeyAuth
// @Router /topics [post]
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
// @Summary List all topics
// @Description Get a list of all available topics with their subscriber counts
// @Tags topics
// @Produce json
// @Success 200 {object} map[string]interface{} "List of topics"
// @Failure 401 {string} string "Unauthorized - invalid or missing API key"
// @Security ApiKeyAuth
// @Router /topics [get]
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
// @Summary Delete a topic
// @Description Delete a topic and disconnect all its subscribers
// @Tags topics
// @Produce json
// @Param topic path string true "Topic name"
// @Success 200 {object} map[string]string "Topic deleted successfully"
// @Failure 401 {string} string "Unauthorized - invalid or missing API key"
// @Failure 404 {string} string "Not found - topic does not exist"
// @Security ApiKeyAuth
// @Router /topics/{topic} [delete]
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
// @Summary Health check
// @Description Get system health status including uptime and basic metrics
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "System health status"
// @Router /health [get]
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
// @Summary System statistics
// @Description Get detailed system statistics including topic metrics and performance data
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "System statistics"
// @Failure 401 {string} string "Unauthorized - invalid or missing API key"
// @Security ApiKeyAuth
// @Router /stats [get]
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
	apiKey := h.cfg.Security.APIKey
	if apiKey == "" {
		// No API key set, allow all requests
		return true
	}

	providedKey := r.Header.Get("X-API-Key")
	return providedKey == apiKey
}
