package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"receipt-bank/internal/handlers"
)

// Server represents the HTTP server
type Server struct {
	router  *mux.Router
	handler *handlers.Handler
	verbose bool
}

// NewServer creates a new HTTP server
func NewServer(handler *handlers.Handler, verbose bool) *Server {
	server := &Server{
		router:  mux.NewRouter(),
		handler: handler,
		verbose: verbose,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("/submit", s.handler.SubmitHandler).Methods("POST")
	s.router.HandleFunc("/collect/{ephemeral_key}", s.handler.CollectHandler).Methods("GET")
	s.router.HandleFunc("/health", s.handler.HealthHandler).Methods("GET")

	// Add logging middleware
	s.router.Use(s.loggingMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)

	if s.verbose {
		log.Printf("[SERVER] Starting Receipt Bank server on port %d", port)
		log.Printf("[SERVER] Available endpoints:")
		log.Printf("[SERVER]   POST /submit")
		log.Printf("[SERVER]   GET  /collect/{ephemeral_key}")
		log.Printf("[SERVER]   GET  /health")
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.verbose {
			start := time.Now()

			// Call next handler
			next.ServeHTTP(w, r)

			// Log request
			log.Printf("[HTTP] %s %s - %v", r.Method, r.URL.Path, time.Since(start))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
