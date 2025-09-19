package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"plivo/internal/handlers"
	"plivo/internal/pubsub"
	"syscall"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize the hub
	hub := pubsub.NewHub()
	go hub.Run()

	// Initialize handlers
	wsHandler := handlers.NewWebSocketHandler(hub)
	restHandler := handlers.NewRESTHandler(hub)

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

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Println("Server starting on :8080")
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
	if err := server.Shutdown(nil); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server shutdown complete")
}
