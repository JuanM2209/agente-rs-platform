package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

const configFileName = "config.json"

// Config holds persisted authentication configuration.
type Config struct {
	Token  string `json:"token"`
	APIURL string `json:"api_url"`
}

// configDir returns the path to ~/.nucleus/.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".nucleus"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// SaveToken persists the JWT token and API URL to ~/.nucleus/config.json.
// The config directory is created if it does not exist.
func SaveToken(token, apiURL string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %q: %w", dir, err)
	}

	cfg := Config{
		Token:  token,
		APIURL: apiURL,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config to %q: %w", path, err)
	}

	log.Debug().Str("path", path).Msg("token saved")
	return nil
}

// LoadToken reads the token and API URL from ~/.nucleus/config.json.
// Returns an error if the file does not exist or cannot be parsed.
func LoadToken() (token, apiURL string, err error) {
	path, err := configPath()
	if err != nil {
		return "", "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("not logged in: run 'nucleus-helper login' first")
		}
		return "", "", fmt.Errorf("reading config from %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Token == "" {
		return "", "", fmt.Errorf("no token found in config: run 'nucleus-helper login' first")
	}
	if cfg.APIURL == "" {
		return "", "", fmt.Errorf("no api_url found in config: run 'nucleus-helper login' first")
	}

	return cfg.Token, cfg.APIURL, nil
}

// ValidateToken checks whether the stored JWT token is present and not yet expired.
// It does NOT verify the signature — that is the server's responsibility.
func ValidateToken() bool {
	token, _, err := LoadToken()
	if err != nil {
		log.Debug().Err(err).Msg("token load failed during validation")
		return false
	}

	// Parse without verifying the signature so we can inspect the claims.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		log.Debug().Err(err).Msg("token parse failed during validation")
		return false
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		// No expiry claim — treat as valid.
		return true
	}

	return exp.Time.After(time.Now())
}
