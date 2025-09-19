package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`

	// Pub/Sub configuration
	PubSub PubSubConfig `json:"pubsub"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string        `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// PubSubConfig holds pub/sub system configuration
type PubSubConfig struct {
	MaxQueueSize      int           `json:"max_queue_size"`
	RingBufferSize    int           `json:"ring_buffer_size"`
	PingInterval      time.Duration `json:"ping_interval"`
	PongWait          time.Duration `json:"pong_wait"`
	WriteWait         time.Duration `json:"write_wait"`
	MaxMessageSize    int64         `json:"max_message_size"`
	EnableCompression bool          `json:"enable_compression"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	APIKey          string `json:"api_key"`
	EnableCORS      bool   `json:"enable_cors"`
	AllowedOrigins  string `json:"allowed_origins"`
	RateLimitPerMin int    `json:"rate_limit_per_min"`
	RateLimitBurst  int    `json:"rate_limit_burst"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// LoadConfig loads configuration from command-line flags and environment variables
func LoadConfig() *Config {
	// Define command-line flags
	var (
		port            = flag.String("port", getEnv("PORT", "8080"), "Server port")
		readTimeout     = flag.Duration("read-timeout", getDurationEnv("READ_TIMEOUT", 10*time.Second), "HTTP read timeout")
		writeTimeout    = flag.Duration("write-timeout", getDurationEnv("WRITE_TIMEOUT", 10*time.Second), "HTTP write timeout")
		idleTimeout     = flag.Duration("idle-timeout", getDurationEnv("IDLE_TIMEOUT", 60*time.Second), "HTTP idle timeout")
		shutdownTimeout = flag.Duration("shutdown-timeout", getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second), "Graceful shutdown timeout")

		maxQueueSize      = flag.Int("max-queue-size", getIntEnv("MAX_QUEUE_SIZE", 100), "Maximum messages per client queue")
		ringBufferSize    = flag.Int("ring-buffer-size", getIntEnv("RING_BUFFER_SIZE", 100), "Ring buffer size for message replay")
		pingInterval      = flag.Duration("ping-interval", getDurationEnv("PING_INTERVAL", 54*time.Second), "WebSocket ping interval")
		pongWait          = flag.Duration("pong-wait", getDurationEnv("PONG_WAIT", 60*time.Second), "WebSocket pong wait timeout")
		writeWait         = flag.Duration("write-wait", getDurationEnv("WRITE_WAIT", 10*time.Second), "WebSocket write wait timeout")
		maxMessageSize    = flag.Int64("max-message-size", getInt64Env("MAX_MESSAGE_SIZE", 1024*1024), "Maximum message size in bytes")
		enableCompression = flag.Bool("enable-compression", getBoolEnv("ENABLE_COMPRESSION", false), "Enable WebSocket compression")

		apiKey          = flag.String("api-key", getEnv("API_KEY", ""), "API key for authentication")
		enableCORS      = flag.Bool("enable-cors", getBoolEnv("ENABLE_CORS", false), "Enable CORS support")
		allowedOrigins  = flag.String("allowed-origins", getEnv("ALLOWED_ORIGINS", "*"), "Comma-separated list of allowed origins")
		rateLimitPerMin = flag.Int("rate-limit-per-min", getIntEnv("RATE_LIMIT_PER_MIN", 1000), "Rate limit per minute")
		rateLimitBurst  = flag.Int("rate-limit-burst", getIntEnv("RATE_LIMIT_BURST", 100), "Rate limit burst size")

		logLevel  = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
		logFormat = flag.String("log-format", getEnv("LOG_FORMAT", "text"), "Log format (text, json)")

		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)

	// Parse command-line flags
	flag.Parse()

	// Handle special flags
	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Create configuration from flags
	return &Config{
		Server: ServerConfig{
			Port:            *port,
			ReadTimeout:     *readTimeout,
			WriteTimeout:    *writeTimeout,
			IdleTimeout:     *idleTimeout,
			ShutdownTimeout: *shutdownTimeout,
		},
		PubSub: PubSubConfig{
			MaxQueueSize:      *maxQueueSize,
			RingBufferSize:    *ringBufferSize,
			PingInterval:      *pingInterval,
			PongWait:          *pongWait,
			WriteWait:         *writeWait,
			MaxMessageSize:    *maxMessageSize,
			EnableCompression: *enableCompression,
		},
		Security: SecurityConfig{
			APIKey:          *apiKey,
			EnableCORS:      *enableCORS,
			AllowedOrigins:  *allowedOrigins,
			RateLimitPerMin: *rateLimitPerMin,
			RateLimitBurst:  *rateLimitBurst,
		},
		Logging: LoggingConfig{
			Level:  *logLevel,
			Format: *logFormat,
		},
	}
}

// printVersion prints version information
func printVersion() {
	println("Plivo Pub/Sub System v1.0.0")
	println("A production-ready in-memory Pub/Sub system with WebSocket and REST API support")
}

// printHelp prints help information
func printHelp() {
	println("Plivo Pub/Sub System")
	println("A production-ready in-memory Pub/Sub system with WebSocket and REST API support")
	println("")
	println("Usage:")
	println("  plivo [flags]")
	println("")
	println("Server Configuration:")
	println("  -port string")
	println("        Server port (default \"8080\")")
	println("  -read-timeout duration")
	println("        HTTP read timeout (default \"10s\")")
	println("  -write-timeout duration")
	println("        HTTP write timeout (default \"10s\")")
	println("  -idle-timeout duration")
	println("        HTTP idle timeout (default \"60s\")")
	println("  -shutdown-timeout duration")
	println("        Graceful shutdown timeout (default \"10s\")")
	println("")
	println("Pub/Sub Configuration:")
	println("  -max-queue-size int")
	println("        Maximum messages per client queue (default 100)")
	println("  -ring-buffer-size int")
	println("        Ring buffer size for message replay (default 100)")
	println("  -ping-interval duration")
	println("        WebSocket ping interval (default \"54s\")")
	println("  -pong-wait duration")
	println("        WebSocket pong wait timeout (default \"60s\")")
	println("  -write-wait duration")
	println("        WebSocket write wait timeout (default \"10s\")")
	println("  -max-message-size int")
	println("        Maximum message size in bytes (default 1048576)")
	println("  -enable-compression")
	println("        Enable WebSocket compression (default false)")
	println("")
	println("Security Configuration:")
	println("  -api-key string")
	println("        API key for authentication (default \"\")")
	println("  -enable-cors")
	println("        Enable CORS support (default false)")
	println("  -allowed-origins string")
	println("        Comma-separated list of allowed origins (default \"*\")")
	println("  -rate-limit-per-min int")
	println("        Rate limit per minute (default 1000)")
	println("  -rate-limit-burst int")
	println("        Rate limit burst size (default 100)")
	println("")
	println("Logging Configuration:")
	println("  -log-level string")
	println("        Log level (debug, info, warn, error) (default \"info\")")
	println("  -log-format string")
	println("        Log format (text, json) (default \"text\")")
	println("")
	println("Other:")
	println("  -help")
	println("        Show help information")
	println("  -version")
	println("        Show version information")
	println("")
	println("Environment Variables:")
	println("  All flags can also be set via environment variables with the same names in uppercase.")
	println("  For example: PORT=8080, API_KEY=secret, LOG_LEVEL=debug")
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
