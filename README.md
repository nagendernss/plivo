# In-Memory Pub/Sub System

A production-ready in-memory Pub/Sub system built in Go with WebSocket and REST API support, featuring advanced backpressure management, message replay, authentication, and comprehensive monitoring.

## 🚀 Features

### Core Functionality
- **WebSocket Endpoint** (`/ws`): Real-time publish/subscribe operations with full protocol support
- **REST API**: Complete topic management and system observability
- **Thread-Safe**: Handles multiple publishers and subscribers safely with proper concurrency control
- **In-Memory Only**: No external dependencies or persistence (as required)
- **Containerized**: Production-ready Docker support with multi-stage builds

### Advanced Features
- **Backpressure Management**: Sophisticated queue overflow handling with configurable policies
- **Message Replay**: Ring buffer with last 100 messages per topic and `last_n` support
- **Authentication**: Optional X-API-Key authentication for both REST and WebSocket endpoints
- **Graceful Shutdown**: Signal handling with best-effort message flushing
- **Comprehensive Monitoring**: Real-time statistics and health checks
- **Heartbeat Support**: WebSocket ping/pong with automatic connection health monitoring

## 🏗️ Architecture

### Core Components

1. **Hub**: Central message broker managing clients, topic subscriptions, and message routing
2. **Client**: WebSocket connection handler with subscription management and backpressure control
3. **Message**: Structured message format for all communications with proper validation
4. **REST Handlers**: HTTP endpoints for management operations with authentication
5. **WebSocket Handler**: Real-time communication handler with connection management

### Concurrency Model

- **Channel-based Communication**: All operations flow through channels to Hub's single goroutine
- **RWMutex Protection**: Shared data structures protected with read-write mutexes
- **Goroutine Isolation**: Each WebSocket connection runs in separate read/write goroutines
- **Race-free Design**: Hub runs in single goroutine to eliminate race conditions
- **Atomic Operations**: Queue size tracking and statistics with proper synchronization

### Design Choices

#### Backpressure Policy
- **Bounded Queues**: Each client has a bounded queue of 100 messages
- **Overflow Handling**: When queue is full, oldest message is dropped and new message is added
- **Slow Consumer Detection**: If dropping messages fails, client is marked as slow consumer
- **Automatic Disconnection**: Slow consumers receive `SLOW_CONSUMER` error and are disconnected
- **Queue Monitoring**: Real-time tracking of queue sizes for monitoring and alerting

#### Memory Management
- **Ring Buffer**: Each topic maintains a ring buffer of last 100 messages for replay
- **Topic Cleanup**: Topics are automatically removed when no subscribers remain
- **Client Cleanup**: Resources are freed when clients disconnect
- **No Persistence**: All state is lost on restart (as required)
- **Memory Bounds**: Fixed-size buffers prevent memory leaks

#### Graceful Shutdown
- **Signal Handling**: Responds to SIGINT and SIGTERM signals
- **Best-Effort Flush**: Waits up to 5 seconds for clients to process remaining messages
- **Connection Closure**: All WebSocket connections are closed cleanly
- **Resource Cleanup**: All goroutines and channels are properly cleaned up
- **Timeout Protection**: Forces closure if graceful shutdown takes too long

#### Authentication
- **X-API-Key**: Optional authentication via X-API-Key header
- **Environment Variable**: API key configured via `API_KEY` environment variable
- **Flexible**: If no API key is set, all requests are allowed
- **REST & WebSocket**: Authentication applies to both REST and WebSocket endpoints
- **Security**: Proper unauthorized response handling with HTTP 401

#### Scalability Considerations
- **Vertical Scaling Only**: Single-process, in-memory design
- **Connection Limits**: Limited by available memory and file descriptors
- **Message Throughput**: Optimized for concurrent operations using Go channels
- **Performance**: Designed for high-throughput, low-latency message delivery

## 📡 API Reference

### WebSocket Protocol (`/ws`)

#### Client → Server Messages

