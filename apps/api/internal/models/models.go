package models

import (
	"encoding/json"
	"time"
)

// DeviceStatus represents the online state of a device.
type DeviceStatus string

const (
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
	DeviceStatusUnknown DeviceStatus = "unknown"
)

// EndpointType classifies what kind of access endpoint is exposed.
type EndpointType string

const (
	EndpointTypeWeb     EndpointType = "WEB"
	EndpointTypeProgram EndpointType = "PROGRAM"
	EndpointTypeBridge  EndpointType = "BRIDGE"
)

// SessionStatus reflects the lifecycle state of a remote session.
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusExpired SessionStatus = "expired"
	SessionStatusStopped SessionStatus = "stopped"
)

// DeliveryMode describes how the session is consumed by the client.
type DeliveryMode string

const (
	DeliveryModeWeb    DeliveryMode = "web"
	DeliveryModeExport DeliveryMode = "export"
)

// Tenant represents an organisation using the portal.
type Tenant struct {
	ID        string    `json:"id"         db:"id"`
	Name      string    `json:"name"       db:"name"`
	Slug      string    `json:"slug"       db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Site is a physical or logical location belonging to a tenant.
type Site struct {
	ID        string    `json:"id"         db:"id"`
	TenantID  string    `json:"tenant_id"  db:"tenant_id"`
	Name      string    `json:"name"       db:"name"`
	Location  string    `json:"location"   db:"location"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// User is a human account associated with a tenant.
type User struct {
	ID           string    `json:"id"            db:"id"`
	TenantID     string    `json:"tenant_id"     db:"tenant_id"`
	Email        string    `json:"email"         db:"email"`
	PasswordHash string    `json:"-"             db:"password_hash"`
	Role         string    `json:"role"          db:"role"`
	DisplayName  string    `json:"display_name"  db:"display_name"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

// Device is an edge agent device registered to a tenant site.
type Device struct {
	ID              string       `json:"id"               db:"id"`
	TenantID        string       `json:"tenant_id"        db:"tenant_id"`
	SiteID          string       `json:"site_id"          db:"site_id"`
	DeviceID        string       `json:"device_id"        db:"device_id"`
	DisplayName     string       `json:"display_name"     db:"display_name"`
	Status          DeviceStatus `json:"status"           db:"status"`
	LastSeen        *time.Time   `json:"last_seen"        db:"last_seen"`
	FirmwareVersion string       `json:"firmware_version" db:"firmware_version"`
	IPAddress       string       `json:"ip_address"       db:"ip_address"`
	CreatedAt       time.Time    `json:"created_at"       db:"created_at"`
}

// Endpoint is a service port exposed by a device.
type Endpoint struct {
	ID       string       `json:"id"        db:"id"`
	DeviceID string       `json:"device_id" db:"device_id"`
	Type     EndpointType `json:"type"      db:"type"`
	Port     int          `json:"port"      db:"port"`
	Label    string       `json:"label"     db:"label"`
	Protocol string       `json:"protocol"  db:"protocol"`
	Enabled  bool         `json:"enabled"   db:"enabled"`
}

// SessionTelemetry captures the health of an exported or web session from the
// latest probe source.
type SessionTelemetry struct {
	ConnectionStatus string     `json:"connection_status,omitempty"`
	LatencyMS        *int       `json:"latency_ms,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
	ProbeSource      string     `json:"probe_source,omitempty"`
}

// Session tracks an active or historical remote access session.
type Session struct {
	ID                   string            `json:"id"                     db:"id"`
	DeviceID             string            `json:"device_id"              db:"device_id"`
	EndpointID           string            `json:"endpoint_id"            db:"endpoint_id"`
	UserID               string            `json:"user_id"                db:"user_id"`
	TenantID             string            `json:"tenant_id"              db:"tenant_id"`
	Status               SessionStatus     `json:"status"                 db:"status"`
	LocalPort            int               `json:"local_port"             db:"local_port"`
	RemotePort           int               `json:"remote_port,omitempty"  db:"remote_port"`
	RemoteHost           string            `json:"remote_host,omitempty"`
	DeliveryMode         DeliveryMode      `json:"delivery_mode"          db:"delivery_mode"`
	TTLSeconds           int               `json:"ttl_seconds"            db:"ttl_seconds"`
	IdleTimeoutSeconds   int               `json:"idle_timeout_seconds"   db:"idle_timeout_seconds"`
	StartedAt            time.Time         `json:"started_at"             db:"started_at"`
	ExpiresAt            *time.Time        `json:"expires_at"             db:"expires_at"`
	StoppedAt            *time.Time        `json:"stopped_at"             db:"stopped_at"`
	StopReason           string            `json:"stop_reason"            db:"stop_reason"`
	TunnelURL            string            `json:"tunnel_url,omitempty"   db:"tunnel_url"`
	AuditData            json.RawMessage   `json:"audit_data"             db:"audit_data"`
	LastActivityAt       *time.Time        `json:"last_activity_at,omitempty" db:"last_activity_at"`
	Telemetry            *SessionTelemetry `json:"telemetry,omitempty"`
	Device               *Device           `json:"device,omitempty"`
	Endpoint             *Endpoint         `json:"endpoint,omitempty"`
	User                 *User             `json:"user,omitempty"`
}

// ExportHistory records completed export-mode sessions for auditing.
type ExportHistory struct {
	ID            string          `json:"id"             db:"id"`
	SessionID     string          `json:"session_id"     db:"session_id"`
	UserID        string          `json:"user_id"        db:"user_id"`
	DeviceID      string          `json:"device_id"      db:"device_id"`
	EndpointID    string          `json:"endpoint_id"    db:"endpoint_id"`
	TenantID      string          `json:"tenant_id"      db:"tenant_id"`
	SiteID        string          `json:"site_id"        db:"site_id"`
	StartedAt     time.Time       `json:"started_at"     db:"started_at"`
	StoppedAt     *time.Time      `json:"stopped_at"     db:"stopped_at"`
	StopReason    string          `json:"stop_reason"    db:"stop_reason"`
	LocalBindPort int             `json:"local_bind_port" db:"local_bind_port"`
	DeliveryMode  DeliveryMode    `json:"delivery_mode"  db:"delivery_mode"`
	DurationSeconds int           `json:"duration_seconds,omitempty" db:"duration_seconds"`
	BytesTransferred int64        `json:"bytes_transferred,omitempty" db:"bytes_transferred"`
	Metadata      json.RawMessage `json:"metadata"       db:"metadata"`
	Telemetry     *SessionTelemetry `json:"telemetry,omitempty"`
	Device        *Device         `json:"device,omitempty"`
	Endpoint      *Endpoint       `json:"endpoint,omitempty"`
	User          *User           `json:"user,omitempty"`
}

// BridgeProfile holds serial-to-TCP bridge configuration for a device.
type BridgeProfile struct {
	ID         string    `json:"id"          db:"id"`
	DeviceID   string    `json:"device_id"   db:"device_id"`
	SerialPort string    `json:"serial_port" db:"serial_port"`
	BaudRate   int       `json:"baud_rate"   db:"baud_rate"`
	Parity     string    `json:"parity"      db:"parity"`
	StopBits   int       `json:"stop_bits"   db:"stop_bits"`
	DataBits   int       `json:"data_bits"   db:"data_bits"`
	TCPPort    int       `json:"tcp_port"    db:"tcp_port"`
	Status     string    `json:"status"      db:"status"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

// AuditLog records every significant action taken in the system.
type AuditLog struct {
	ID           string          `json:"id"            db:"id"`
	TenantID     string          `json:"tenant_id"     db:"tenant_id"`
	UserID       string          `json:"user_id"       db:"user_id"`
	Action       string          `json:"action"        db:"action"`
	ResourceType string          `json:"resource_type" db:"resource_type"`
	ResourceID   string          `json:"resource_id"   db:"resource_id"`
	Metadata     json.RawMessage `json:"metadata"      db:"metadata"`
	CreatedAt    time.Time       `json:"created_at"    db:"created_at"`
	IPAddress    string          `json:"ip_address"    db:"ip_address"`
}

// AgentConnection tracks a live WebSocket connection from an agent device.
type AgentConnection struct {
	DeviceID    string    `json:"device_id"    db:"device_id"`
	TenantID    string    `json:"tenant_id"    db:"tenant_id"`
	ConnectedAt time.Time `json:"connected_at" db:"connected_at"`
	LastPing    time.Time `json:"last_ping"    db:"last_ping"`
	WSConnID    string    `json:"ws_conn_id"   db:"ws_conn_id"`
}

// APIResponse is the standard envelope for all HTTP responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

// MetaInfo carries pagination metadata.
type MetaInfo struct {
	Total  int `json:"total"`
	Page   int `json:"page"`
	Limit  int `json:"limit"`
}
