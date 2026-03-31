package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VOLTTRON/sim-rtu/internal/api"
	"github.com/VOLTTRON/sim-rtu/internal/bacnet"
	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/engine"
	"github.com/VOLTTRON/sim-rtu/internal/points"
)

func main() {
	configPath := flag.String("config", "configs/default.yml", "path to config file")
	logLevel := flag.String("log-level", "INFO", "log level (DEBUG, INFO, WARN, ERROR)")
	flag.Parse()

	// Setup slog
	var level slog.Level
	switch *logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "path", *configPath, "devices", len(cfg.Devices))

	// Create engine
	eng, err := engine.New(cfg)
	if err != nil {
		slog.Error("failed to create engine", "error", err)
		os.Exit(1)
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start engine
	go func() {
		if err := eng.Start(ctx); err != nil && err != context.Canceled {
			slog.Error("engine error", "error", err)
		}
	}()

	// Start API server
	if cfg.API.Enabled {
		apiSrv := api.New(eng, cfg.API.Host, cfg.API.Port)
		go func() {
			if err := apiSrv.Start(ctx); err != nil {
				slog.Error("API server error", "error", err)
			}
		}()
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := apiSrv.Stop(shutdownCtx); err != nil {
				slog.Error("API server shutdown error", "error", err)
			}
		}()
	}

	// Start BACnet server
	if cfg.BACnet.Enabled {
		bacnetSrv := bacnet.New(cfg.BACnet)
		stores := make(map[int]*points.PointStore)
		for id, dev := range eng.Devices() {
			stores[id] = dev.Store
		}
		if err := bacnetSrv.Start(ctx, stores); err != nil {
			slog.Error("BACnet server error", "error", err)
		}
		defer func() {
			if err := bacnetSrv.Stop(); err != nil {
				slog.Error("BACnet server shutdown error", "error", err)
			}
		}()
	}

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	slog.Info("received signal, shutting down", "signal", sig)

	cancel()
	eng.Stop()
	slog.Info("shutdown complete")
}