```json
{
  "type": "subscribe" | "unsubscribe" | "publish" | "ping",
  "topic": "orders", // required for subscribe/unsubscribe/publish
  "message": { // required for publish
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": "..." // any JSON-serializable data
  },
  "client_id": "s1", // required for subscribe/unsubscribe
  "last_n": 0, // optional: number of historical messages to replay (1-100)
  "request_id": "uuid-optional" // optional: correlation id for tracking
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
    "code": "BAD_REQUEST" | "SLOW_CONSUMER",
    "message": "Human-readable error description"
  },
  "status": "ok", // for ack messages
  "ts": "2025-08-25T10:00:00Z" // RFC3339 timestamp
}
```

### REST API Endpoints

#### Topic Management
- `POST /topics` - Create a new topic
- `GET /topics` - List all topics with subscriber counts
- `DELETE /topics/{name}` - Delete a topic and disconnect all subscribers

#### Observability
- `GET /health` - System health status (no auth required)
- `GET /stats` - Detailed system statistics and metrics

#### Authentication
All endpoints (except `/health`) require `X-API-Key` header if `API_KEY` environment variable is set.

## 📚 API Documentation (Swagger)

The API includes comprehensive Swagger/OpenAPI documentation that provides an interactive interface for exploring and testing all endpoints.

### Accessing Swagger Documentation

Once the server is running, you can access the Swagger documentation at:

- **Swagger UI**: `http://localhost:8080/swagger/`
- **OpenAPI JSON Spec**: `http://localhost:8080/swagger/doc.json`

### Features

- **Interactive API Explorer**: Test all REST endpoints directly from the browser
- **Request/Response Examples**: See example payloads and responses for each endpoint
- **Authentication Support**: Test endpoints with API key authentication
- **Real-time Testing**: Make actual API calls and see live responses
- **Schema Validation**: View detailed request/response schemas

### Using Swagger UI

1. **Navigate to** `http://localhost:8080/swagger/` in your browser
2. **Explore Endpoints**: Click on any endpoint to expand its details
3. **Test Endpoints**: Click "Try it out" to test any endpoint
4. **Authentication**: If using API keys, click the "Authorize" button and enter your API key
5. **View Responses**: Execute requests and see real responses from your server

### Available Endpoints in Swagger

- **POST /topics** - Create a new topic
- **GET /topics** - List all topics with subscriber counts  
- **DELETE /topics/{topic}** - Delete a topic and disconnect all subscribers
- **GET /health** - System health status (no authentication required)
- **GET /stats** - Detailed system statistics and metrics

### Example: Testing with Swagger

1. Open `http://localhost:8080/swagger/`
2. Find the "POST /topics" endpoint
3. Click "Try it out"
4. Enter a topic name in the request body:
   ```json
   {
     "name": "test-topic"
   }
   ```
5. Click "Execute" to create the topic
6. View the response and status code

The Swagger documentation is automatically generated from the code comments and annotations in your Go source files, ensuring it stays up-to-date with your API implementation.

## 💡 Usage Examples

### WebSocket Operations

#### Subscribe to Topic with Historical Messages
```json
{
  "type": "subscribe",
  "topic": "orders",
  "client_id": "subscriber-1",
  "last_n": 5,
  "request_id": "sub-001"
}
```

**Response (Acknowledgment):**
```json
{
  "type": "ack",
  "request_id": "sub-001",
  "topic": "orders",
  "status": "ok",
  "ts": "2025-01-15T10:00:00Z"
}
```

**Response (Historical Messages):**
```json
{
  "type": "event",
  "topic": "orders",
  "message": {
    "id": "msg-001",
    "payload": {"order_id": "ORD-123", "amount": 99.50}
  },
  "ts": "2025-01-15T09:59:30Z"
}
```

#### Publish Message
```json
{
  "type": "publish",
  "topic": "orders",
  "message": {
    "id": "msg-002",
    "payload": {
      "order_id": "ORD-124",
      "amount": 149.99,
      "currency": "USD",
      "customer": "john@example.com"
    }
  },
  "request_id": "pub-001"
}
```

**Response:**
```json
{
  "type": "ack",
  "request_id": "pub-001",
  "topic": "orders",
  "status": "ok",
  "ts": "2025-01-15T10:00:00Z"
}
```

