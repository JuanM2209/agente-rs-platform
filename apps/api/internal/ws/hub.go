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
}

// NewAgentHub constructs an AgentHub with the provided authentication secret.
func NewAgentHub(secret string) *AgentHub {
	return &AgentHub{
		agents: make(map[string]*AgentConn),
		secret: secret,
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

	h.Register(conn)

	ctx, cancel := context.WithCancel(r.Context())

	go conn.writePump(cancel)
	conn.readPump(ctx, cancel)
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

		var ack AgentAck
		if jsonErr := json.Unmarshal(message, &ack); jsonErr == nil && ack.ID != "" {
			log.Debug().
				Str("device_id", c.DeviceID).
				Str("msg_id", ack.ID).
				Bool("success", ack.Success).
				Msg("received agent ack")
			continue
		}

		log.Debug().
			Str("device_id", c.DeviceID).
			RawJSON("message", message).
			Msg("received agent message")
	}
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
