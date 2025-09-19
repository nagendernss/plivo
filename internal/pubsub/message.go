package pubsub

import "time"

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	// Client to Server
	PublishMessage     MessageType = "publish"
	SubscribeMessage   MessageType = "subscribe"
	UnsubscribeMessage MessageType = "unsubscribe"
	PingMessage        MessageType = "ping"

	// Server to Client
	AckMessage   MessageType = "ack"
	EventMessage MessageType = "event"
	ErrorMessage MessageType = "error"
	PongMessage  MessageType = "pong"
	InfoMessage  MessageType = "info"
)

// ClientMessage represents incoming WebSocket messages from clients
type ClientMessage struct {
	Type      MessageType  `json:"type"`
	Topic     string       `json:"topic,omitempty"`
	Message   *MessageData `json:"message,omitempty"`
	ClientID  string       `json:"client_id,omitempty"`
	LastN     int          `json:"last_n,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
}

// MessageData represents the message payload structure
type MessageData struct {
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

// ServerMessage represents outgoing WebSocket messages to clients
type ServerMessage struct {
	Type      MessageType  `json:"type"`
	RequestID string       `json:"request_id,omitempty"`
	Topic     string       `json:"topic,omitempty"`
	Message   *MessageData `json:"message,omitempty"`
	Error     *ErrorData   `json:"error,omitempty"`
	Status    string       `json:"status,omitempty"`
	Msg       string       `json:"msg,omitempty"`
	TS        string       `json:"ts"`
}

// ErrorData represents error information
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PubSubMessage represents a message being published to a topic
type PubSubMessage struct {
	Topic     string       `json:"topic"`
	Message   *MessageData `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
}
