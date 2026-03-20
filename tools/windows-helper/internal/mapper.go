package internal

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	probeInterval      = 10 * time.Second
	probeTimeout       = 3 * time.Second
	degradedThreshold  = 250 * time.Millisecond
)

// Mapping represents a single active TCP port-forward from a local port to a
// remote Nucleus session endpoint.
type Mapping struct {
	SessionID        string    `json:"session_id"`
	LocalPort        int       `json:"local_port"`
	RemoteHost       string    `json:"remote_host"`
	RemotePort       int       `json:"remote_port"`
	StartedAt        time.Time `json:"started_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	BytesFwd         int64     `json:"bytes_forwarded"`
	ConnectionStatus string    `json:"connection_status"`
	LatencyMS        int       `json:"latency_ms"`
	LastCheckedAt    time.Time `json:"last_checked_at"`
	LastError        string    `json:"last_error,omitempty"`

	listener net.Listener
	cancel   context.CancelFunc
}

// TTL returns how long until the mapping expires. A negative duration means
// the session has already expired.
func (m *Mapping) TTL() time.Duration {
	return time.Until(m.ExpiresAt)
}

// Mapper manages the full set of active TCP port mappings.
type Mapper struct {
	mappings map[string]*Mapping
	mu       sync.RWMutex
}

// NewMapper creates an initialised Mapper.
func NewMapper() *Mapper {
	return &Mapper{
		mappings: make(map[string]*Mapping),
	}
}

// StartMapping binds 127.0.0.1:<localPort> and begins forwarding every
// accepted connection to remoteHost:remotePort.
func (m *Mapper) StartMapping(
	sessionID string,
	localPort int,
	remoteHost string,
	remotePort int,
	expiresAt time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mappings[sessionID]; exists {
		return fmt.Errorf("mapping already active for session %q", sessionID)
	}

	listenAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("binding %s: %w", listenAddr, err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), expiresAt)

	mapping := &Mapping{
		SessionID:        sessionID,
		LocalPort:        localPort,
		RemoteHost:       remoteHost,
		RemotePort:       remotePort,
		StartedAt:        time.Now(),
		ExpiresAt:        expiresAt,
		ConnectionStatus: "pending",
		listener:         listener,
		cancel:           cancel,
	}

	m.mappings[sessionID] = mapping

	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	go m.acceptLoop(ctx, mapping, remoteAddr)
	go m.watchExpiry(ctx, sessionID)
	go m.probeLoop(ctx, sessionID, remoteAddr)

	log.Info().
		Str("session_id", sessionID).
		Str("listen", listenAddr).
		Str("remote", remoteAddr).
		Time("expires_at", expiresAt).
		Msg("TCP mapping started")

	return nil
}

// StopMapping cancels the named mapping, closes the listener, and removes it
// from the internal registry. Stopping an unknown sessionID is a no-op.
func (m *Mapper) StopMapping(sessionID string) error {
	m.mu.Lock()
	mapping, exists := m.mappings[sessionID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("no active mapping for session %q", sessionID)
	}
	delete(m.mappings, sessionID)
	m.mu.Unlock()

	mapping.cancel()
	if err := mapping.listener.Close(); err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("error closing listener")
	}

	log.Info().
		Str("session_id", sessionID).
		Int64("bytes_forwarded", atomic.LoadInt64(&mapping.BytesFwd)).
		Str("connection_status", mapping.ConnectionStatus).
		Int("latency_ms", mapping.LatencyMS).
		Msg("TCP mapping stopped")

	return nil
}

// StopAll stops every active mapping. Used during graceful shutdown.
func (m *Mapper) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.mappings))
	for id := range m.mappings {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		if err := m.StopMapping(id); err != nil {
			log.Warn().Err(err).Str("session_id", id).Msg("stop-all error")
		}
	}
}

// GetMapping returns a snapshot for a single mapping.
func (m *Mapper) GetMapping(sessionID string) (Mapping, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping, ok := m.mappings[sessionID]
	if !ok {
		return Mapping{}, false
	}

	snap := *mapping
	snap.listener = nil
	snap.cancel = nil
	snap.BytesFwd = atomic.LoadInt64(&mapping.BytesFwd)
	return snap, true
}

// ListMappings returns a snapshot of all active mappings.
func (m *Mapper) ListMappings() []Mapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Mapping, 0, len(m.mappings))
	for _, mp := range m.mappings {
		snap := *mp
		snap.listener = nil
		snap.cancel = nil
		snap.BytesFwd = atomic.LoadInt64(&mp.BytesFwd)
		result = append(result, snap)
	}
	return result
}

func (m *Mapper) acceptLoop(ctx context.Context, mapping *Mapping, remoteAddr string) {
	for {
		conn, err := mapping.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				log.Error().Err(err).Str("session_id", mapping.SessionID).Msg("accept error")
			}
			return
		}
		go m.forwardTCP(ctx, mapping.SessionID, conn, remoteAddr, &mapping.BytesFwd)
	}
}

func (m *Mapper) forwardTCP(
	ctx context.Context,
	sessionID string,
	local net.Conn,
	remoteAddr string,
	counter *int64,
) {
	defer local.Close()

	var dialer net.Dialer
	remote, err := dialer.DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		m.updateProbeState(sessionID, "unreachable", 0, time.Now(), err.Error())
		log.Warn().Err(err).Str("remote", remoteAddr).Msg("dial remote failed")
		return
	}
	defer remote.Close()

	go func() {
		<-ctx.Done()
		_ = local.Close()
		_ = remote.Close()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		n, err := io.Copy(dst, src)
		if err != nil {
			log.Debug().Err(err).Msg("copy error (may be normal close)")
		}
		if counter != nil {
			atomic.AddInt64(counter, n)
		}
		_ = dst.Close()
	}

	go pipe(remote, local)
	go pipe(local, remote)

	wg.Wait()
}

func (m *Mapper) watchExpiry(ctx context.Context, sessionID string) {
	<-ctx.Done()

	m.mu.RLock()
	_, stillActive := m.mappings[sessionID]
	m.mu.RUnlock()

	if !stillActive {
		return
	}

	log.Info().Str("session_id", sessionID).Msg("session TTL expired - removing mapping")
	if err := m.StopMapping(sessionID); err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("expiry cleanup error")
	}
}

func (m *Mapper) probeLoop(ctx context.Context, sessionID, remoteAddr string) {
	m.runProbe(sessionID, remoteAddr)

	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runProbe(sessionID, remoteAddr)
		}
	}
}

func (m *Mapper) runProbe(sessionID, remoteAddr string) {
	started := time.Now()
	conn, err := net.DialTimeout("tcp", remoteAddr, probeTimeout)
	checkedAt := time.Now()
	if err != nil {
		m.updateProbeState(sessionID, "unreachable", 0, checkedAt, err.Error())
		return
	}
	_ = conn.Close()

	latency := time.Since(started)
	status := "reachable"
	if latency > degradedThreshold {
		status = "degraded"
	}

	m.updateProbeState(sessionID, status, int(latency.Milliseconds()), checkedAt, "")
}

func (m *Mapper) updateProbeState(
	sessionID string,
	status string,
	latencyMS int,
	checkedAt time.Time,
	lastError string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapping, ok := m.mappings[sessionID]
	if !ok {
		return
	}

	mapping.ConnectionStatus = status
	mapping.LatencyMS = latencyMS
	mapping.LastCheckedAt = checkedAt
	mapping.LastError = lastError
}
