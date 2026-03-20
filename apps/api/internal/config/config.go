package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	ServerPort             int
	DatabaseURL            string
	RedisURL               string
	JWTSecret              string
	JWTExpiryHours         int
	Environment            string
	CloudflareTunnelToken  string
	AgentWSSecret          string
}

// Load reads configuration from environment variables and returns a populated Config.
// Returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.ServerPort = getEnvInt("SERVER_PORT", 8080)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	cfg.DatabaseURL = dbURL

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	cfg.RedisURL = redisURL

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWTSecret = jwtSecret

	agentSecret := os.Getenv("AGENT_WS_SECRET")
	if agentSecret == "" {
		return nil, fmt.Errorf("AGENT_WS_SECRET is required")
	}
	cfg.AgentWSSecret = agentSecret

	cfg.JWTExpiryHours = getEnvInt("JWT_EXPIRY_HOURS", 24)

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	cfg.Environment = env

	// Optional
	cfg.CloudflareTunnelToken = os.Getenv("CLOUDFLARE_TUNNEL_TOKEN")

	return cfg, nil
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}
