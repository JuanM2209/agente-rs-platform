package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Session represents an active Nucleus session returned by the API.
type Session struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
	RemoteHost string    `json:"remote_host"`
	RemotePort int       `json:"remote_port"`
	ExpiresAt  time.Time `json:"expires_at"`
	Status     string    `json:"status"`
}

// Device represents a Nucleus-managed device.
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Online   bool   `json:"online"`
}

// APIClient is an authenticated HTTP client for the Nucleus API.
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewAPIClient constructs an APIClient using the token and API URL stored in
// the local config.  Call LoadToken() before this if you need to obtain them.
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// NewAPIClientFromConfig is a convenience constructor that loads credentials
// from the persisted config file.
func NewAPIClientFromConfig() (*APIClient, error) {
	token, apiURL, err := LoadToken()
	if err != nil {
		return nil, err
	}
	return NewAPIClient(apiURL, token), nil
}

// get executes an authenticated GET request and decodes the JSON response body
// into dest.
func (c *APIClient) get(path string, dest interface{}) error {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building GET request for %q: %w", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — run 'nucleus-helper login' to refresh your token")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %q returned HTTP %d", url, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response from %q: %w", url, err)
	}
	return nil
}

// delete executes an authenticated DELETE request.
func (c *APIClient) delete(path string) error {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("building DELETE request for %q: %w", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — run 'nucleus-helper login' to refresh your token")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("DELETE %q returned HTTP %d", url, resp.StatusCode)
	}

	return nil
}

// GetActiveSessions returns all active sessions for the authenticated user.
func (c *APIClient) GetActiveSessions() ([]Session, error) {
	var sessions []Session
	if err := c.get("/api/v1/me/active-sessions", &sessions); err != nil {
		return nil, fmt.Errorf("fetching active sessions: %w", err)
	}
	return sessions, nil
}

// StopSession terminates the session with the given ID on the server.
func (c *APIClient) StopSession(id string) error {
	if id == "" {
		return fmt.Errorf("session ID must not be empty")
	}
	if err := c.delete("/api/v1/sessions/" + id); err != nil {
		return fmt.Errorf("stopping session %q: %w", id, err)
	}
	return nil
}

// GetDevice fetches metadata for a single device by ID.
func (c *APIClient) GetDevice(id string) (*Device, error) {
	if id == "" {
		return nil, fmt.Errorf("device ID must not be empty")
	}
	var device Device
	if err := c.get("/api/v1/devices/"+id, &device); err != nil {
		return nil, fmt.Errorf("fetching device %q: %w", id, err)
	}
	return &device, nil
}
