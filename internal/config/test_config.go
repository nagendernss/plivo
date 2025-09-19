package config

// NewTestConfig creates a test configuration with default values
func NewTestConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           "8080",
			ReadTimeout:    10 * 1000000000, // 10 seconds in nanoseconds
			WriteTimeout:   10 * 1000000000, // 10 seconds in nanoseconds
			IdleTimeout:    60 * 1000000000, // 60 seconds in nanoseconds
			ShutdownTimeout: 10 * 1000000000, // 10 seconds in nanoseconds
		},
		PubSub: PubSubConfig{
			MaxQueueSize:     100,
			RingBufferSize:   100,
			PingInterval:     54 * 1000000000, // 54 seconds in nanoseconds
			PongWait:         60 * 1000000000, // 60 seconds in nanoseconds
			WriteWait:        10 * 1000000000, // 10 seconds in nanoseconds
			MaxMessageSize:   1024 * 1024,     // 1MB
			EnableCompression: false,
		},
		Security: SecurityConfig{
			APIKey:          "",
			EnableCORS:      false,
			AllowedOrigins:  "*",
			RateLimitPerMin: 1000,
			RateLimitBurst:  100,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// NewTestConfigWithAPIKey creates a test configuration with an API key
func NewTestConfigWithAPIKey(apiKey string) *Config {
	cfg := NewTestConfig()
	cfg.Security.APIKey = apiKey
	return cfg
}
