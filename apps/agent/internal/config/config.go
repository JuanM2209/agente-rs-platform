package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all agent configuration loaded from environment variables.
type Config struct {
	DeviceID              string
	DeviceIDSource        string
	DeviceIDFile          string
	ControlPlaneURL       string
	LocalIPOverride       string
	PreferredLANInterface string
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

	cfg.DeviceIDFile = os.Getenv("NUCLEUS_SERIAL_NUMBER_FILE")
	if cfg.DeviceIDFile == "" {
		cfg.DeviceIDFile = "/data/nucleus/factory/nucleus_serial_number"
	}

	deviceID, source, err := resolveDeviceID(os.Getenv("DEVICE_ID"), cfg.DeviceIDFile)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		cfg.DeviceID = deviceID
		cfg.DeviceIDSource = source
	}

	cfg.ControlPlaneURL = os.Getenv("CONTROL_PLANE_URL")
	if cfg.ControlPlaneURL == "" {
		cfg.ControlPlaneURL = "wss://api.nucleus.example.com/ws/agent"
	}
	cfg.LocalIPOverride = strings.TrimSpace(os.Getenv("LOCAL_IP_OVERRIDE"))
	cfg.PreferredLANInterface = strings.TrimSpace(os.Getenv("PREFERRED_LAN_INTERFACE"))

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

func resolveDeviceID(envValue, filePath string) (string, string, error) {
	deviceID := strings.TrimSpace(envValue)
	if deviceID != "" {
		return deviceID, "env", nil
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("DEVICE_ID is required or %s must exist", filePath)
		}
		return "", "", fmt.Errorf("reading device ID from %s failed: %w", filePath, err)
	}

	deviceID = strings.TrimSpace(string(raw))
	if deviceID == "" {
		return "", "", fmt.Errorf("device ID file %s is empty", filePath)
	}

	return deviceID, "file", nil
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
