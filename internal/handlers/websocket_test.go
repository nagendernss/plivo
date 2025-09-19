package handlers

import (
	"net/http"
	"net/http/httptest"
	"plivo/internal/config"
	"plivo/internal/pubsub"
	"testing"
)

func TestNewWebSocketHandler(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfig()
	handler := NewWebSocketHandler(hub, cfg)

	if handler == nil {
		t.Fatal("NewWebSocketHandler() returned nil")
	}

	if handler.hub != hub {
		t.Error("Hub reference is incorrect")
	}

	if handler.cfg != cfg {
		t.Error("Config reference is incorrect")
	}
}

func TestWebSocketAuthentication(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// API key is set in the config

	// Test request without API key
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Test request with correct API key
	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("X-API-Key", "test-key")
	w = httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	// Should not return 401 (WebSocket upgrade might fail for other reasons)
	if w.Code == http.StatusUnauthorized {
		t.Error("Request with correct API key should not return 401")
	}

	// Test request with incorrect API key
	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w = httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestWebSocketNoAuthenticationWhenKeyNotSet(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfig() // No API key set
	handler := NewWebSocketHandler(hub, cfg)

	// Test request without API key should not return 401
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Error("Request without API key should not return 401 when no key is set")
	}
}

func TestWebSocketUpgrader(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfig()
	handler := NewWebSocketHandler(hub, cfg)

	// Test that upgrader is configured correctly
	upgrader := handler.getUpgrader()
	if upgrader.CheckOrigin == nil {
		t.Error("Upgrader CheckOrigin should be set")
	}

	// Test CheckOrigin function
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	// Should allow all origins (returns true)
	if !upgrader.CheckOrigin(req) {
		t.Error("CheckOrigin should allow all origins")
	}
}

func TestWebSocketHandlerIntegration(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown()

	// Test WebSocket connection attempt
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	// The response code will depend on whether WebSocket upgrade succeeds
	// In a real test environment, this might fail due to missing WebSocket headers
	// But we can verify the handler doesn't crash
}

func TestAuthenticationFunction(t *testing.T) {
	hub := pubsub.NewHub()

	// Test with no API key set
	cfg := config.NewTestConfig() // No API key
	handler := NewWebSocketHandler(hub, cfg)
	req := httptest.NewRequest("GET", "/ws", nil)

	if !handler.authenticateRequest(req) {
		t.Error("Should authenticate when no API key is set")
	}

	// Test with API key set but no header
	cfgWithKey := config.NewTestConfigWithAPIKey("test-key")
	handlerWithKey := NewWebSocketHandler(hub, cfgWithKey)

	if handlerWithKey.authenticateRequest(req) {
		t.Error("Should not authenticate when API key is set but no header provided")
	}

	// Test with correct API key
	req.Header.Set("X-API-Key", "test-key")

	if !handlerWithKey.authenticateRequest(req) {
		t.Error("Should authenticate with correct API key")
	}

	// Test with incorrect API key
	req.Header.Set("X-API-Key", "wrong-key")

	if handlerWithKey.authenticateRequest(req) {
		t.Error("Should not authenticate with incorrect API key")
	}
}

func TestWebSocketHandlerConcurrency(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown()

	// Test concurrent WebSocket connection attempts
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/ws", nil)
			w := httptest.NewRecorder()

			handler.HandleWebSocket(w, req)

			// Handler should not crash
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestWebSocketHandlerWithDifferentOrigins(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// Test with different origins
	origins := []string{
		"http://localhost:3000",
		"https://example.com",
		"http://192.168.1.1:8080",
		"",
	}

	for _, origin := range origins {
		req := httptest.NewRequest("GET", "/ws", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}

		// Should not crash regardless of origin
		w := httptest.NewRecorder()
		handler.HandleWebSocket(w, req)
	}
}

func TestWebSocketHandlerErrorHandling(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// Test with malformed request
	req := httptest.NewRequest("POST", "/ws", nil) // Wrong method
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	// Should handle gracefully without crashing
}

func TestWebSocketHandlerWithHeaders(t *testing.T) {
	hub := pubsub.NewHub()
	cfg := config.NewTestConfigWithAPIKey("test-key")
	handler := NewWebSocketHandler(hub, cfg)

	// Test with various headers
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("User-Agent", "TestClient/1.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	// Should handle gracefully
}
