package bacnet

import (
	"context"
	"log/slog"

	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/points"
)

// Server is a stub BACnet/IP server.
// A full implementation using a Go BACnet library will replace this.
type Server struct {
	config  config.BACnetConfig
	devices map[int]*points.PointStore
}

// New creates a new BACnet server stub.
func New(cfg config.BACnetConfig) *Server {
	return &Server{
		config:  cfg,
		devices: make(map[int]*points.PointStore),
	}
}

// Start is a stub that logs a message and returns nil.
func (s *Server) Start(_ context.Context, devices map[int]*points.PointStore) error {
	s.devices = devices
	slog.Warn("BACnet server is a stub — not serving BACnet/IP traffic",
		"interface", s.config.Interface,
		"port", s.config.Port,
		"device_count", len(devices),
	)
	return nil
}

// Stop is a stub that returns nil.
func (s *Server) Stop() error {
	return nil
}
