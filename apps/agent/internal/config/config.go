package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all agent configuration loaded from environment variables.
type Config struct {
	DeviceID              string
	ControlPlaneURL       string
	AgentSecret           string
	TenantID              string
	InventoryScanInterval time.Duration
	HeartbeatInterval     time.Duration
	LogLevel              string
	MaxConcurrentSessions int
}

// Load reads configuration from an optional .env file and then from environment
// variables. Returns a validation error if any required field is missing.
func Load(envFile string) (*Config, error) {
	// Best-effort load of .env file — ignore error if it does not exist.
	if envFile != "" {
		_ = godotenv.Load(envFile)
	} else {
		_ = godotenv.Load()
	}

	cfg := &Config{}

	var errs []string

	cfg.DeviceID = os.Getenv("DEVICE_ID")
	if cfg.DeviceID == "" {
		errs = append(errs, "DEVICE_ID is required")
	}

	cfg.ControlPlaneURL = os.Getenv("CONTROL_PLANE_URL")
	if cfg.ControlPlaneURL == "" {
		cfg.ControlPlaneURL = "wss://api.nucleus.example.com/ws/agent"
	}

	cfg.AgentSecret = os.Getenv("AGENT_SECRET")
	if cfg.AgentSecret == "" {
		errs = append(errs, "AGENT_SECRET is required")
	}

	cfg.TenantID = os.Getenv("TENANT_ID")
	if cfg.TenantID == "" {
		errs = append(errs, "TENANT_ID is required")
	}

	cfg.InventoryScanInterval = parseDuration("INVENTORY_SCAN_INTERVAL", 60*time.Second)
	cfg.HeartbeatInterval = parseDuration("HEARTBEAT_INTERVAL", 30*time.Second)

	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	cfg.MaxConcurrentSessions = parseInt("MAX_CONCURRENT_SESSIONS", 5)

	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed: %w", errors.New(joinStrings(errs, "; ")))
	}

	return cfg, nil
}

func parseDuration(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return d
}

func parseInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return n
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
