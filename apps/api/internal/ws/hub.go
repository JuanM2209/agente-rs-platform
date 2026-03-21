package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Agent connections are authenticated via secret header; origin is irrelevant.
		return true
	},
}

// AgentConn represents a single live WebSocket connection from an edge agent.
type AgentConn struct {
	DeviceID string
	TenantID string
	ConnID   string
	conn     *websocket.Conn
	send     chan []byte
	hub      *AgentHub
}

// AgentHub maintains the registry of connected agents and routes commands to them.
type AgentHub struct {
	mu      sync.RWMutex
	agents  map[string]*AgentConn // keyed by DeviceID
	secret  string
	db      *pgxpool.Pool
}

// NewAgentHub constructs an AgentHub with the provided authentication secret.
func NewAgentHub(secret string, db *pgxpool.Pool) *AgentHub {
	return &AgentHub{
		agents: make(map[string]*AgentConn),
		secret: secret,
		db:     db,
	}
}

// Register adds a new agent connection to the hub.
func (h *AgentHub) Register(conn *AgentConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close any stale connection for the same device.
	if existing, ok := h.agents[conn.DeviceID]; ok {
		log.Warn().
			Str("device_id", conn.DeviceID).
			Msg("replacing existing agent connection")
		close(existing.send)
	}

	h.agents[conn.DeviceID] = conn
	log.Info().
		Str("device_id", conn.DeviceID).
		Str("conn_id", conn.ConnID).
		Msg("agent registered")
}

// Unregister removes an agent connection from the hub.
func (h *AgentHub) Unregister(conn *AgentConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if current, ok := h.agents[conn.DeviceID]; ok && current.ConnID == conn.ConnID {
		delete(h.agents, conn.DeviceID)
		log.Info().
			Str("device_id", conn.DeviceID).
			Str("conn_id", conn.ConnID).
			Msg("agent unregistered")
		h.touchDevice(conn.DeviceID, "offline", time.Now().UTC(), nil)
	}
}

// SendCommand serialises and sends a command to the agent identified by deviceID.
func (h *AgentHub) SendCommand(deviceID string, cmd AgentMessage) error {
	h.mu.RLock()
	conn, ok := h.agents[deviceID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent not connected: %s", deviceID)
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	select {
	case conn.send <- data:
		return nil
	default:
		return fmt.Errorf("agent send buffer full for device: %s", deviceID)
	}
}

// BroadcastToTenant sends a message to all agents belonging to the given tenant.
func (h *AgentHub) BroadcastToTenant(tenantID string, msg AgentMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("broadcast marshal error")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conn := range h.agents {
		if conn.TenantID != tenantID {
			continue
		}
		select {
		case conn.send <- data:
		default:
			log.Warn().
				Str("device_id", conn.DeviceID).
				Msg("dropped broadcast message: send buffer full")
		}
	}
}

// IsConnected reports whether the given device has an active connection.
func (h *AgentHub) IsConnected(deviceID string) bool {
	h.mu.RLock()
	_, ok := h.agents[deviceID]
	h.mu.RUnlock()
	return ok
}

// HandleAgentConnection upgrades an HTTP request to a WebSocket and manages the
// full lifecycle of the agent connection. The caller must have already validated
// the AGENT_WS_SECRET before invoking this handler.
func (h *AgentHub) HandleAgentConnection(w http.ResponseWriter, r *http.Request) {
	deviceID := r.Header.Get("X-Device-ID")
	tenantID := r.Header.Get("X-Tenant-ID")

	if deviceID == "" || tenantID == "" {
		http.Error(w, "X-Device-ID and X-Tenant-ID headers are required", http.StatusBadRequest)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("websocket upgrade failed")
		return
	}

	conn := &AgentConn{
		DeviceID: deviceID,
		TenantID: tenantID,
		ConnID:   uuid.New().String(),
		conn:     wsConn,
		send:     make(chan []byte, 256),
		hub:      h,
	}

	if err := h.ensureDeviceRecord(deviceID, tenantID); err != nil {
		log.Warn().
			Err(err).
			Str("device_id", deviceID).
			Str("tenant_id", tenantID).
			Msg("failed to ensure device record before agent registration")
	}

	h.Register(conn)

	ctx, cancel := context.WithCancel(r.Context())

	go conn.writePump(cancel)
	conn.readPump(ctx, cancel)
}

func (h *AgentHub) ensureDeviceRecord(deviceStringID, tenantID string) error {
	if h.db == nil {
		return nil
	}

	const lookupQ = `SELECT id FROM devices WHERE device_id = $1 LIMIT 1`

	var existingID string
	err := h.db.QueryRow(context.Background(), lookupQ, deviceStringID).Scan(&existingID)
	if err == nil {
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}

	const insertQ = `
		INSERT INTO devices
			(id, tenant_id, device_id, display_name, status, hardware_model, serial_number, created_at, updated_at, last_seen)
		VALUES
			($1, $2, $3, $4, 'online', 'Nucleus Remote-S', $5, NOW(), NOW(), NOW())
		ON CONFLICT (device_id) DO NOTHING`

	displayName := fmt.Sprintf("Auto-registered %s", deviceStringID)
	_, err = h.db.Exec(
		context.Background(),
		insertQ,
		uuid.New().String(),
		tenantID,
		deviceStringID,
		displayName,
		deviceStringID,
	)

	return err
}

// readPump processes inbound messages from the agent until the connection closes.
func (c *AgentConn) readPump(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		cancel()
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("device_id", c.DeviceID).Msg("websocket read error")
			}
			return
		}

		var msg AgentMessage
		if jsonErr := json.Unmarshal(message, &msg); jsonErr != nil {
			log.Warn().Err(jsonErr).Str("device_id", c.DeviceID).Msg("failed to decode agent message")
			continue
		}

		switch msg.Type {
		case EventAck:
			var ack AgentAck
			if jsonErr := json.Unmarshal(msg.Payload, &ack); jsonErr != nil {
				log.Warn().Err(jsonErr).Str("device_id", c.DeviceID).Msg("failed to decode agent ack payload")
				continue
			}
			log.Debug().
				Str("device_id", c.DeviceID).
				Str("msg_id", ack.ID).
				Bool("success", ack.Success).
				Str("error", ack.Error).
				Msg("received agent ack")
		default:
			c.hub.handleAgentEvent(c, msg)
		}
	}
}

