package pubsub

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageTypes(t *testing.T) {
	// Test client message types
	clientTypes := []MessageType{
		PublishMessage,
		SubscribeMessage,
		UnsubscribeMessage,
		PingMessage,
	}

	for _, msgType := range clientTypes {
		if msgType == "" {
			t.Error("Message type should not be empty")
		}
	}

	// Test server message types
	serverTypes := []MessageType{
		AckMessage,
		EventMessage,
		ErrorMessage,
		PongMessage,
		InfoMessage,
	}

	for _, msgType := range serverTypes {
		if msgType == "" {
			t.Error("Message type should not be empty")
		}
	}
}

func TestClientMessageSerialization(t *testing.T) {
	// Test valid client message
	msg := ClientMessage{
		Type:      PublishMessage,
		Topic:     "test-topic",
		Message:   &MessageData{ID: "msg-1", Payload: "test payload"},
		ClientID:  "client-1",
		LastN:     5,
		RequestID: "req-1",
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal ClientMessage: %v", err)
	}

	// Deserialize
	var deserialized ClientMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal ClientMessage: %v", err)
	}

	// Verify fields
	if deserialized.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, deserialized.Type)
	}

	if deserialized.Topic != msg.Topic {
		t.Errorf("Expected topic %s, got %s", msg.Topic, deserialized.Topic)
	}

	if deserialized.ClientID != msg.ClientID {
		t.Errorf("Expected client ID %s, got %s", msg.ClientID, deserialized.ClientID)
	}

	if deserialized.LastN != msg.LastN {
		t.Errorf("Expected LastN %d, got %d", msg.LastN, deserialized.LastN)
	}

	if deserialized.RequestID != msg.RequestID {
		t.Errorf("Expected request ID %s, got %s", msg.RequestID, deserialized.RequestID)
	}

	if deserialized.Message == nil {
		t.Error("Message should not be nil")
	} else {
		if deserialized.Message.ID != msg.Message.ID {
			t.Errorf("Expected message ID %s, got %s", msg.Message.ID, deserialized.Message.ID)
		}

		if deserialized.Message.Payload != msg.Message.Payload {
			t.Errorf("Expected payload %v, got %v", msg.Message.Payload, deserialized.Message.Payload)
		}
	}
}

func TestServerMessageSerialization(t *testing.T) {
	// Test valid server message
	msg := ServerMessage{
		Type:      AckMessage,
		RequestID: "req-1",
		Topic:     "test-topic",
		Status:    "ok",
		TS:        time.Now().Format(time.RFC3339),
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal ServerMessage: %v", err)
	}

	// Deserialize
	var deserialized ServerMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal ServerMessage: %v", err)
	}

	// Verify fields
	if deserialized.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, deserialized.Type)
	}

	if deserialized.RequestID != msg.RequestID {
		t.Errorf("Expected request ID %s, got %s", msg.RequestID, deserialized.RequestID)
	}

	if deserialized.Topic != msg.Topic {
		t.Errorf("Expected topic %s, got %s", msg.Topic, deserialized.Topic)
	}

	if deserialized.Status != msg.Status {
		t.Errorf("Expected status %s, got %s", msg.Status, deserialized.Status)
	}

	if deserialized.TS != msg.TS {
		t.Errorf("Expected timestamp %s, got %s", msg.TS, deserialized.TS)
	}
}

func TestErrorMessageSerialization(t *testing.T) {
	// Test error message
	msg := ServerMessage{
		Type:      ErrorMessage,
		RequestID: "req-1",
		Error: &ErrorData{
			Code:    "BAD_REQUEST",
			Message: "Invalid message format",
		},
		TS: time.Now().Format(time.RFC3339),
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal error message: %v", err)
	}

	// Deserialize
	var deserialized ServerMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal error message: %v", err)
	}

	// Verify error fields
	if deserialized.Error == nil {
		t.Error("Error should not be nil")
	} else {
		if deserialized.Error.Code != msg.Error.Code {
			t.Errorf("Expected error code %s, got %s", msg.Error.Code, deserialized.Error.Code)
		}

		if deserialized.Error.Message != msg.Error.Message {
			t.Errorf("Expected error message %s, got %s", msg.Error.Message, deserialized.Error.Message)
		}
	}
}

func TestEventMessageSerialization(t *testing.T) {
	// Test event message
	msg := ServerMessage{
		Type:  EventMessage,
		Topic: "test-topic",
		Message: &MessageData{
			ID:      "msg-1",
			Payload: map[string]interface{}{"key": "value"},
		},
		TS: time.Now().Format(time.RFC3339),
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal event message: %v", err)
	}

	// Deserialize
	var deserialized ServerMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal event message: %v", err)
	}

	// Verify message fields
	if deserialized.Message == nil {
		t.Error("Message should not be nil")
	} else {
		if deserialized.Message.ID != msg.Message.ID {
			t.Errorf("Expected message ID %s, got %s", msg.Message.ID, deserialized.Message.ID)
		}
	}
}

