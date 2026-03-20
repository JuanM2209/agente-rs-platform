package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/nucleus-portal/agent/internal/config"
	"github.com/nucleus-portal/agent/internal/inventory"
	"github.com/nucleus-portal/agent/internal/mbusd"
	"github.com/nucleus-portal/agent/internal/ws"
)

const agentVersion = "1.0.0"

// SessionState tracks a live port-forward session managed by the agent.
type SessionState struct {
	ID        string
	Port      int
	Protocol  string
	StartedAt time.Time
	TTL       time.Duration
	cancel    context.CancelFunc
}

// Agent is the central coordinator: it owns the WebSocket hub, inventory
// scanner, and active session/bridge registries.
type Agent struct {
	config    *config.Config
	hub       *ws.Hub
	inventory *inventory.Scanner
	log       zerolog.Logger

	sessions map[string]*SessionState
	bridges  map[string]*mbusd.MBUSDProcess
	mu       sync.RWMutex

	startedAt time.Time
	stopCh    chan struct{}
}

// New constructs an Agent from the supplied config and logger. The caller must
// call Run() to begin operation.
func New(cfg *config.Config, hub *ws.Hub, inv *inventory.Scanner, log zerolog.Logger) *Agent {
	return &Agent{
		config:    cfg,
		hub:       hub,
		inventory: inv,
		log:       log,
		sessions:  make(map[string]*SessionState),
		bridges:   make(map[string]*mbusd.MBUSDProcess),
		startedAt: time.Now().UTC(),
		stopCh:    make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection and sends the registration
// message. It uses the Hub's built-in exponential backoff.
func (a *Agent) Connect() error {
	if err := a.hub.Connect(); err != nil {
		return fmt.Errorf("hub connect: %w", err)
	}
	return a.sendRegistration()
}

// Reconnect is called after a disconnect. It delegates backoff to the Hub,
// then re-registers the agent with the control plane.
func (a *Agent) Reconnect() error {
	a.log.Info().Msg("attempting reconnect")
	if err := a.hub.Connect(); err != nil {
		return fmt.Errorf("hub reconnect: %w", err)
	}
	return a.sendRegistration()
}

// Run starts the hub's read/write pumps in a goroutine, then enters the main
// message dispatch loop. It returns when Stop() is called.
func (a *Agent) Run(ctx context.Context) {
	go a.hub.Run()

	go a.heartbeatLoop(ctx)

	a.log.Info().Str("device_id", a.config.DeviceID).Msg("agent running")

	for {
		select {
		case <-ctx.Done():
			a.log.Info().Msg("context cancelled, shutting down agent")
			a.hub.Stop()
			return

		case <-a.stopCh:
			a.log.Info().Msg("stop signal received, shutting down agent")
			a.hub.Stop()
			return

		case msg, ok := <-a.hub.Recv():
			if !ok {
				a.log.Warn().Msg("receive channel closed")
				return
			}
			a.handleCommand(msg)
		}
	}
}

// Stop signals the agent to shut down.
func (a *Agent) Stop() {
	select {
	case <-a.stopCh:
	default:
		close(a.stopCh)
	}
}

// handleCommand dispatches an inbound AgentMessage to the appropriate handler.
func (a *Agent) handleCommand(msg ws.AgentMessage) {
	a.log.Debug().
		Str("id", msg.ID).
		Str("type", string(msg.Type)).
		Msg("received command")

	var err error

	switch msg.Type {
	case ws.CommandSyncInventory:
		err = a.handleSyncInventory()

	case ws.CommandStartSession:
		var payload ws.StartSessionPayload
		if jsonErr := json.Unmarshal(msg.Payload, &payload); jsonErr != nil {
			a.sendAck(msg.ID, false, fmt.Sprintf("invalid payload: %v", jsonErr))
			return
		}
		err = a.handleStartSession(msg.ID, payload)

	case ws.CommandStopSession:
		var payload ws.StopSessionPayload
		if jsonErr := json.Unmarshal(msg.Payload, &payload); jsonErr != nil {
			a.sendAck(msg.ID, false, fmt.Sprintf("invalid payload: %v", jsonErr))
			return
		}
		err = a.handleStopSession(msg.ID, payload)

	case ws.CommandStartMBUSD:
		var payload ws.StartMBUSDPayload
		if jsonErr := json.Unmarshal(msg.Payload, &payload); jsonErr != nil {
			a.sendAck(msg.ID, false, fmt.Sprintf("invalid payload: %v", jsonErr))
			return
		}
		err = a.handleStartMBUSD(msg.ID, payload)

	case ws.CommandStopMBUSD:
		var payload ws.StopMBUSDPayload
		if jsonErr := json.Unmarshal(msg.Payload, &payload); jsonErr != nil {
			a.sendAck(msg.ID, false, fmt.Sprintf("invalid payload: %v", jsonErr))
			return
		}
		err = a.handleStopMBUSD(msg.ID, payload)

	case ws.CommandHealthPing:
		err = a.handleHealthPing(msg.ID)

	default:
		a.log.Warn().Str("type", string(msg.Type)).Msg("unknown command type, ignoring")
		return
	}

	if err != nil {
		a.log.Error().Err(err).Str("id", msg.ID).Str("type", string(msg.Type)).Msg("command handler error")
		a.sendAck(msg.ID, false, err.Error())
	}
}

// handleSyncInventory triggers a fresh inventory scan and pushes the result.
func (a *Agent) handleSyncInventory() error {
	a.log.Info().Msg("sync_inventory requested")
	return a.sendInventoryUpdate()
}

// handleStartSession creates a new session, enforcing the concurrency cap.
func (a *Agent) handleStartSession(cmdID string, payload ws.StartSessionPayload) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.sessions) >= a.config.MaxConcurrentSessions {
		return fmt.Errorf("max concurrent sessions (%d) reached", a.config.MaxConcurrentSessions)
	}
	if _, exists := a.sessions[payload.SessionID]; exists {
		return fmt.Errorf("session %s already exists", payload.SessionID)
	}

	ttl := time.Duration(payload.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), ttl)
	session := &SessionState{
		ID:        payload.SessionID,
		Port:      payload.TargetPort,
		Protocol:  payload.Protocol,
		StartedAt: time.Now().UTC(),
		TTL:       ttl,
		cancel:    cancel,
	}
	a.sessions[payload.SessionID] = session

	// Auto-expire the session when TTL elapses.
	go func() {
		<-ctx.Done()
		a.mu.Lock()
		if _, ok := a.sessions[payload.SessionID]; ok {
			delete(a.sessions, payload.SessionID)
			a.log.Info().Str("session_id", payload.SessionID).Msg("session expired via TTL")
		}
		a.mu.Unlock()
	}()

	a.log.Info().
		Str("session_id", payload.SessionID).
		Int("port", payload.TargetPort).
		Str("protocol", payload.Protocol).
		Dur("ttl", ttl).
		Msg("session started")

	a.sendAck(cmdID, true, "")

	return a.hub.SendJSON(ws.EventSessionStarted, map[string]interface{}{
		"session_id": payload.SessionID,
		"port":       payload.TargetPort,
		"protocol":   payload.Protocol,
		"started_at": session.StartedAt,
	})
}

