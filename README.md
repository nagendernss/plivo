# In-Memory Pub/Sub System

A simplified in-memory Pub/Sub system built in Go with WebSocket and REST API support.

## Features

- **WebSocket Endpoint** (`/ws`): Real-time publish/subscribe operations
- **REST API**: Topic management and system observability
- **Thread-Safe**: Handles multiple publishers and subscribers safely
- **In-Memory Only**: No external dependencies or persistence
- **Containerized**: Docker support for easy deployment

## Architecture

### Core Components

1. **Hub**: Central message broker managing clients and topic subscriptions
2. **Client**: WebSocket connection handler with subscription management
3. **Message**: Structured message format for all communications
4. **REST Handlers**: HTTP endpoints for management operations

### Concurrency Model

- Uses Go channels for thread-safe communication between goroutines
- RWMutex for protecting shared data structures
- Each WebSocket connection runs in separate goroutines for reading and writing
- Hub runs in a single goroutine to avoid race conditions

### Design Choices

#### Backpressure Policy
- **Bounded Queues**: Each client has a bounded queue of 100 messages
- **Overflow Handling**: When queue is full, oldest message is dropped and new message is added
- **Slow Consumer Detection**: If dropping messages fails, client is marked as slow consumer
- **Automatic Disconnection**: Slow consumers receive SLOW_CONSUMER error and are disconnected
- **Queue Monitoring**: Real-time tracking of queue sizes for monitoring

#### Memory Management
- **Ring Buffer**: Each topic maintains a ring buffer of last 100 messages for replay
- **Topic Cleanup**: Topics are automatically removed when no subscribers remain
- **Client Cleanup**: Resources are freed when clients disconnect
- **No Persistence**: All state is lost on restart (as required)

#### Graceful Shutdown
- **Signal Handling**: Responds to SIGINT and SIGTERM signals
- **Best-Effort Flush**: Waits up to 5 seconds for clients to process remaining messages
- **Connection Closure**: All WebSocket connections are closed cleanly
- **Resource Cleanup**: All goroutines and channels are properly cleaned up

#### Authentication
- **X-API-Key**: Optional authentication via X-API-Key header
- **Environment Variable**: API key configured via API_KEY environment variable
- **Flexible**: If no API key is set, all requests are allowed
- **REST & WebSocket**: Authentication applies to both REST and WebSocket endpoints

#### Scalability Considerations
- **Vertical Scaling Only**: Single-process, in-memory design
- **Connection Limits**: Limited by available memory and file descriptors
- **Message Throughput**: Optimized for concurrent operations using Go channels

## API Reference

### WebSocket Protocol (`/ws`)

#### Client → Server Messages

```json
{
  "type": "subscribe" | "unsubscribe" | "publish" | "ping",
  "topic": "orders", // required for subscribe/unsubscribe/publish
  "message": { // required for publish
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": "..."
  },
  "client_id": "s1", // required for subscribe/unsubscribe
  "last_n": 0, // optional: number of historical messages to replay
  "request_id": "uuid-optional" // optional: correlation id
}
```

#### Server → Client Messages

```json
{
  "type": "ack" | "event" | "error" | "pong" | "info",
  "request_id": "uuid-optional", // echoed if provided
  "topic": "orders",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": "..."
  },
  "error": {
    "code": "BAD_REQUEST",
    "message": "..."
  },
  "ts": "2025-08-25T10:00:00Z" // optional server timestamp
}
```

### REST API Endpoints

#### Topic Management
- `POST /topics` - Create a topic
- `GET /topics` - List all topics  
- `DELETE /topics/{name}` - Delete a topic

#### Observability
- `GET /health` - System health status
- `GET /stats` - Detailed statistics

### Examples

#### WebSocket Subscribe
```json
{
  "type": "subscribe",
  "topic": "orders",
  "client_id": "s1",
  "last_n": 5,
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### WebSocket Publish
```json
{
  "type": "publish",
  "topic": "orders",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": {
      "order_id": "ORD-123",
      "amount": "99.5",
      "currency": "USD"
    }
  },
  "request_id": "340e8400-e29b-41d4-a716-4466554480098"
}
```

#### REST Create Topic
```bash
POST /topics
{
  "name": "orders"
}
```

Response:
```json
{
  "status": "created",
  "topic": "orders"
}
```

#### REST List Topics
```bash
GET /topics
```

Response:
```json
{
  "topics": [
    {
      "name": "orders",
      "subscribers": 3
    }
  ]
}
```

#### REST Health Check
```bash
GET /health
```

Response:
```json
{
  "uptime_sec": 123,
  "topics": 2,
  "subscribers": 4
}
```

#### REST Statistics
```bash
GET /stats
```

Response:
```json
{
  "topics": {
    "orders": {
      "messages": 42,
      "subscribers": 3
    }
  }
}