func TestPubSubMessageSerialization(t *testing.T) {
	// Test pub/sub message
	msg := PubSubMessage{
		Topic: "test-topic",
		Message: &MessageData{
			ID:      "msg-1",
			Payload: "test payload",
		},
		Timestamp: time.Now(),
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal PubSubMessage: %v", err)
	}

	// Deserialize
	var deserialized PubSubMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal PubSubMessage: %v", err)
	}

	// Verify fields
	if deserialized.Topic != msg.Topic {
		t.Errorf("Expected topic %s, got %s", msg.Topic, deserialized.Topic)
	}

	if deserialized.Message == nil {
		t.Error("Message should not be nil")
	} else {
		if deserialized.Message.ID != msg.Message.ID {
			t.Errorf("Expected message ID %s, got %s", msg.Message.ID, deserialized.Message.ID)
		}

		if deserialized.Message.Payload != msg.Message.Payload {
			t.Errorf("Expected payload %v, got %v", msg.Message.Payload, deserialized.Message.Payload)
		}
	}
}

func TestMessageDataWithComplexPayload(t *testing.T) {
	// Test with complex payload
	complexPayload := map[string]interface{}{
		"order_id": "ORD-123",
		"amount":   99.50,
		"items":    []string{"item1", "item2"},
		"metadata": map[string]interface{}{
			"source":  "web",
			"user_id": 12345,
		},
	}

	msg := MessageData{
		ID:      "msg-1",
		Payload: complexPayload,
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal complex payload: %v", err)
	}

	// Deserialize
	var deserialized MessageData
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal complex payload: %v", err)
	}

	// Verify ID
	if deserialized.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, deserialized.ID)
	}

	// Verify payload structure
	deserializedPayload, ok := deserialized.Payload.(map[string]interface{})
	if !ok {
		t.Error("Payload should be a map")
	}

	if deserializedPayload["order_id"] != "ORD-123" {
		t.Error("Order ID should be preserved")
	}

	if deserializedPayload["amount"] != 99.50 {
		t.Error("Amount should be preserved")
	}
}

func TestMessageValidation(t *testing.T) {
	// Test empty message
	msg := ClientMessage{}

	if msg.Type != "" {
		t.Error("Empty message should have empty type")
	}

	// Test message with only required fields
	msg = ClientMessage{
		Type:  PublishMessage,
		Topic: "test-topic",
		Message: &MessageData{
			ID: "msg-1",
		},
	}

	if msg.Type != PublishMessage {
		t.Error("Type should be set correctly")
	}

	if msg.Topic != "test-topic" {
		t.Error("Topic should be set correctly")
	}

	if msg.Message == nil {
		t.Error("Message should not be nil")
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Test that constants are properly defined
	expectedTypes := map[string]MessageType{
		"publish":     PublishMessage,
		"subscribe":   SubscribeMessage,
		"unsubscribe": UnsubscribeMessage,
		"ping":        PingMessage,
		"ack":         AckMessage,
		"event":       EventMessage,
		"error":       ErrorMessage,
		"pong":        PongMessage,
		"info":        InfoMessage,
	}

	for name, msgType := range expectedTypes {
		if msgType == "" {
			t.Errorf("Message type %s should not be empty", name)
		}

		if string(msgType) != name {
			t.Errorf("Expected message type %s, got %s", name, string(msgType))
		}
	}
}

func TestMessageWithNilFields(t *testing.T) {
	// Test message with nil optional fields
	msg := ClientMessage{
		Type:      PublishMessage,
		Topic:     "test-topic",
		Message:   nil, // This should be handled gracefully
		ClientID:  "",
		LastN:     0,
		RequestID: "",
	}

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal message with nil fields: %v", err)
	}

	// Deserialize
	var deserialized ClientMessage
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Errorf("Failed to unmarshal message with nil fields: %v", err)
	}

	// Verify that nil fields are handled
	if deserialized.Message != nil {
		t.Error("Nil message should remain nil")
	}
}

func TestMessageTimestampFormat(t *testing.T) {
	// Test timestamp formatting
	now := time.Now()
	msg := ServerMessage{
		TS: now.Format(time.RFC3339),
	}

	// Parse the timestamp back
	parsedTime, err := time.Parse(time.RFC3339, msg.TS)
	if err != nil {
		t.Errorf("Failed to parse timestamp: %v", err)
	}

	// Check that it's close to the original time (within 1 second)
	diff := now.Sub(parsedTime)
	if diff < 0 {
		diff = -diff
	}

	if diff > time.Second {
		t.Error("Timestamp should be accurate within 1 second")
	}
}
