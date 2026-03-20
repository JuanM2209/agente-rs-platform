package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type apiEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

type SessionTelemetry struct {
	ConnectionStatus string     `json:"connection_status"`
	LatencyMS        *int       `json:"latency_ms,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
	ProbeSource      string     `json:"probe_source,omitempty"`
}

type SessionDevice struct {
	DeviceID    string `json:"device_id"`
	DisplayName string `json:"display_name"`
}

type SessionEndpoint struct {
	Label string `json:"label"`
	Port  int    `json:"port"`
}

// Session represents an active Nucleus session returned by the API.
type Session struct {
	ID         string            `json:"id"`
	DeviceID   string            `json:"device_id"`
	DeviceName string            `json:"device_name"`
	RemoteHost string            `json:"remote_host"`
	RemotePort int               `json:"remote_port"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Status     string            `json:"status"`
	Telemetry  *SessionTelemetry `json:"telemetry,omitempty"`
	Device     *SessionDevice    `json:"device,omitempty"`
	Endpoint   *SessionEndpoint  `json:"endpoint,omitempty"`
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
// the local config. Call LoadToken() before this if you need to obtain them.
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

// get executes an authenticated GET request and decodes the standard API
// envelope payload into dest.
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
		return fmt.Errorf("unauthorized - run 'nucleus-helper login' to refresh your token")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %q returned HTTP %d", url, resp.StatusCode)
	}

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decoding response from %q: %w", url, err)
	}
	if !envelope.Success {
		if envelope.Error == "" {
			envelope.Error = "request failed"
		}
		return fmt.Errorf("%s", envelope.Error)
	}
	if dest == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return fmt.Errorf("decoding response payload from %q: %w", url, err)
	}
	return nil
}

// post executes an authenticated POST request and decodes the standard API
// envelope payload into dest when provided.
func (c *APIClient) post(path string, payload interface{}, dest interface{}) error {
	url := c.baseURL + path
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding POST payload for %q: %w", url, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building POST request for %q: %w", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized - run 'nucleus-helper login' to refresh your token")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %q returned HTTP %d", url, resp.StatusCode)
	}

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decoding response from %q: %w", url, err)
	}
	if !envelope.Success {
		if envelope.Error == "" {
			envelope.Error = "request failed"
		}
		return fmt.Errorf("%s", envelope.Error)
	}
	if dest == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return fmt.Errorf("decoding response payload from %q: %w", url, err)
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
		return fmt.Errorf("unauthorized - run 'nucleus-helper login' to refresh your token")
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

	for i := range sessions {
		if sessions[i].DeviceName == "" && sessions[i].Device != nil {
			if sessions[i].Device.DisplayName != "" {
				sessions[i].DeviceName = sessions[i].Device.DisplayName
			} else {
				sessions[i].DeviceName = sessions[i].Device.DeviceID
			}
		}
		if sessions[i].RemotePort == 0 && sessions[i].Endpoint != nil {
			sessions[i].RemotePort = sessions[i].Endpoint.Port
		}
	}

	return sessions, nil
}

// UpdateSessionTelemetry pushes helper-side probe telemetry back to the API.
func (c *APIClient) UpdateSessionTelemetry(id string, telemetry SessionTelemetry) error {
	if id == "" {
		return fmt.Errorf("session ID must not be empty")
	}
	if err := c.post("/api/v1/sessions/"+id+"/telemetry", telemetry, nil); err != nil {
		return fmt.Errorf("updating telemetry for session %q: %w", id, err)
	}
	return nil
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
