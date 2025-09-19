package pubsub

import (
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map is nil")
	}

	if hub.subscriptions == nil {
		t.Error("subscriptions map is nil")
	}

	if hub.topics == nil {
		t.Error("topics map is nil")
	}

	if hub.Register == nil {
		t.Error("Register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}

	if hub.publish == nil {
		t.Error("publish channel is nil")
	}

	if hub.subscribe == nil {
		t.Error("subscribe channel is nil")
	}

	if hub.unsubscribe == nil {
		t.Error("unsubscribe channel is nil")
	}

	if hub.shutdown == nil {
		t.Error("shutdown channel is nil")
	}

	if hub.shuttingDown != false {
		t.Error("shuttingDown should be false initially")
	}
}

func TestCreateTopic(t *testing.T) {
	hub := NewHub()

	// Test creating a new topic
	err := hub.CreateTopic("test-topic")
	if err != nil {
		t.Errorf("CreateTopic failed: %v", err)
	}

	// Verify topic was created
	hub.mu.RLock()
	topic, exists := hub.topics["test-topic"]
	hub.mu.RUnlock()

	if !exists {
		t.Error("Topic was not created")
	}

	if topic.Name != "test-topic" {
		t.Errorf("Expected topic name 'test-topic', got '%s'", topic.Name)
	}

	if topic.MessageCount != 0 {
		t.Errorf("Expected message count 0, got %d", topic.MessageCount)
	}

	if topic.SubscriberCount != 0 {
		t.Errorf("Expected subscriber count 0, got %d", topic.SubscriberCount)
	}

	// Test creating duplicate topic
	err = hub.CreateTopic("test-topic")
	if err == nil {
		t.Error("Expected error when creating duplicate topic")
	}
}

func TestDeleteTopic(t *testing.T) {
	hub := NewHub()

	// Create a topic first
	hub.CreateTopic("test-topic")

	// Test deleting existing topic
	err := hub.DeleteTopic("test-topic")
	if err != nil {
		t.Errorf("DeleteTopic failed: %v", err)
	}

	// Verify topic was deleted
	hub.mu.RLock()
	_, exists := hub.topics["test-topic"]
	hub.mu.RUnlock()

	if exists {
		t.Error("Topic was not deleted")
	}

	// Test deleting non-existent topic
	err = hub.DeleteTopic("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent topic")
	}
}

func TestGetRecentMessages(t *testing.T) {
	hub := NewHub()

	// Create a topic
	hub.CreateTopic("test-topic")

	// Test getting messages from non-existent topic
	messages := hub.GetRecentMessages("non-existent", 5)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for non-existent topic, got %d", len(messages))
	}

	// Test getting messages from empty topic
	messages = hub.GetRecentMessages("test-topic", 5)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for empty topic, got %d", len(messages))
	}

	// Add some messages to the topic
	hub.mu.Lock()
	topic := hub.topics["test-topic"]
	hub.mu.Unlock()

	// Simulate adding messages to ring buffer
	for i := 0; i < 3; i++ {
		msg := &PubSubMessage{
			Topic: "test-topic",
			Message: &MessageData{
				ID:      "msg-" + string(rune(i)),
				Payload: "test payload",
			},
			Timestamp: time.Now(),
		}
		topic.RecentMessages[i] = msg
		topic.RingSize++
	}

	// Test getting recent messages
	messages = hub.GetRecentMessages("test-topic", 2)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Test getting more messages than available
	messages = hub.GetRecentMessages("test-topic", 10)
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}
}

func TestGetStats(t *testing.T) {
	hub := NewHub()

	// Create some topics
	hub.CreateTopic("topic1")
	hub.CreateTopic("topic2")

	// Get initial stats
	stats := hub.GetStats()

	if stats.TotalTopics != 2 {
		t.Errorf("Expected 2 topics, got %d", stats.TotalTopics)
	}

	if stats.ActiveTopics != 0 {
		t.Errorf("Expected 0 active topics, got %d", stats.ActiveTopics)
	}

	if stats.TotalClients != 0 {
		t.Errorf("Expected 0 clients, got %d", stats.TotalClients)
	}

	if stats.TotalMessages != 0 {
		t.Errorf("Expected 0 messages, got %d", stats.TotalMessages)
	}

	if stats.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}
}

func TestShutdown(t *testing.T) {
	hub := NewHub()

	// Test initial state
	if hub.shuttingDown != false {
		t.Error("shuttingDown should be false initially")
	}

	// Start hub in goroutine
	go hub.Run()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test shutdown
	hub.Shutdown()

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	shuttingDown := hub.shuttingDown
	hub.mu.RUnlock()

	if shuttingDown != true {
		t.Error("shuttingDown should be true after Shutdown()")
	}
}

func TestTopicIsolation(t *testing.T) {
	hub := NewHub()

	// Create two topics
	hub.CreateTopic("topic1")
	hub.CreateTopic("topic2")

	// Verify topics are separate
	hub.mu.RLock()
	topic1, exists1 := hub.topics["topic1"]
	topic2, exists2 := hub.topics["topic2"]
	hub.mu.RUnlock()

	if !exists1 || !exists2 {
		t.Error("Both topics should exist")
	}

	if topic1 == topic2 {
		t.Error("Topics should be separate instances")
	}

	// Verify separate subscription maps
	hub.mu.RLock()
	subs1, exists1 := hub.subscriptions["topic1"]
	subs2, exists2 := hub.subscriptions["topic2"]
	hub.mu.RUnlock()

	if !exists1 || !exists2 {
		t.Error("Both subscription maps should exist")
	}

	// Verify subscription maps are separate (can't compare maps directly)
	if len(subs1) != 0 || len(subs2) != 0 {
		t.Error("Subscription maps should be empty initially")
	}
}

func TestConcurrentTopicOperations(t *testing.T) {
	hub := NewHub()

	// Test concurrent topic creation
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			topicName := "topic-" + string(rune(id))
			err := hub.CreateTopic(topicName)
			if err != nil {
				t.Errorf("Failed to create topic %s: %v", topicName, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all topics were created
	hub.mu.RLock()
	topicCount := len(hub.topics)
	hub.mu.RUnlock()

	if topicCount != 10 {
		t.Errorf("Expected 10 topics, got %d", topicCount)
	}
}

func TestMessageCountTracking(t *testing.T) {
	hub := NewHub()
	hub.CreateTopic("test-topic")

	// Simulate message publishing
	hub.mu.Lock()
	hub.stats.TotalMessages = 5
	hub.topics["test-topic"].MessageCount = 3
	hub.mu.Unlock()

	stats := hub.GetStats()
	if stats.TotalMessages != 5 {
		t.Errorf("Expected 5 total messages, got %d", stats.TotalMessages)
	}

	hub.mu.RLock()
	topic := hub.topics["test-topic"]
	hub.mu.RUnlock()

	if topic.MessageCount != 3 {
		t.Errorf("Expected 3 topic messages, got %d", topic.MessageCount)
	}
}