// handleStopSession terminates a named session.
func (a *Agent) handleStopSession(cmdID string, payload ws.StopSessionPayload) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, exists := a.sessions[payload.SessionID]
	if !exists {
		return fmt.Errorf("session %s not found", payload.SessionID)
	}

	session.cancel()
	delete(a.sessions, payload.SessionID)

	a.log.Info().Str("session_id", payload.SessionID).Msg("session stopped")
	a.sendAck(cmdID, true, "")

	return a.hub.SendJSON(ws.EventSessionStopped, map[string]interface{}{
		"session_id": payload.SessionID,
		"stopped_at": time.Now().UTC(),
	})
}

// handleStartMBUSD spawns an mbusd bridge process for a serial port.
func (a *Agent) handleStartMBUSD(cmdID string, payload ws.StartMBUSDPayload) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.bridges[payload.BridgeID]; exists {
		return fmt.Errorf("bridge %s is already running", payload.BridgeID)
	}

	proc := mbusd.New(mbusd.Config{
		BridgeID:   payload.BridgeID,
		SerialPort: payload.SerialPort,
		BaudRate:   payload.BaudRate,
		TCPPort:    payload.TCPPort,
		Parity:     payload.Parity,
	}, a.log.With().Str("bridge_id", payload.BridgeID).Logger())

	if err := proc.Start(); err != nil {
		return fmt.Errorf("failed to start mbusd bridge %s: %w", payload.BridgeID, err)
	}

	a.bridges[payload.BridgeID] = proc

	a.log.Info().
		Str("bridge_id", payload.BridgeID).
		Str("serial_port", payload.SerialPort).
		Int("tcp_port", payload.TCPPort).
		Msg("mbusd bridge started")

	a.sendAck(cmdID, true, "")

	return a.hub.SendJSON(ws.EventMBUSDStarted, map[string]interface{}{
		"bridge_id":   payload.BridgeID,
		"serial_port": payload.SerialPort,
		"tcp_port":    payload.TCPPort,
		"started_at":  proc.StartedAt,
	})
}

