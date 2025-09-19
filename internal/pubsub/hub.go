package pubsub

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Hub maintains active clients and handles pub/sub operations
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Topic subscriptions: topic -> set of clients
	subscriptions map[string]map[*Client]bool

	// Available topics
	topics map[string]*Topic

	// Channel for new client registrations
	Register chan *Client

	// Channel for client unregistrations
	unregister chan *Client

	// Channel for publishing messages
	publish chan *PubSubMessage

	// Channel for subscribing to topics
	subscribe chan *Subscription

	// Channel for unsubscribing from topics
	unsubscribe chan *Subscription

	// Graceful shutdown
	shutdown     chan struct{}
	shuttingDown bool

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Statistics
	stats Stats
}

// Subscription represents a client subscribing to a topic
type Subscription struct {
	client *Client
	topic  string
}

// Topic represents a pub/sub topic
type Topic struct {
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"created_at"`
	MessageCount    int64     `json:"message_count"`
	SubscriberCount int       `json:"subscriber_count"`
	// Ring buffer for replay (last 100 messages)
	RecentMessages []*PubSubMessage `json:"-"`
	RingHead       int              `json:"-"` // Head of ring buffer
	RingSize       int              `json:"-"` // Current size of ring buffer
}

// Stats holds system statistics
type Stats struct {
	TotalClients  int           `json:"total_clients"`
	TotalTopics   int           `json:"total_topics"`
	TotalMessages int64         `json:"total_messages"`
	ActiveTopics  int           `json:"active_topics"`
	Uptime        time.Duration `json:"uptime"`
	startTime     time.Time
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),
		topics:        make(map[string]*Topic),
		Register:      make(chan *Client),
		unregister:    make(chan *Client),
		publish:       make(chan *PubSubMessage),
		subscribe:     make(chan *Subscription),
		unsubscribe:   make(chan *Subscription),
		shutdown:      make(chan struct{}),
		shuttingDown:  false,
		stats: Stats{
			startTime: time.Now(),
		},
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.publish:
			h.publishMessage(message)

		case subscription := <-h.subscribe:
			h.subscribeClient(subscription)

		case subscription := <-h.unsubscribe:
			h.unsubscribeClient(subscription)

		case <-h.shutdown:
			h.gracefulShutdown()
			return
		}
	}
}

// Shutdown initiates graceful shutdown
func (h *Hub) Shutdown() {
	h.mu.Lock()
	h.shuttingDown = true
	h.mu.Unlock()

	close(h.shutdown)
}

// gracefulShutdown performs graceful shutdown
func (h *Hub) gracefulShutdown() {
	log.Println("Starting graceful shutdown...")

	// Stop accepting new operations
	h.mu.Lock()
	h.shuttingDown = true
	h.mu.Unlock()

	// Best-effort flush: give clients time to process remaining messages
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			log.Println("Shutdown timeout reached, forcing close")
			h.forceCloseAllClients()
			return
		case <-ticker.C:
			if h.allClientsFlushed() {
				log.Println("All clients flushed, closing connections")
				h.forceCloseAllClients()
				return
			}
		}
	}
}

// allClientsFlushed checks if all clients have empty queues
func (h *Hub) allClientsFlushed() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		if client.queueSize > 0 {
			client.mu.RUnlock()
			return false
		}
		client.mu.RUnlock()
	}
	return true
}

// forceCloseAllClients closes all client connections
func (h *Hub) forceCloseAllClients() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.conn.Close()
	}
}

// registerClient adds a new client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Reject new clients during shutdown
	if h.shuttingDown {
		client.conn.Close()
		return
	}

	h.clients[client] = true
	h.stats.TotalClients = len(h.clients)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove client from all topic subscriptions
		for topic, clients := range h.subscriptions {
			if _, exists := clients[client]; exists {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.subscriptions, topic)
				}
				// Update subscriber count
				if topicInfo, exists := h.topics[topic]; exists {
					topicInfo.SubscriberCount = len(clients)
				}
			}
		}

		h.stats.TotalClients = len(h.clients)
	}
}

// publishMessage publishes a message to all subscribers of a topic
func (h *Hub) publishMessage(message *PubSubMessage) {
	h.mu.RLock()
	subscribers, exists := h.subscriptions[message.Topic]
	if !exists {
		h.mu.RUnlock()
		return
	}

	// Update message count and store recent message in ring buffer
	if topic, exists := h.topics[message.Topic]; exists {
		topic.MessageCount++
		// Store in ring buffer
		topic.RecentMessages[topic.RingHead] = message
		topic.RingHead = (topic.RingHead + 1) % 100
		if topic.RingSize < 100 {
			topic.RingSize++
		}
	}
	h.stats.TotalMessages++

	// Create a copy of subscribers to avoid holding the lock while sending
	clientList := make([]*Client, 0, len(subscribers))
	for client := range subscribers {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	// Send message to all subscribers
	for _, client := range clientList {
		select {
		case client.send <- h.createEventMessageBytes(message):
		default:
			// Client's send buffer is full, skip
		}
	}
}

// subscribeClient subscribes a client to a topic
func (h *Hub) subscribeClient(subscription *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscriptions[subscription.topic] == nil {
		h.subscriptions[subscription.topic] = make(map[*Client]bool)
	}
	h.subscriptions[subscription.topic][subscription.client] = true

	// Update subscriber count
	if topic, exists := h.topics[subscription.topic]; exists {
		topic.SubscriberCount = len(h.subscriptions[subscription.topic])
	}
}

// GetRecentMessages returns recent messages for a topic from ring buffer
func (h *Hub) GetRecentMessages(topicName string, lastN int) []*PubSubMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if topic, exists := h.topics[topicName]; exists {
		if lastN <= 0 || lastN > topic.RingSize {
			lastN = topic.RingSize
		}

		if lastN > 0 {
			messages := make([]*PubSubMessage, 0, lastN)

			// Calculate start position in ring buffer
			start := (topic.RingHead - lastN + 100) % 100

			for i := 0; i < lastN; i++ {
				pos := (start + i) % 100
				if topic.RecentMessages[pos] != nil {
					messages = append(messages, topic.RecentMessages[pos])
				}
			}

			return messages
		}
	}
	return []*PubSubMessage{}
}

// unsubscribeClient unsubscribes a client from a topic
func (h *Hub) unsubscribeClient(subscription *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, exists := h.subscriptions[subscription.topic]; exists {
		delete(clients, subscription.client)
		if len(clients) == 0 {
			delete(h.subscriptions, subscription.topic)
		}

		// Update subscriber count
		if topic, exists := h.topics[subscription.topic]; exists {
			topic.SubscriberCount = len(clients)
		}
	}
}

// CreateTopic creates a new topic
func (h *Hub) CreateTopic(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.topics[name]; exists {
		return ErrTopicExists
	}

	h.topics[name] = &Topic{
		Name:            name,
		CreatedAt:       time.Now(),
		MessageCount:    0,
		SubscriberCount: 0,
		RecentMessages:  make([]*PubSubMessage, 100), // Ring buffer of 100 messages
		RingHead:        0,
		RingSize:        0,
	}

	h.stats.TotalTopics = len(h.topics)
	return nil
}

// DeleteTopic removes a topic
func (h *Hub) DeleteTopic(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.topics[name]; !exists {
		return ErrTopicNotFound
	}

	delete(h.topics, name)
	delete(h.subscriptions, name)
	h.stats.TotalTopics = len(h.topics)
	return nil
}

// GetTopics returns all topics
func (h *Hub) GetTopics() map[string]*Topic {
	h.mu.RLock()
	defer h.mu.RUnlock()

	topics := make(map[string]*Topic)
	for name, topic := range h.topics {
		topics[name] = &Topic{
			Name:            topic.Name,
			CreatedAt:       topic.CreatedAt,
			MessageCount:    topic.MessageCount,
			SubscriberCount: topic.SubscriberCount,
		}
	}
	return topics
}

// GetStats returns system statistics
func (h *Hub) GetStats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := h.stats
	stats.Uptime = time.Since(h.stats.startTime)
	stats.ActiveTopics = len(h.subscriptions)
	return stats
}

// createEventMessageBytes converts a PubSubMessage to event JSON bytes
func (h *Hub) createEventMessageBytes(message *PubSubMessage) []byte {
	msg := ServerMessage{
		Type:    EventMessage,
		Topic:   message.Topic,
		Message: message.Message,
		TS:      message.Timestamp.Format(time.RFC3339),
	}

	data, _ := json.Marshal(msg)
	return data
}

// createAckMessageBytes creates an acknowledgment message
func (h *Hub) createAckMessageBytes(requestID, topic, status string) []byte {
	msg := ServerMessage{
		Type:      AckMessage,
		RequestID: requestID,
		Topic:     topic,
		Status:    status,
		TS:        time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(msg)
	return data
}

// createErrorMessageBytes creates an error message
func (h *Hub) createErrorMessageBytes(requestID string, errorCode, errorMsg string) []byte {
	msg := ServerMessage{
		Type:      ErrorMessage,
		RequestID: requestID,
		Error: &ErrorData{
			Code:    errorCode,
			Message: errorMsg,
		},
		TS: time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(msg)
	return data
}

// createPongMessageBytes creates a pong message
func (h *Hub) createPongMessageBytes(requestID string) []byte {
	msg := ServerMessage{
		Type:      PongMessage,
		RequestID: requestID,
		TS:        time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(msg)
	return data
}

// Error definitions
var (
	ErrTopicExists   = fmt.Errorf("topic already exists")
	ErrTopicNotFound = fmt.Errorf("topic not found")
)
