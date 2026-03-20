package ws

import (
	"encoding/json"
	"time"
)

// CommandType identifies the operation being sent to or received from an agent.
type CommandType string

const (
	CmdSyncInventory       CommandType = "sync_inventory"
	CmdStartSession        CommandType = "start_session"
	CmdStopSession         CommandType = "stop_session"
	CmdStartMBUSD          CommandType = "start_mbusd"
	CmdStopMBUSD           CommandType = "stop_mbusd"
	CmdRefreshCapabilities CommandType = "refresh_capabilities"
	CmdHealthPing          CommandType = "health_ping"

	EventRegistration    CommandType = "registration"
	EventAck             CommandType = "ack"
	EventInventoryUpdate CommandType = "inventory_update"
	EventHeartbeat       CommandType = "heartbeat"
	EventSessionStarted  CommandType = "session_started"
	EventSessionStopped  CommandType = "session_stopped"
	EventMBUSDStarted    CommandType = "mbusd_started"
	EventMBUSDStopped    CommandType = "mbusd_stopped"
)

// AgentMessage is the envelope for commands sent from the API to an agent.
type AgentMessage struct {
	ID        string          `json:"id"`
	Type      CommandType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// AgentAck is the acknowledgement sent back by the agent after processing a command.
type AgentAck struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type RegistrationMessage struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	Secret   string `json:"secret"`
	Version  string `json:"version"`
}

// EndpointInfo describes a single service endpoint discovered on the device.
type EndpointInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`
}

// InventoryPayload is the data sent by the agent after a sync_inventory command.
type InventoryPayload struct {
	DeviceID  string         `json:"device_id"`
	Endpoints []EndpointInfo `json:"endpoints"`
	Timestamp time.Time      `json:"timestamp"`
}

// StartSessionPayload carries the parameters needed to open a remote session.
type StartSessionPayload struct {
	SessionID string `json:"session_id"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	TTL       int    `json:"ttl_seconds"`
}

// StopSessionPayload identifies the session to be terminated.
type StopSessionPayload struct {
	SessionID string `json:"session_id"`
}

// StartMBUSDPayload carries the configuration required to start a MBUSD bridge.
type StartMBUSDPayload struct {
	BridgeID   string `json:"bridge_id"`
	SerialPort string `json:"serial_port"`
	BaudRate   int    `json:"baud_rate"`
	Parity     string `json:"parity"`
	StopBits   int    `json:"stop_bits"`
	DataBits   int    `json:"data_bits"`
	TCPPort    int    `json:"tcp_port"`
}

// StopMBUSDPayload identifies the bridge to stop.
type StopMBUSDPayload struct {
	BridgeID string `json:"bridge_id"`
}

type HeartbeatPayload struct {
	DeviceID       string    `json:"device_id"`
	TenantID       string    `json:"tenant_id"`
	Uptime         float64   `json:"uptime_seconds"`
	ActiveSessions int       `json:"active_sessions"`
	ActiveBridges  int       `json:"active_bridges"`
	Timestamp      time.Time `json:"timestamp"`
}

// AgentStatusMessage is a generic status update pushed by the agent.
type AgentStatusMessage struct {
	DeviceID  string    `json:"device_id"`
	Event     string    `json:"event"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
