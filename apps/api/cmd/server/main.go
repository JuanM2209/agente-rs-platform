package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nucleus-portal/api/internal/config"
	"github.com/nucleus-portal/api/internal/database"
	"github.com/nucleus-portal/api/internal/ws"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load .env file if present (ignored in production when vars are injected).
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warn().Err(err).Msg("could not load .env file")
	}

	// ── Logging ───────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENVIRONMENT") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	// ── Configuration ─────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	if cfg.Environment == "production" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().
		Str("environment", cfg.Environment).
		Int("port", cfg.ServerPort).
		Msg("starting Nucleus API server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Database ──────────────────────────────────────────────────────────────
	if err := database.Connect(ctx, cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer database.Close()

	// ── Redis ─────────────────────────────────────────────────────────────────
	if err := database.ConnectRedis(ctx, cfg.RedisURL); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer database.CloseRedis()

	// ── WebSocket hub ─────────────────────────────────────────────────────────
	hub := ws.NewAgentHub(cfg.AgentWSSecret)

	// ── Router ────────────────────────────────────────────────────────────────
	router := buildRouter(cfg, hub)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Run server in background goroutine.
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("server error")
	}

	log.Info().Msg("shutting down gracefully (30s timeout)...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed; forcing close")
		_ = srv.Close()
	} else {
		log.Info().Msg("server shutdown complete")
	}
}
