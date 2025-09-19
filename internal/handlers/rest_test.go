package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"plivo/internal/pubsub"
	"testing"
)

func TestNewRESTHandler(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	if handler == nil {
		t.Fatal("NewRESTHandler() returned nil")
	}

	if handler.hub != hub {
		t.Error("Hub reference is incorrect")
	}
}

func TestCreateTopic(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Test valid topic creation
	reqBody := CreateTopicRequest{Name: "test-topic"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test duplicate topic creation
	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Test missing topic name
	reqBody = CreateTopicRequest{Name: ""}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestListTopics(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Create some topics
	hub.CreateTopic("topic1")
	hub.CreateTopic("topic2")

	req := httptest.NewRequest("GET", "/topics", nil)
	w := httptest.NewRecorder()

	handler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok := response["topics"].([]interface{})
	if !ok {
		t.Error("Response should contain topics array")
	}

	if len(topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(topics))
	}
}

func TestDeleteTopic(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Create a topic first
	hub.CreateTopic("test-topic")

	// Test deleting existing topic
	req := httptest.NewRequest("DELETE", "/topics/test-topic", nil)
	w := httptest.NewRecorder()

	handler.DeleteTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test deleting non-existent topic
	req = httptest.NewRequest("DELETE", "/topics/non-existent", nil)
	w = httptest.NewRecorder()

	handler.DeleteTopic(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHealth(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check required fields
	requiredFields := []string{"uptime_sec", "topics", "subscribers"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Response missing required field: %s", field)
		}
	}
}

func TestStats(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Create some topics
	hub.CreateTopic("topic1")
	hub.CreateTopic("topic2")

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check required fields
	requiredFields := []string{"topics"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Response missing required field: %s", field)
		}
	}
}

func TestAuthentication(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Set API key
	os.Setenv("API_KEY", "test-key")
	defer os.Unsetenv("API_KEY")

	// Test request without API key
	reqBody := CreateTopicRequest{Name: "test-topic"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Test request with correct API key
	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")
	w = httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test request with incorrect API key
	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "wrong-key")
	w = httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNoAuthenticationWhenKeyNotSet(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Ensure no API key is set
	os.Unsetenv("API_KEY")

	// Test request without API key should succeed
	reqBody := CreateTopicRequest{Name: "test-topic"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHealthEndpointNoAuth(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Set API key
	os.Setenv("API_KEY", "test-key")
	defer os.Unsetenv("API_KEY")

	// Health endpoint should not require authentication
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestContentTypeValidation(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Test without Content-Type header
	reqBody := CreateTopicRequest{Name: "test-topic"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.CreateTopic(w, req)

	// Should still work as we're not strictly validating Content-Type
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestConcurrentRequests(t *testing.T) {
	hub := pubsub.NewHub()
	handler := NewRESTHandler(hub)

	// Test concurrent topic creation
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := CreateTopicRequest{Name: "topic-" + string(rune(id))}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateTopic(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Failed to create topic %d: status %d", id, w.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all topics were created
	req := httptest.NewRequest("GET", "/topics", nil)
	w := httptest.NewRecorder()

	handler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to list topics: status %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok := response["topics"].([]interface{})
	if !ok {
		t.Error("Response should contain topics array")
	}

	if len(topics) != 10 {
		t.Errorf("Expected 10 topics, got %d", len(topics))
	}
}
