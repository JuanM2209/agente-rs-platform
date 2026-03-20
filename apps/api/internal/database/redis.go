package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var redisClient *redis.Client

// ConnectRedis creates and validates a Redis client using the provided URL.
func ConnectRedis(ctx context.Context, redisURL string) error {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("parse redis URL: %w", err)
	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.PoolSize = 10

	client := redis.NewClient(opts)

	const maxRetries = 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if pingErr := client.Ping(ctx).Err(); pingErr != nil {
			log.Warn().
				Err(pingErr).
				Int("attempt", attempt).
				Msg("redis ping failed")
			if attempt == maxRetries {
				_ = client.Close()
				return fmt.Errorf("ping redis after %d attempts: %w", maxRetries, pingErr)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		break
	}

	redisClient = client
	log.Info().Msg("redis connection established")
	return nil
}

// PingRedis verifies the Redis connection is alive.
func PingRedis(ctx context.Context) error {
	if redisClient == nil {
		return fmt.Errorf("redis client is not initialised")
	}
	return redisClient.Ping(ctx).Err()
}

// GetRedis returns the active Redis client. Panics if ConnectRedis has not been called.
func GetRedis() *redis.Client {
	if redisClient == nil {
		panic("redis client is not initialised; call database.ConnectRedis first")
	}
	return redisClient
}

// CloseRedis shuts down the Redis client.
func CloseRedis() {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("error closing redis client")
		} else {
			log.Info().Msg("redis connection closed")
		}
	}
}
