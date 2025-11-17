package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/sawdustofmind/geniussports-event-processor/internal/consumer"
	"github.com/sawdustofmind/geniussports-event-processor/internal/log"
	"github.com/sawdustofmind/geniussports-event-processor/internal/models"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type Server struct {
	handler *consumer.Handler
}

func (s *Server) heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(HealthResponse{Status: "ok"}); err != nil {
		log.Error("Failed to write heartbeat response", zap.Error(err))
		return
	}
	log.Debug("Heartbeat received")
}

func (s *Server) processMessageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Error("Failed to close request body", zap.Error(err))
		}
	}()

	var msg models.Message
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Error("Failed to parse JSON", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.handler.ProcessMessage(r.Context(), msg); err != nil {
		log.Error("Error processing message", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to process message: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func run() int {
	port := flag.String("port", "8080", "Port to listen on")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	log.Info("Starting Consumer Service",
		zap.String("port", *port),
		zap.String("redis_addr", *redisAddr),
	)

	// Initialize handler
	handler, err := consumer.NewHandler(*redisAddr)
	if err != nil {
		log.Error("Failed to initialize handler", zap.Error(err))
		return 1
	}
	defer func() {
		if err := handler.Close(); err != nil {
			log.Error("Failed to close handler", zap.Error(err))
		}
	}()

	server := &Server{
		handler: handler,
	}

	// Setup router
	r := mux.NewRouter()
	r.HandleFunc("/heartbeat", server.heartbeatHandler).Methods("POST")
	r.HandleFunc("/process-msg", server.processMessageHandler).Methods("POST")

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:    ":" + *port,
		Handler: r,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Info("Consumer service listening", zap.String("port", *port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server error", zap.Error(err))
			errChan <- err
		}
	}()

	select {
	case <-sigChan:
		log.Info("Shutdown signal received, stopping server")
	case <-errChan:
		return 1
	}

	if err := httpServer.Close(); err != nil {
		log.Error("Error closing server", zap.Error(err))
	}
	log.Info("Consumer service stopped")
	return 0
}

func main() {
	// Initialize global logger
	if err := log.Init(true); err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	os.Exit(run())
}