// handleStopMBUSD terminates a running mbusd bridge.
func (a *Agent) handleStopMBUSD(cmdID string, payload ws.StopMBUSDPayload) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	proc, exists := a.bridges[payload.BridgeID]
	if !exists {
		return fmt.Errorf("bridge %s not found", payload.BridgeID)
	}

	if err := proc.Stop(); err != nil {
		return fmt.Errorf("failed to stop bridge %s: %w", payload.BridgeID, err)
	}

	delete(a.bridges, payload.BridgeID)

	a.log.Info().Str("bridge_id", payload.BridgeID).Msg("mbusd bridge stopped")
	a.sendAck(cmdID, true, "")

	return a.hub.SendJSON(ws.EventMBUSDStopped, map[string]interface{}{
		"bridge_id":  payload.BridgeID,
		"stopped_at": time.Now().UTC(),
	})
}

// handleHealthPing responds to a health check from the control plane.
func (a *Agent) handleHealthPing(cmdID string) error {
	a.log.Debug().Msg("health ping received")
	a.sendAck(cmdID, true, "")
	return nil
}

// sendAck sends an acknowledgement message for a command.
func (a *Agent) sendAck(id string, success bool, errMsg string) {
	ack := ws.AgentAck{
		ID:      id,
		Success: success,
		Error:   errMsg,
	}
	raw, err := json.Marshal(ack)
	if err != nil {
		a.log.Error().Err(err).Msg("failed to marshal ack")
		return
	}
	if sendErr := a.hub.Send(ws.AgentMessage{
		ID:        uuid.New().String(),
		Type:      ws.EventAck,
		Payload:   raw,
		Timestamp: time.Now().UTC(),
	}); sendErr != nil {
		a.log.Error().Err(sendErr).Msg("failed to send ack")
	}
}

// sendInventoryUpdate pushes the current inventory snapshot to the control plane.
func (a *Agent) sendInventoryUpdate() error {
	inv := a.inventory.Current()
	if inv == nil {
		return fmt.Errorf("inventory not yet available")
	}
	return a.hub.SendJSON(ws.EventInventoryUpdate, inv)
}

// sendRegistration sends the initial registration message after connecting.
func (a *Agent) sendRegistration() error {
	reg := ws.RegistrationMessage{
		DeviceID: a.config.DeviceID,
		TenantID: a.config.TenantID,
		Secret:   a.config.AgentSecret,
		Version:  agentVersion,
	}
	raw, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}
	return a.hub.Send(ws.AgentMessage{
		ID:        uuid.New().String(),
		Type:      ws.EventRegistration,
		Payload:   raw,
		Timestamp: time.Now().UTC(),
	})
}

// heartbeatLoop sends periodic heartbeat messages to the control plane.
func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.sendHeartbeat()
		}
	}
}

// sendHeartbeat publishes a heartbeat event with current runtime stats.
func (a *Agent) sendHeartbeat() {
	a.mu.RLock()
	activeSessions := len(a.sessions)
	activeBridges := len(a.bridges)
	a.mu.RUnlock()

	payload := ws.HeartbeatPayload{
		DeviceID:       a.config.DeviceID,
		TenantID:       a.config.TenantID,
		Uptime:         time.Since(a.startedAt).Seconds(),
		ActiveSessions: activeSessions,
		ActiveBridges:  activeBridges,
		Timestamp:      time.Now().UTC(),
	}

	if err := a.hub.SendJSON(ws.EventHeartbeat, payload); err != nil {
		a.log.Warn().Err(err).Msg("failed to send heartbeat")
	}
}

// ActiveSessionCount returns the number of currently live sessions.
func (a *Agent) ActiveSessionCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.sessions)
}

// ActiveBridgeCount returns the number of currently running mbusd bridges.
func (a *Agent) ActiveBridgeCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.bridges)
}
