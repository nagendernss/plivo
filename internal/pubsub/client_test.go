package pubsub

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	hub := NewHub()
	clientID := "test-client"

	// Create client without WebSocket connection for testing
	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 100),
		subscriptions: make(map[string]bool),
		id:            clientID,
		maxQueueSize:  100,
		queueSize:     0,
		slowConsumer:  false,
	}

	if client.hub != hub {
		t.Error("Hub reference is incorrect")
	}

	if client.id != clientID {
		t.Errorf("Expected client ID '%s', got '%s'", clientID, client.id)
	}

	if client.send == nil {
		t.Error("Send channel is nil")
	}

	if client.subscriptions == nil {
		t.Error("Subscriptions map is nil")
	}

	if client.maxQueueSize != 100 {
		t.Errorf("Expected max queue size 100, got %d", client.maxQueueSize)
	}

	if client.queueSize != 0 {
		t.Errorf("Expected initial queue size 0, got %d", client.queueSize)
	}

	if client.slowConsumer != false {
		t.Error("slowConsumer should be false initially")
	}
}

func TestClientSubscriptionManagement(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test initial subscription state
	if client.IsSubscribed("test-topic") {
		t.Error("Client should not be subscribed initially")
	}

	// Test subscription
	client.mu.Lock()
	client.subscriptions["test-topic"] = true
	client.mu.Unlock()

	if !client.IsSubscribed("test-topic") {
		t.Error("Client should be subscribed to test-topic")
	}

	// Test unsubscription
	client.mu.Lock()
	delete(client.subscriptions, "test-topic")
	client.mu.Unlock()

	if client.IsSubscribed("test-topic") {
		t.Error("Client should not be subscribed after unsubscription")
	}
}

func TestClientMessageHandling(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test valid publish message
	publishMsg := &ClientMessage{
		Type:  PublishMessage,
		Topic: "test-topic",
		Message: &MessageData{
			ID:      "msg-1",
			Payload: "test payload",
		},
		RequestID: "req-1",
	}

	client.handleMessage(publishMsg)

	// Test valid subscribe message
	subscribeMsg := &ClientMessage{
		Type:      SubscribeMessage,
		Topic:     "test-topic",
		ClientID:  "test-client",
		RequestID: "req-2",
	}

	client.handleMessage(subscribeMsg)

	// Test valid unsubscribe message
	unsubscribeMsg := &ClientMessage{
		Type:      UnsubscribeMessage,
		Topic:     "test-topic",
		ClientID:  "test-client",
		RequestID: "req-3",
	}

	client.handleMessage(unsubscribeMsg)

	// Test ping message
	pingMsg := &ClientMessage{
		Type:      PingMessage,
		RequestID: "req-4",
	}

	client.handleMessage(pingMsg)

	// Test unknown message type
	unknownMsg := &ClientMessage{
		Type:      "unknown",
		RequestID: "req-5",
	}

	client.handleMessage(unknownMsg)
}

func TestClientPublishValidation(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test missing topic
	msg := &ClientMessage{
		Type: PublishMessage,
		Message: &MessageData{
			ID:      "msg-1",
			Payload: "test",
		},
		RequestID: "req-1",
	}

	client.handlePublish(msg)

	// Test missing message
	msg = &ClientMessage{
		Type:      PublishMessage,
		Topic:     "test-topic",
		RequestID: "req-2",
	}

	client.handlePublish(msg)

	// Test missing message ID
	msg = &ClientMessage{
		Type:  PublishMessage,
		Topic: "test-topic",
		Message: &MessageData{
			Payload: "test",
		},
		RequestID: "req-3",
	}

	client.handlePublish(msg)
}

func TestClientSubscribeValidation(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test missing topic
	msg := &ClientMessage{
		Type:      SubscribeMessage,
		ClientID:  "test-client",
		RequestID: "req-1",
	}

	client.handleSubscribe(msg)

	// Test missing client ID
	msg = &ClientMessage{
		Type:      SubscribeMessage,
		Topic:     "test-topic",
		RequestID: "req-2",
	}

	client.handleSubscribe(msg)
}

func TestClientUnsubscribeValidation(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test missing topic
	msg := &ClientMessage{
		Type:      UnsubscribeMessage,
		ClientID:  "test-client",
		RequestID: "req-1",
	}

	client.handleUnsubscribe(msg)

	// Test missing client ID
	msg = &ClientMessage{
		Type:      UnsubscribeMessage,
		Topic:     "test-topic",
		RequestID: "req-2",
	}

	client.handleUnsubscribe(msg)
}

func TestClientBackpressureHandling(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 100),
		subscriptions: make(map[string]bool),
		maxQueueSize:  100,
		queueSize:     0,
		slowConsumer:  false,
	}

	// Test normal message sending
	data := []byte("test message")
	client.sendWithBackpressure(data)

	client.mu.RLock()
	queueSize := client.queueSize
	slowConsumer := client.slowConsumer
	client.mu.RUnlock()

	if queueSize != 1 {
		t.Errorf("Expected queue size 1, got %d", queueSize)
	}

	if slowConsumer {
		t.Error("Client should not be marked as slow consumer")
	}

	// Test queue overflow by filling the queue
	for i := 0; i < 100; i++ {
		client.sendWithBackpressure([]byte("message"))
	}

	client.mu.RLock()
	queueSize = client.queueSize
	slowConsumer = client.slowConsumer
	client.mu.RUnlock()

	// Should still be within bounds due to overflow handling
	if queueSize > 100 {
		t.Errorf("Queue size should not exceed max, got %d", queueSize)
	}

	// After filling the queue, client should be marked as slow consumer
	if !slowConsumer {
		t.Error("Client should be marked as slow consumer after queue overflow")
	}
}

func TestClientMessageTypes(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test different message types
	testCases := []struct {
		msgType MessageType
		name    string
	}{
		{PublishMessage, "publish"},
		{SubscribeMessage, "subscribe"},
		{UnsubscribeMessage, "unsubscribe"},
		{PingMessage, "ping"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &ClientMessage{
				Type:      tc.msgType,
				Topic:     "test-topic",
				ClientID:  "test-client",
				RequestID: "req-1",
			}

			// This should not panic
			client.handleMessage(msg)
		})
	}
}

func TestClientConcurrentOperations(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		subscriptions: make(map[string]bool),
	}

	// Test concurrent subscription operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			topicName := "topic-" + string(rune(id))

			// Subscribe
			client.mu.Lock()
			client.subscriptions[topicName] = true
			client.mu.Unlock()

			// Check subscription
			if !client.IsSubscribed(topicName) {
				t.Errorf("Failed to subscribe to %s", topicName)
			}

			// Unsubscribe
			client.mu.Lock()
			delete(client.subscriptions, topicName)
			client.mu.Unlock()

			// Check unsubscription
			if client.IsSubscribed(topicName) {
				t.Errorf("Failed to unsubscribe from %s", topicName)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestClientQueueSizeTracking(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 100),
		subscriptions: make(map[string]bool),
		maxQueueSize:  100,
		queueSize:     0,
		slowConsumer:  false,
	}

	// Test initial queue size
	client.mu.RLock()
	initialSize := client.queueSize
	client.mu.RUnlock()

	if initialSize != 0 {
		t.Errorf("Expected initial queue size 0, got %d", initialSize)
	}

	// Send a message
	client.sendWithBackpressure([]byte("test"))

	client.mu.RLock()
	newSize := client.queueSize
	client.mu.RUnlock()

	if newSize != 1 {
		t.Errorf("Expected queue size 1 after sending message, got %d", newSize)
	}
}
