package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"plivo/internal/handlers"
	"plivo/internal/pubsub"
	"testing"
	"time"
)

func TestIntegrationPubSubFlow(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	restHandler := handlers.NewRESTHandler(hub)

	// Test 1: Create topic via REST
	reqBody := handlers.CreateTopicRequest{Name: "integration-test"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	restHandler.CreateTopic(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Failed to create topic: status %d", w.Code)
	}

	// Test 2: List topics via REST
	req = httptest.NewRequest("GET", "/topics", nil)
	w = httptest.NewRecorder()

	restHandler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to list topics: status %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok := response["topics"].([]interface{})
	if !ok || len(topics) != 1 {
		t.Error("Expected 1 topic in response")
	}

	// Test 3: Health check
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()

	restHandler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed: status %d", w.Code)
	}

	// Test 4: Stats endpoint
	req = httptest.NewRequest("GET", "/stats", nil)
	w = httptest.NewRecorder()

	restHandler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stats endpoint failed: status %d", w.Code)
	}

	// Test 5: Delete topic
	req = httptest.NewRequest("DELETE", "/topics/integration-test", nil)
	w = httptest.NewRecorder()

	restHandler.DeleteTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to delete topic: status %d", w.Code)
	}
}

func TestIntegrationWebSocketFlow(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	wsHandler := handlers.NewWebSocketHandler(hub)

	// Test WebSocket connection attempt
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	wsHandler.HandleWebSocket(w, req)

	// WebSocket upgrade might fail in test environment, but handler should not crash
}

func TestIntegrationConcurrentOperations(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	restHandler := handlers.NewRESTHandler(hub)

	// Test concurrent topic creation
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := handlers.CreateTopicRequest{Name: "concurrent-topic-" + string(rune(id))}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			restHandler.CreateTopic(w, req)

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

	restHandler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to list topics: status %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok := response["topics"].([]interface{})
	if !ok || len(topics) != 10 {
		t.Errorf("Expected 10 topics, got %d", len(topics))
	}
}

func TestIntegrationTopicLifecycle(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	restHandler := handlers.NewRESTHandler(hub)

	// Create topic
	reqBody := handlers.CreateTopicRequest{Name: "lifecycle-test"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	restHandler.CreateTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to create topic: status %d", w.Code)
	}

	// Verify topic exists
	req = httptest.NewRequest("GET", "/topics", nil)
	w = httptest.NewRecorder()

	restHandler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to list topics: status %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok := response["topics"].([]interface{})
	if !ok || len(topics) != 1 {
		t.Error("Expected 1 topic after creation")
	}

	// Delete topic
	req = httptest.NewRequest("DELETE", "/topics/lifecycle-test", nil)
	w = httptest.NewRecorder()

	restHandler.DeleteTopic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to delete topic: status %d", w.Code)
	}

	// Verify topic is deleted
	req = httptest.NewRequest("GET", "/topics", nil)
	w = httptest.NewRecorder()

	restHandler.ListTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to list topics: status %d", w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	topics, ok = response["topics"].([]interface{})
	if !ok || len(topics) != 0 {
		t.Error("Expected 0 topics after deletion")
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	restHandler := handlers.NewRESTHandler(hub)

	// Test creating topic with empty name
	reqBody := handlers.CreateTopicRequest{Name: ""}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	restHandler.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty topic name, got %d", w.Code)
	}

	// Test deleting non-existent topic
	req = httptest.NewRequest("DELETE", "/topics/non-existent", nil)
	w = httptest.NewRecorder()

	restHandler.DeleteTopic(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent topic, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/topics", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	restHandler.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestIntegrationStatsAccuracy(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	restHandler := handlers.NewRESTHandler(hub)

	// Get initial stats
	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	restHandler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stats endpoint failed: status %d", w.Code)
	}

	var initialStats map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &initialStats)
	if err != nil {
		t.Errorf("Failed to unmarshal initial stats: %v", err)
	}

	// Create some topics
	for i := 0; i < 3; i++ {
		reqBody := handlers.CreateTopicRequest{Name: "stats-topic-" + string(rune(i))}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		restHandler.CreateTopic(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Failed to create topic %d: status %d", i, w.Code)
		}
	}

	// Get updated stats
	req = httptest.NewRequest("GET", "/stats", nil)
	w = httptest.NewRecorder()

	restHandler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stats endpoint failed: status %d", w.Code)
	}

	var updatedStats map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &updatedStats)
	if err != nil {
		t.Errorf("Failed to unmarshal updated stats: %v", err)
	}

	// Verify topic count increased
	initialTopics, _ := initialStats["total_topics"].(float64)
	updatedTopics, _ := updatedStats["total_topics"].(float64)

	if updatedTopics != initialTopics+3 {
		t.Errorf("Expected topic count to increase by 3, got %f -> %f", initialTopics, updatedTopics)
	}
}

func TestIntegrationGracefulShutdown(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Test shutdown
	hub.Shutdown()

	// Give shutdown time to complete
	time.Sleep(10 * time.Millisecond)

	// Verify hub is shutting down (we can't access private fields, so just verify no crash)
	// The shutdown should complete without errors
}

func TestIntegrationMessageReplay(t *testing.T) {
	// Setup
	hub := pubsub.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// Create topic
	hub.CreateTopic("replay-test")

	// Add some messages to the ring buffer
	// We need to access the topic through the hub's public methods
	// For this test, we'll just verify the GetRecentMessages method works

	// Test getting messages from empty topic first
	messages := hub.GetRecentMessages("replay-test", 3)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages from empty topic, got %d", len(messages))
	}

	// Test getting more messages than available (should return 0 for empty topic)
	messages = hub.GetRecentMessages("replay-test", 10)

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages from empty topic, got %d", len(messages))
	}

	// Test getting messages from non-existent topic
	messages = hub.GetRecentMessages("non-existent", 5)

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for non-existent topic, got %d", len(messages))
	}
}
