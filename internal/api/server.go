package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/VOLTTRON/sim-rtu/internal/engine"
)

// Server provides a REST API for the simulation engine.
type Server struct {
	engine *engine.Engine
	srv    *http.Server
}

// New creates a new API server.
func New(eng *engine.Engine, host string, port int) *Server {
	s := &Server{engine: eng}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: mux,
	}

	return s
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
