package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var pool *pgxpool.Pool

// Connect establishes a pgx connection pool using the provided DSN.
// It retries up to maxRetries times before returning an error.
func Connect(ctx context.Context, dsn string) error {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 10 * time.Minute

	const maxRetries = 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		p, connErr := pgxpool.NewWithConfig(ctx, cfg)
		if connErr != nil {
			log.Warn().
				Err(connErr).
				Int("attempt", attempt).
				Msg("database connection attempt failed")
			if attempt == maxRetries {
				return fmt.Errorf("connect to database after %d attempts: %w", maxRetries, connErr)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		if pingErr := p.Ping(ctx); pingErr != nil {
			p.Close()
			log.Warn().
				Err(pingErr).
				Int("attempt", attempt).
				Msg("database ping failed")
			if attempt == maxRetries {
				return fmt.Errorf("ping database after %d attempts: %w", maxRetries, pingErr)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		pool = p
		log.Info().Msg("database connection pool established")
		return nil
	}

	return fmt.Errorf("could not connect to database after %d attempts", maxRetries)
}

// Ping verifies the database connection is alive.
func Ping(ctx context.Context) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialised")
	}
	return pool.Ping(ctx)
}

// GetPool returns the active pgxpool.Pool. Panics if Connect has not been called.
func GetPool() *pgxpool.Pool {
	if pool == nil {
		panic("database pool is not initialised; call database.Connect first")
	}
	return pool
}

// Close gracefully shuts down the connection pool.
func Close() {
	if pool != nil {
		pool.Close()
		log.Info().Msg("database connection pool closed")
	}
}