type inventoryEventPayload struct {
	Endpoints []struct {
		Port       int    `json:"port,omitempty"`
		Protocol   string `json:"protocol,omitempty"`
		Type       string `json:"type"`
		Label      string `json:"label"`
		SerialPort string `json:"serial_port,omitempty"`
	} `json:"endpoints"`
	Capabilities []string  `json:"capabilities"`
	LocalIP      string    `json:"local_ip,omitempty"`
	ScannedAt    time.Time `json:"scanned_at"`
}

func (h *AgentHub) handleAgentEvent(conn *AgentConn, msg AgentMessage) {
	switch msg.Type {
	case EventRegistration:
		var payload RegistrationMessage
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Warn().Err(err).Str("device_id", conn.DeviceID).Msg("invalid registration payload")
			return
		}
		h.touchDevice(conn.DeviceID, "online", time.Now().UTC(), nil)
		log.Info().
			Str("device_id", conn.DeviceID).
			Str("tenant_id", payload.TenantID).
			Str("version", payload.Version).
			Msg("agent registration received")

	case EventHeartbeat:
		var payload HeartbeatPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Warn().Err(err).Str("device_id", conn.DeviceID).Msg("invalid heartbeat payload")
			return
		}
		h.touchDevice(conn.DeviceID, "online", payload.Timestamp.UTC(), nil)

	case EventInventoryUpdate:
		var payload inventoryEventPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Warn().Err(err).Str("device_id", conn.DeviceID).Msg("invalid inventory payload")
			return
		}
		if payload.ScannedAt.IsZero() {
			payload.ScannedAt = time.Now().UTC()
		}
		if err := h.persistInventory(conn.DeviceID, payload); err != nil {
			log.Warn().Err(err).Str("device_id", conn.DeviceID).Msg("failed to persist inventory update")
			return
		}
		log.Info().
			Str("device_id", conn.DeviceID).
			Int("endpoints", len(payload.Endpoints)).
			Msg("inventory update persisted")

	case EventMBUSDStarted, EventMBUSDStopped, EventSessionStarted, EventSessionStopped:
		log.Info().
			Str("device_id", conn.DeviceID).
			Str("event_type", string(msg.Type)).
			RawJSON("payload", msg.Payload).
			Msg("agent event received")

	default:
		log.Debug().
			Str("device_id", conn.DeviceID).
			Str("event_type", string(msg.Type)).
			RawJSON("payload", msg.Payload).
			Msg("unhandled agent event")
	}
}

