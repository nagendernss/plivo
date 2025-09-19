package pubsub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in this demo
	},
}

// Client represents a WebSocket client
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool
	mu            sync.RWMutex
	id            string
	// Backpressure management
	queueSize    int
	maxQueueSize int
	slowConsumer bool
}

// NewClient creates a new client
func NewClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 100), // Reduced buffer size for backpressure
		subscriptions: make(map[string]bool),
		id:            id,
		maxQueueSize:  100,
		queueSize:     0,
		slowConsumer:  false,
	}
}

// ReadPump handles reading messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			c.sendError("", "BAD_REQUEST", "Invalid JSON format")
			continue
		}

		c.handleMessage(&msg)
	}
}

// WritePump handles writing messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// Update queue size after successful send
			c.mu.Lock()
			if c.queueSize > 0 {
				c.queueSize--
			}
			c.mu.Unlock()

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(msg *ClientMessage) {
	switch msg.Type {
	case PublishMessage:
		c.handlePublish(msg)
	case SubscribeMessage:
		c.handleSubscribe(msg)
	case UnsubscribeMessage:
		c.handleUnsubscribe(msg)
	case PingMessage:
		c.handlePing(msg)
	default:
		c.sendError(msg.RequestID, "BAD_REQUEST", "Unknown message type")
	}
}

// handlePublish processes publish requests
func (c *Client) handlePublish(msg *ClientMessage) {
	if msg.Topic == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Topic is required for publish")
		return
	}

	if msg.Message == nil {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Message is required for publish")
		return
	}

	if msg.Message.ID == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Message ID is required")
		return
	}

	c.hub.publish <- &PubSubMessage{
		Topic:     msg.Topic,
		Message:   msg.Message,
		Timestamp: time.Now(),
	}

	// Send acknowledgment
	c.sendAck(msg.RequestID, msg.Topic, "ok")
}

// handleSubscribe processes subscription requests
func (c *Client) handleSubscribe(msg *ClientMessage) {
	if msg.Topic == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Topic is required for subscribe")
		return
	}

	if msg.ClientID == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Client ID is required for subscribe")
		return
	}

	c.mu.Lock()
	c.subscriptions[msg.Topic] = true
	c.mu.Unlock()

	c.hub.subscribe <- &Subscription{
		client: c,
		topic:  msg.Topic,
	}

	// Send historical messages if requested
	if msg.LastN > 0 {
		recentMessages := c.hub.GetRecentMessages(msg.Topic, msg.LastN)
		for _, recentMsg := range recentMessages {
			c.sendEvent(recentMsg)
		}
	}

	// Send acknowledgment
	c.sendAck(msg.RequestID, msg.Topic, "ok")
}

// handleUnsubscribe processes unsubscription requests
func (c *Client) handleUnsubscribe(msg *ClientMessage) {
	if msg.Topic == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Topic is required for unsubscribe")
		return
	}

	if msg.ClientID == "" {
		c.sendError(msg.RequestID, "BAD_REQUEST", "Client ID is required for unsubscribe")
		return
	}

	c.mu.Lock()
	delete(c.subscriptions, msg.Topic)
	c.mu.Unlock()

	c.hub.unsubscribe <- &Subscription{
		client: c,
		topic:  msg.Topic,
	}

	// Send acknowledgment
	c.sendAck(msg.RequestID, msg.Topic, "ok")
}

// handlePing responds to ping messages
func (c *Client) handlePing(msg *ClientMessage) {
	c.sendPong(msg.RequestID)
}

// sendMessage sends a message to the client with backpressure handling
func (c *Client) sendMessage(msg *ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	c.sendWithBackpressure(data)
}

// sendWithBackpressure handles message sending with backpressure management
func (c *Client) sendWithBackpressure(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if client is marked as slow consumer
	if c.slowConsumer {
		return
	}

	// Try to send immediately
	select {
	case c.send <- data:
		c.queueSize++
		return
	default:
		// Queue is full, handle overflow
		c.handleQueueOverflow()
	}
}

// handleQueueOverflow handles queue overflow according to policy
func (c *Client) handleQueueOverflow() {
	// Policy: Drop oldest message and add new one
	select {
	case <-c.send: // Remove oldest message
		c.queueSize--
		select {
		case c.send <- <-c.send: // Add new message
			c.queueSize++
		default:
			// Still can't add, mark as slow consumer
			c.slowConsumer = true
			c.sendSlowConsumerError()
		}
	default:
		// Can't remove any message, mark as slow consumer
		c.slowConsumer = true
		c.sendSlowConsumerError()
	}
}

// sendSlowConsumerError sends SLOW_CONSUMER error and disconnects
func (c *Client) sendSlowConsumerError() {
	errorData := c.hub.createErrorMessageBytes("", "SLOW_CONSUMER", "Client queue overflow, disconnecting")
	select {
	case c.send <- errorData:
	default:
		// Can't even send error, force close
	}

	// Schedule disconnection
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for error to be sent
		c.conn.Close()
	}()
}

// sendAck sends an acknowledgment message
func (c *Client) sendAck(requestID, topic, status string) {
	data := c.hub.createAckMessageBytes(requestID, topic, status)
	c.sendWithBackpressure(data)
}

// sendError sends an error message to the client
func (c *Client) sendError(requestID, errorCode, errorMsg string) {
	data := c.hub.createErrorMessageBytes(requestID, errorCode, errorMsg)
	c.sendWithBackpressure(data)
}

// sendPong sends a pong message
func (c *Client) sendPong(requestID string) {
	data := c.hub.createPongMessageBytes(requestID)
	c.sendWithBackpressure(data)
}

// sendEvent sends an event message
func (c *Client) sendEvent(msg *PubSubMessage) {
	data := c.hub.createEventMessageBytes(msg)
	c.sendWithBackpressure(data)
}

// IsSubscribed checks if the client is subscribed to a topic
func (c *Client) IsSubscribed(topic string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscriptions[topic]
}
