package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod is how often we send pings to the peer (must be less than pongWait).
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from the peer.
	maxMessageSize = 512 * 1024 // 512 KB

	// backoffMax caps the exponential reconnect delay.
	backoffMax = 60 * time.Second
)

// ConnState represents the current connection state.
type ConnState int

const (
	StateDisconnected ConnState = iota
	StateConnecting
	StateConnected
)

// Hub manages the WebSocket connection to the control plane, including
// reconnect logic, ping/pong keepalive, and thread-safe send/receive.
type Hub struct {
	url    string
	header http.Header
	log    zerolog.Logger

	mu    sync.Mutex
	conn  *websocket.Conn
	state ConnState

	recvCh chan AgentMessage
	sendCh chan AgentMessage
	stopCh chan struct{}
}

// NewHub creates a Hub that will connect to the given URL.
// header is used for authentication (e.g. Authorization bearer token).
func NewHub(url string, header http.Header, log zerolog.Logger) *Hub {
	return &Hub{
		url:    url,
		header: header,
		log:    log,
		recvCh: make(chan AgentMessage, 64),
		sendCh: make(chan AgentMessage, 64),
		stopCh: make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection with exponential backoff.
// It blocks until connected or the stop channel is closed.
func (h *Hub) Connect() error {
	attempt := 0
	backoff := time.Second

	for {
		select {
		case <-h.stopCh:
			return fmt.Errorf("hub stopped before connection established")
		default:
		}

		h.setState(StateConnecting)
		h.log.Info().Str("url", h.url).Int("attempt", attempt+1).Msg("connecting to control plane")

		conn, _, err := websocket.DefaultDialer.Dial(h.url, h.header)
		if err != nil {
			h.setState(StateDisconnected)
			h.log.Warn().Err(err).Dur("backoff", backoff).Msg("connection failed, retrying")

			select {
			case <-time.After(backoff):
			case <-h.stopCh:
				return fmt.Errorf("hub stopped during backoff")
			}

			attempt++
			backoff = minDuration(backoff*2, backoffMax)
			continue
		}

		h.mu.Lock()
		h.conn = conn
		h.state = StateConnected
		h.mu.Unlock()

		h.log.Info().Str("url", h.url).Msg("connected to control plane")
		return nil
	}
}

// Run starts the read and write pumps. It blocks until the hub is stopped.
// On disconnect it will attempt to reconnect with exponential backoff.
func (h *Hub) Run() {
	for {
		select {
		case <-h.stopCh:
			h.closeConn()
			return
		default:
		}

		readDone := make(chan struct{})
		go func() {
			h.readPump()
			close(readDone)
		}()

		h.writePump(readDone)

		select {
		case <-h.stopCh:
			return
		default:
			h.log.Warn().Msg("disconnected from control plane, will reconnect")
			h.setState(StateDisconnected)
			if err := h.Connect(); err != nil {
				return
			}
		}
	}
}

// Send queues a message to be written to the WebSocket. Non-blocking; drops
// the message if the send buffer is full (caller should handle this).
func (h *Hub) Send(msg AgentMessage) error {
	select {
	case h.sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full, message dropped (type=%s)", msg.Type)
	}
}

// Recv returns the channel on which inbound messages are delivered.
func (h *Hub) Recv() <-chan AgentMessage {
	return h.recvCh
}

// Stop signals the hub to shut down cleanly.
func (h *Hub) Stop() {
	select {
	case <-h.stopCh:
		// already stopped
	default:
		close(h.stopCh)
	}
}

// State returns the current connection state.
func (h *Hub) State() ConnState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

// SendJSON is a convenience wrapper that marshals v and sends it.
func (h *Hub) SendJSON(msgType CommandType, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return h.Send(AgentMessage{
		Type:      msgType,
		Payload:   raw,
		Timestamp: time.Now().UTC(),
	})
}

// readPump reads messages from the WebSocket and puts them on recvCh.
func (h *Hub) readPump() {
	conn := h.activeConn()
	if conn == nil {
		return
	}

	defer func() {
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				h.log.Error().Err(err).Msg("unexpected WebSocket close")
			} else {
				h.log.Info().Err(err).Msg("WebSocket read ended")
			}
			return
		}

		var msg AgentMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.log.Warn().Err(err).Msg("failed to unmarshal incoming message")
			continue
		}

		select {
		case h.recvCh <- msg:
		case <-h.stopCh:
			return
		}
	}
}

// writePump writes queued messages to the WebSocket and sends periodic pings.
// It returns when readDone is closed (read pump exited) or the hub is stopped.
func (h *Hub) writePump(readDone <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-readDone:
			return

		case <-h.stopCh:
			conn := h.activeConn()
			if conn != nil {
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			}
			return

		case msg := <-h.sendCh:
			conn := h.activeConn()
			if conn == nil {
				h.log.Warn().Msg("send attempted with no active connection, dropping message")
				continue
			}
			data, err := json.Marshal(msg)
			if err != nil {
				h.log.Error().Err(err).Msg("failed to marshal outbound message")
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				h.log.Error().Err(err).Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			conn := h.activeConn()
			if conn == nil {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.log.Warn().Err(err).Msg("ping failed")
				return
			}
		}
	}
}

func (h *Hub) setState(s ConnState) {
	h.mu.Lock()
	h.state = s
	h.mu.Unlock()
}

func (h *Hub) activeConn() *websocket.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.conn
}

func (h *Hub) closeConn() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn != nil {
		_ = h.conn.Close()
		h.conn = nil
	}
	h.state = StateDisconnected
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