func (h *AgentHub) persistInventory(deviceStringID string, payload inventoryEventPayload) error {
	if h.db == nil {
		return nil
	}

	var internalID string
	if err := h.db.QueryRow(context.Background(), `SELECT id FROM devices WHERE device_id = $1 LIMIT 1`, deviceStringID).Scan(&internalID); err != nil {
		return err
	}

	h.touchDevice(deviceStringID, "online", time.Now().UTC(), &payload.ScannedAt)

	if payload.LocalIP != "" {
		if _, err := h.db.Exec(
			context.Background(),
			`UPDATE devices SET ip_address = $1::inet, updated_at = NOW() WHERE device_id = $2`,
			payload.LocalIP,
			deviceStringID,
		); err != nil {
			log.Warn().Err(err).Str("device_id", deviceStringID).Str("ip_address", payload.LocalIP).Msg("failed to update device ip_address")
		}
	}

	for _, ep := range payload.Endpoints {
		if ep.Type == "BRIDGE" && ep.SerialPort != "" {
			if err := h.ensureBridgeProfile(internalID, ep.SerialPort); err != nil {
				log.Warn().Err(err).Str("device_id", deviceStringID).Str("serial_port", ep.SerialPort).Msg("failed to ensure bridge profile")
			}
			continue
		}

		if ep.Port <= 0 {
			continue
		}

		protocol := ep.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		const upsertEndpoint = `
			INSERT INTO endpoints
				(id, device_id, type, port, label, protocol, description, enabled, discovered_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NULL, true, $7, NOW())
			ON CONFLICT (device_id, port)
			DO UPDATE SET
				type = EXCLUDED.type,
				label = EXCLUDED.label,
				protocol = EXCLUDED.protocol,
				enabled = true,
				discovered_at = EXCLUDED.discovered_at`

		if _, err := h.db.Exec(
			context.Background(),
			upsertEndpoint,
			uuid.New().String(),
			internalID,
			ep.Type,
			ep.Port,
			ep.Label,
			protocol,
			payload.ScannedAt.UTC(),
		); err != nil {
			log.Warn().Err(err).Str("device_id", deviceStringID).Int("port", ep.Port).Msg("failed to upsert endpoint")
		}
	}

	return nil
}

func (h *AgentHub) ensureBridgeProfile(deviceInternalID, serialPort string) error {
	if h.db == nil {
		return nil
	}

	var existingID string
	err := h.db.QueryRow(
		context.Background(),
		`SELECT id FROM bridge_profiles WHERE device_id = $1 AND serial_port = $2 LIMIT 1`,
		deviceInternalID,
		serialPort,
	).Scan(&existingID)
	if err == nil {
		return nil
	}

	const insertQ = `
		INSERT INTO bridge_profiles
			(id, device_id, serial_port, baud_rate, parity, stop_bits, data_bits, tcp_port, status, created_at)
		VALUES ($1, $2, $3, 9600, 'N', 1, 8, NULL, 'idle', NOW())`

	_, err = h.db.Exec(context.Background(), insertQ, uuid.New().String(), deviceInternalID, serialPort)
	return err
}

func (h *AgentHub) touchDevice(deviceStringID, status string, lastSeen time.Time, inventoryUpdatedAt *time.Time) {
	if h.db == nil {
		return
	}

	if inventoryUpdatedAt != nil {
		_, _ = h.db.Exec(
			context.Background(),
			`UPDATE devices SET status = $1, last_seen = $2, inventory_updated_at = $3, updated_at = NOW() WHERE device_id = $4`,
			status,
			lastSeen,
			inventoryUpdatedAt.UTC(),
			deviceStringID,
		)
		return
	}

	_, _ = h.db.Exec(
		context.Background(),
		`UPDATE devices SET status = $1, last_seen = $2, updated_at = NOW() WHERE device_id = $3`,
		status,
		lastSeen,
		deviceStringID,
	)
}

// writePump delivers outbound messages to the agent and sends periodic pings.
func (c *AgentConn) writePump(cancel context.CancelFunc) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		cancel()
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error().Err(err).Str("device_id", c.DeviceID).Msg("websocket write error")
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
