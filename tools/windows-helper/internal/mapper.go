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

// Mapping represents a single active TCP port-forward from a local port to a
// remote Nucleus session endpoint.
//
// The unexported fields (listener, cancel) are intentionally excluded from any
// serialisation so that a future GUI can safely copy the public fields for
// display without touching the networking state.
type Mapping struct {
	SessionID  string    `json:"session_id"`
	LocalPort  int       `json:"local_port"`
	RemoteHost string    `json:"remote_host"`
	RemotePort int       `json:"remote_port"`
	StartedAt  time.Time `json:"started_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	BytesFwd   int64     `json:"bytes_forwarded"`

	listener net.Listener
	cancel   context.CancelFunc
}

// TTL returns how long until the mapping expires.  A negative duration means
// the session has already expired.
func (m *Mapping) TTL() time.Duration {
	return time.Until(m.ExpiresAt)
}

// Mapper manages the full set of active TCP port mappings.
// It is safe for concurrent use.
//
// Design note: Mapper is intentionally free of any GUI or Cobra dependencies
// so that it can be embedded directly in a future Windows systray application.
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
//
// The mapping is automatically stopped when the session TTL reaches zero.
// Calling StartMapping for an already-active sessionID returns an error.
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
		SessionID:  sessionID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		StartedAt:  time.Now(),
		ExpiresAt:  expiresAt,
		listener:   listener,
		cancel:     cancel,
	}

	m.mappings[sessionID] = mapping

	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	go m.acceptLoop(ctx, mapping, remoteAddr)
	go m.watchExpiry(ctx, sessionID)

	log.Info().
		Str("session_id", sessionID).
		Str("listen", listenAddr).
		Str("remote", remoteAddr).
		Time("expires_at", expiresAt).
		Msg("TCP mapping started")

	return nil
}

// StopMapping cancels the named mapping, closes the listener, and removes it
// from the internal registry.  Stopping an unknown sessionID is a no-op.
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
		Msg("TCP mapping stopped")

	return nil
}

// StopAll stops every active mapping.  Used during graceful shutdown.
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

// ListMappings returns a snapshot of all active mappings.
// Callers receive value copies; they cannot modify internal state.
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

// acceptLoop accepts inbound connections and spawns a forwardTCP goroutine for
// each one.  It exits when the context is done or the listener is closed.
func (m *Mapper) acceptLoop(ctx context.Context, mapping *Mapping, remoteAddr string) {
	for {
		conn, err := mapping.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// Expected — context expired or mapping was stopped.
			default:
				log.Error().Err(err).Str("session_id", mapping.SessionID).Msg("accept error")
			}
			return
		}
		go m.forwardTCP(ctx, conn, remoteAddr, &mapping.BytesFwd)
	}
}

// forwardTCP proxies data bidirectionally between the inbound local connection
// and a freshly-dialled remote connection.  Both connections are closed when
// either side finishes or the context is cancelled.
func (m *Mapper) forwardTCP(ctx context.Context, local net.Conn, remoteAddr string, counter *int64) {
	defer local.Close()

	var dialer net.Dialer
	remote, err := dialer.DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		log.Warn().Err(err).Str("remote", remoteAddr).Msg("dial remote failed")
		return
	}
	defer remote.Close()

	// Close both sides when the session context expires.
	go func() {
		<-ctx.Done()
		local.Close()
		remote.Close()
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
		// Half-close signals EOF to the other direction.
		dst.Close()
	}

	go pipe(remote, local)
	go pipe(local, remote)

	wg.Wait()
}

// watchExpiry blocks until the context deadline is reached and then stops the
// mapping.  This is the mechanism that enforces session TTL.
func (m *Mapper) watchExpiry(ctx context.Context, sessionID string) {
	<-ctx.Done()

	m.mu.RLock()
	_, stillActive := m.mappings[sessionID]
	m.mu.RUnlock()

	if !stillActive {
		// Already stopped by StopMapping — nothing to do.
		return
	}

	log.Info().Str("session_id", sessionID).Msg("session TTL expired — removing mapping")
	if err := m.StopMapping(sessionID); err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("expiry cleanup error")
	}
}
