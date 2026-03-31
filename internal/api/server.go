package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/VOLTTRON/sim-rtu/internal/engine"
)

// Server provides a REST API for the simulation engine.
type Server struct {
	engine *engine.Engine
	srv    *http.Server
	token  string
}

// New creates a new API server. If token is non-empty, PUT/POST
// requests require a matching Bearer token in the Authorization header.
// The API_TOKEN environment variable is used as a fallback when the
// config token is empty.
func New(eng *engine.Engine, host string, port int, token string) *Server {
	if token == "" {
		token = os.Getenv("API_TOKEN")
	}
	s := &Server{engine: eng, token: token}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.srv = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           s.authMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// authMiddleware enforces bearer-token authentication on mutating
// (PUT, POST) requests when a token is configured.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && (r.Method == "PUT" || r.Method == "POST") {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+s.token {
				writeJSON(w, http.StatusUnauthorized, Response{Error: "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Start begins serving HTTP requests.
func (s *Server) Start(_ context.Context) error {
	slog.Info("API server starting", "addr", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("api server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