#### Unsubscribe from Topic
```json
{
  "type": "unsubscribe",
  "topic": "orders",
  "client_id": "subscriber-1",
  "request_id": "unsub-001"
}
```

#### Ping/Pong Heartbeat
```json
{
  "type": "ping",
  "request_id": "ping-001"
}
```

**Response:**
```json
{
  "type": "pong",
  "request_id": "ping-001",
  "ts": "2025-01-15T10:00:00Z"
}
```

### REST API Operations

#### Create Topic
```bash
curl -X POST http://localhost:8080/topics \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"name": "orders"}'
```

**Response:**
```json
{
  "status": "created",
  "topic": "orders"
}
```

#### List Topics
```bash
curl -X GET http://localhost:8080/topics \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "topics": [
    {
      "name": "orders",
      "created_at": "2025-01-15T10:00:00Z",
      "message_count": 42,
      "subscriber_count": 3
    },
    {
      "name": "notifications",
      "created_at": "2025-01-15T09:30:00Z",
      "message_count": 15,
      "subscriber_count": 1
    }
  ]
}
```

#### Delete Topic
```bash
curl -X DELETE http://localhost:8080/topics/orders \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "status": "deleted",
  "topic": "orders"
}
```

#### Health Check
```bash
curl -X GET http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "uptime_sec": 3600,
  "topics": 2,
  "subscribers": 4,
  "total_messages": 57
}
```

#### System Statistics
```bash
curl -X GET http://localhost:8080/stats \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "total_clients": 4,
  "total_topics": 2,
  "total_messages": 57,
  "active_topics": 2,
  "uptime": "1h0m0s",
  "topics": {
    "orders": {
      "name": "orders",
      "created_at": "2025-01-15T10:00:00Z",
      "message_count": 42,
      "subscriber_count": 3
    },
    "notifications": {
      "name": "notifications",
      "created_at": "2025-01-15T09:30:00Z",
      "message_count": 15,
      "subscriber_count": 1
    }
  }
}
```

## 🐳 Docker Deployment

### Build and Run
```bash
# Build the Docker image
docker build -t plivo-pubsub .

# Run without authentication
docker run -p 8080:8080 plivo-pubsub

# Run with authentication
docker run -p 8080:8080 -e API_KEY=your-secret-key plivo-pubsub
```

### Docker Compose
```yaml
version: '3.8'
services:
  pubsub:
    build: .
    ports:
      - "8080:8080"
    environment:
      - API_KEY=your-secret-key
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
```

## 🔧 Configuration

The system supports comprehensive configuration via command-line flags and environment variables. Command-line flags take precedence over environment variables.

### Command-Line Flags

#### Server Configuration
- `-port`: Server port (default: `8080`)
- `-read-timeout`: HTTP read timeout (default: `10s`)
- `-write-timeout`: HTTP write timeout (default: `10s`)
- `-idle-timeout`: HTTP idle timeout (default: `60s`)
- `-shutdown-timeout`: Graceful shutdown timeout (default: `10s`)

#### Pub/Sub System Configuration
- `-max-queue-size`: Maximum messages per client queue (default: `100`)
- `-ring-buffer-size`: Ring buffer size for message replay (default: `100`)
- `-ping-interval`: WebSocket ping interval (default: `54s`)
- `-pong-wait`: WebSocket pong wait timeout (default: `60s`)
- `-write-wait`: WebSocket write wait timeout (default: `10s`)
- `-max-message-size`: Maximum message size in bytes (default: `1048576` = 1MB)
- `-enable-compression`: Enable WebSocket compression (default: `false`)

#### Security Configuration
- `-api-key`: API key for authentication (default: empty = no auth required)
- `-enable-cors`: Enable CORS support (default: `false`)
- `-allowed-origins`: Comma-separated list of allowed origins (default: `*`)
- `-rate-limit-per-min`: Rate limit per minute (default: `1000`)
- `-rate-limit-burst`: Rate limit burst size (default: `100`)

#### Logging Configuration
- `-log-level`: Log level (debug, info, warn, error) (default: `info`)
- `-log-format`: Log format (text, json) (default: `text`)

