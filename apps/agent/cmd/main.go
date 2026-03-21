package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nucleus-portal/agent/internal/agent"
	"github.com/nucleus-portal/agent/internal/config"
	"github.com/nucleus-portal/agent/internal/inventory"
	"github.com/nucleus-portal/agent/internal/ws"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// -------------------------------------------------------------------------
	// Configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// -------------------------------------------------------------------------
	// Logging
	// -------------------------------------------------------------------------
	logger := buildLogger(cfg.LogLevel)
	logger.Info().
		Str("device_id", cfg.DeviceID).
		Str("device_id_source", cfg.DeviceIDSource).
		Str("tenant_id", cfg.TenantID).
		Str("control_plane_url", cfg.ControlPlaneURL).
		Msg("nucleus agent starting")

	// -------------------------------------------------------------------------
	// WebSocket hub
	// -------------------------------------------------------------------------
	header := http.Header{}
	header.Set("X-Device-ID", cfg.DeviceID)
	header.Set("X-Tenant-ID", cfg.TenantID)
	header.Set("X-Agent-Secret", cfg.AgentSecret)
	header.Set("Authorization", "Bearer "+cfg.AgentSecret)

	hub := ws.NewHub(cfg.ControlPlaneURL, header, logger.With().Str("component", "hub").Logger())

	// -------------------------------------------------------------------------
	// Inventory scanner
	// -------------------------------------------------------------------------
	scanner := inventory.NewScanner(
		cfg.InventoryScanInterval,
		cfg.ControlPlaneURL,
		cfg.LocalIPOverride,
		cfg.PreferredLANInterface,
		logger.With().Str("component", "inventory").Logger(),
	)

	// -------------------------------------------------------------------------
	// Agent
	// -------------------------------------------------------------------------
	a := agent.New(
		cfg,
		hub,
		scanner,
		logger.With().Str("component", "agent").Logger(),
	)

	// -------------------------------------------------------------------------
	// Context + signal handling
	// -------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// -------------------------------------------------------------------------
	// Start background services
	// -------------------------------------------------------------------------
	// Inventory scanner — runs in its own goroutine.
	go scanner.Start()

	// Connect with retry before entering the main loop.
	logger.Info().Msg("connecting to control plane")
	if err := connectWithRetry(ctx, a, logger); err != nil {
		return fmt.Errorf("initial connect failed: %w", err)
	}

	// Agent main loop.
	agentDone := make(chan struct{})
	go func() {
		defer close(agentDone)
		a.Run(ctx)
	}()

	// -------------------------------------------------------------------------
	// Wait for shutdown signal
	// -------------------------------------------------------------------------
	select {
	case sig := <-sigCh:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case <-agentDone:
		logger.Info().Msg("agent exited")
	}

	// Initiate graceful shutdown.
	logger.Info().Msg("shutting down")
	cancel()
	scanner.Stop()
	a.Stop()

	// Give agent a moment to flush in-flight messages.
	shutdownTimeout := 5 * time.Second
	select {
	case <-agentDone:
		logger.Info().Msg("agent stopped cleanly")
	case <-time.After(shutdownTimeout):
		logger.Warn().Dur("timeout", shutdownTimeout).Msg("agent did not stop within timeout")
	}

	return nil
}

// connectWithRetry attempts to connect the agent to the control plane,
// retrying with exponential backoff. The Hub itself handles the backoff;
// this wrapper just surfaces a context-cancellation error.
func connectWithRetry(ctx context.Context, a *agent.Agent, logger zerolog.Logger) error {
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- a.Connect()
	}()

	select {
	case err := <-resultCh:
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		logger.Info().Msg("connected to control plane")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before connection established")
	}
}

// buildLogger creates a zerolog logger at the given level.
func buildLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	// Pretty console output during development; JSON in production.
	if level == "debug" {
		return log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(lvl)
	}

	return zerolog.New(os.Stderr).With().Timestamp().Logger().Level(lvl)
}
