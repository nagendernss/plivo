package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"plivo/docs"
	"plivo/internal/config"
	"plivo/internal/handlers"
	"plivo/internal/pubsub"
	"syscall"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Plivo Pub/Sub System API
// @version 1.0
// @description A production-ready in-memory Pub/Sub system with WebSocket and REST API support
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
	// Load configuration from command-line flags and environment variables
	cfg := config.LoadConfig()

	log.Printf("Starting Plivo Pub/Sub System with configuration:")
	log.Printf("  Server Port: %s", cfg.Server.Port)
	log.Printf("  Max Queue Size: %d", cfg.PubSub.MaxQueueSize)
	log.Printf("  Ring Buffer Size: %d", cfg.PubSub.RingBufferSize)
	log.Printf("  API Key Required: %t", cfg.Security.APIKey != "")
	log.Printf("  CORS Enabled: %t", cfg.Security.EnableCORS)
	log.Printf("  Log Level: %s", cfg.Logging.Level)

	// Initialize the hub
	hub := pubsub.NewHub()
	go hub.Run()

	// Initialize handlers with configuration
	wsHandler := handlers.NewWebSocketHandler(hub, cfg)
	restHandler := handlers.NewRESTHandler(hub, cfg)

	// Setup routes
	r := mux.NewRouter()

	// WebSocket endpoint
	r.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// REST API endpoints
	r.HandleFunc("/topics", restHandler.CreateTopic).Methods("POST")
	r.HandleFunc("/topics", restHandler.ListTopics).Methods("GET")
	r.HandleFunc("/topics/{topic}", restHandler.DeleteTopic).Methods("DELETE")
	r.HandleFunc("/health", restHandler.Health).Methods("GET")
	r.HandleFunc("/stats", restHandler.Stats).Methods("GET")

	// Swagger documentation
	r.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(docs.SwaggerInfo.ReadDoc()))
	}).Methods("GET")
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // The url pointing to API definition
	))

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on :%s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, starting graceful shutdown...")

	// Shutdown hub first
	hub.Shutdown()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server shutdown complete")
}