#### Other Flags
- `-help`: Show help information
- `-version`: Show version information

### Environment Variables

All command-line flags can also be set via environment variables with the same names in uppercase:

- `PORT`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT`, `SHUTDOWN_TIMEOUT`
- `MAX_QUEUE_SIZE`, `RING_BUFFER_SIZE`, `PING_INTERVAL`, `PONG_WAIT`, `WRITE_WAIT`, `MAX_MESSAGE_SIZE`, `ENABLE_COMPRESSION`
- `API_KEY`, `ENABLE_CORS`, `ALLOWED_ORIGINS`, `RATE_LIMIT_PER_MIN`, `RATE_LIMIT_BURST`
- `LOG_LEVEL`, `LOG_FORMAT`

### Usage Examples

#### Command-Line Configuration
```bash
# Run with custom port and API key
./plivo -port=9090 -api-key=my-secret-key

# Run with custom queue size and compression
./plivo -max-queue-size=200 -enable-compression

# Run with debug logging
./plivo -log-level=debug -log-format=json

# Show help
./plivo -help

# Show version
./plivo -version
```

#### Environment Variable Configuration
```bash
# Set environment variables
export PORT=9090
export API_KEY=my-secret-key
export MAX_QUEUE_SIZE=200
export LOG_LEVEL=debug

# Run the application
./plivo
```

#### Mixed Configuration
```bash
# Environment variables as defaults, command-line flags for overrides
export API_KEY=default-key
export LOG_LEVEL=info

# Override with command-line flags
./plivo -api-key=override-key -log-level=debug
```

### System Limits (Default Values)
- **Message Queue Size**: 100 messages per client
- **Ring Buffer Size**: 100 messages per topic
- **Connection Timeout**: 60 seconds
- **Graceful Shutdown Timeout**: 10 seconds
- **Ping Interval**: 54 seconds
- **Max Message Size**: 1MB

## 🚨 Error Handling

### WebSocket Errors
- `BAD_REQUEST`: Invalid message format, missing required fields
- `SLOW_CONSUMER`: Client queue overflow, connection will be closed

### REST API Errors
- `400 Bad Request`: Invalid JSON, missing required fields
- `401 Unauthorized`: Missing or invalid API key
- `409 Conflict`: Topic already exists
- `404 Not Found`: Topic not found

## 📊 Monitoring and Observability

### Health Endpoint
- System uptime
- Active topic count
- Total subscriber count
- Total message count

### Statistics Endpoint
- Detailed per-topic metrics
- Client connection counts
- Message throughput statistics
- System performance metrics

### Logging
- Connection events (connect/disconnect)
- Message publish/subscribe events
- Error conditions and backpressure events
- Graceful shutdown progress

## 🔒 Security Considerations

### Authentication
- Optional X-API-Key header authentication
- Environment variable configuration
- Flexible deployment (with or without auth)

### Network Security
- WebSocket origin checking (configurable)
- HTTP header validation
- Request size limits

### Resource Protection
- Bounded message queues prevent memory exhaustion
- Automatic slow consumer detection and disconnection
- Graceful degradation under load

## 🚀 Performance Characteristics

### Throughput
- High-throughput message delivery using Go channels
- Concurrent client handling with goroutines
- Optimized JSON marshaling/unmarshaling

### Latency
- Low-latency message delivery
- Efficient ring buffer for message replay
- Minimal overhead in message routing

### Scalability
- Vertical scaling within single process
- Memory-bounded operation
- Connection limits based on system resources

## 🛠️ Development

### Prerequisites
- Go 1.21 or later
- Docker (optional)

### Local Development
```bash
# Clone repository
git clone <repository-url>
cd plivo-pubsub

# Install dependencies
go mod download

# Run locally
go run main.go

# Run with authentication
API_KEY=test-key go run main.go
```

### Testing
```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## 📝 License

This project is licensed under the MIT License.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📞 Support

For issues and questions:
- Create an issue in the repository
- Check the logs for error details
- Verify configuration and environment variables
- Test with the health endpoint first