package ws

import (
	"encoding/json"
	"time"
)

// CommandType identifies the type of command or event message.
type CommandType string

const (
	// Inbound commands (control plane → agent).
	CommandSyncInventory CommandType = "sync_inventory"
	CommandStartSession  CommandType = "start_session"
	CommandStopSession   CommandType = "stop_session"
	CommandStartMBUSD    CommandType = "start_mbusd"
	CommandStopMBUSD     CommandType = "stop_mbusd"
	CommandHealthPing    CommandType = "health_ping"

	// Outbound events (agent → control plane).
	EventRegistration    CommandType = "registration"
	EventAck             CommandType = "ack"
	EventInventoryUpdate CommandType = "inventory_update"
	EventHeartbeat       CommandType = "heartbeat"
	EventSessionStarted  CommandType = "session_started"
	EventSessionStopped  CommandType = "session_stopped"
	EventMBUSDStarted    CommandType = "mbusd_started"
	EventMBUSDStopped    CommandType = "mbusd_stopped"
)

// AgentMessage is the envelope for all WebSocket messages in both directions.
type AgentMessage struct {
	ID        string          `json:"id"`
	Type      CommandType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// AgentAck is sent back to acknowledge receipt of a command.
type AgentAck struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// RegistrationMessage is the first message sent when the agent connects.
type RegistrationMessage struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	Secret   string `json:"secret"`
	Version  string `json:"version"`
}

// StartSessionPayload carries parameters for a new port-forward session.
type StartSessionPayload struct {
	SessionID    string `json:"session_id"`
	TargetPort   int    `json:"target_port"`
	Protocol     string `json:"protocol"`
	TTLSeconds   int    `json:"ttl_seconds"`
	ListenPort   int    `json:"listen_port,omitempty"`
}

// StopSessionPayload identifies a session to terminate.
type StopSessionPayload struct {
	SessionID string `json:"session_id"`
}

// StartMBUSDPayload carries parameters for spawning an mbusd bridge process.
type StartMBUSDPayload struct {
	BridgeID   string `json:"bridge_id"`
	SerialPort string `json:"serial_port"`
	BaudRate   int    `json:"baud_rate"`
	TCPPort    int    `json:"tcp_port"`
	Parity     string `json:"parity"`
}

// StopMBUSDPayload identifies an mbusd bridge to terminate.
type StopMBUSDPayload struct {
	BridgeID string `json:"bridge_id"`
}

// HeartbeatPayload is attached to outbound heartbeat events.
type HeartbeatPayload struct {
	DeviceID        string    `json:"device_id"`
	TenantID        string    `json:"tenant_id"`
	Uptime          float64   `json:"uptime_seconds"`
	ActiveSessions  int       `json:"active_sessions"`
	ActiveBridges   int       `json:"active_bridges"`
	Timestamp       time.Time `json:"timestamp"`
}
