package mbusd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

const (
	mbusdBinary     = "mbusd"
	startupWait     = 500 * time.Millisecond
	defaultBaudRate = 9600
	defaultParity   = "none"
)

// MBUSDProcess represents a running mbusd bridge subprocess.
type MBUSDProcess struct {
	SerialPort string
	BaudRate   int
	TCPPort    int
	Parity     string
	BridgeID   string
	StartedAt  time.Time

	cmd *exec.Cmd
	log zerolog.Logger
}

// Config carries the parameters needed to start an mbusd process.
type Config struct {
	BridgeID   string
	SerialPort string
	BaudRate   int
	TCPPort    int
	Parity     string
}

// New creates an MBUSDProcess from the supplied config. It does not start the
// process; call Start() for that.
func New(cfg Config, log zerolog.Logger) *MBUSDProcess {
	baudRate := cfg.BaudRate
	if baudRate <= 0 {
		baudRate = defaultBaudRate
	}
	parity := cfg.Parity
	if parity == "" {
		parity = defaultParity
	}

	return &MBUSDProcess{
		SerialPort: cfg.SerialPort,
		BaudRate:   baudRate,
		TCPPort:    cfg.TCPPort,
		Parity:     parity,
		BridgeID:   cfg.BridgeID,
		log:        log,
	}
}

// Validate checks that prerequisites are met before starting:
//   - mbusd binary is available on PATH
//   - serial port device exists
//   - TCP port is not already in use
func (m *MBUSDProcess) Validate() error {
	if _, err := exec.LookPath(mbusdBinary); err != nil {
		return fmt.Errorf("mbusd binary not found on PATH: %w", err)
	}

	if _, err := os.Stat(m.SerialPort); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("serial port %s does not exist", m.SerialPort)
		}
		return fmt.Errorf("cannot access serial port %s: %w", m.SerialPort, err)
	}

	if err := checkPortFree(m.TCPPort); err != nil {
		return fmt.Errorf("TCP port %d is not available: %w", m.TCPPort, err)
	}

	return nil
}

// Start spawns the mbusd process. Returns an error if validation fails or the
// process cannot be started. A short delay is applied to detect immediate exits.
func (m *MBUSDProcess) Start() error {
	if m.IsRunning() {
		return errors.New("mbusd process is already running")
	}

	if err := m.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	args := m.buildArgs()
	m.log.Info().
		Str("bridge_id", m.BridgeID).
		Str("serial_port", m.SerialPort).
		Int("baud_rate", m.BaudRate).
		Int("tcp_port", m.TCPPort).
		Str("parity", m.Parity).
		Strs("args", args).
		Msg("starting mbusd process")

	cmd := exec.Command(mbusdBinary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mbusd: %w", err)
	}

	m.cmd = cmd
	m.StartedAt = time.Now().UTC()

	// Give the process a moment and check it hasn't already exited.
	time.Sleep(startupWait)
	if !m.IsRunning() {
		return errors.New("mbusd process exited immediately after start")
	}

	m.log.Info().
		Str("bridge_id", m.BridgeID).
		Int("pid", cmd.Process.Pid).
		Msg("mbusd process started successfully")

	return nil
}

// Stop sends SIGTERM to the mbusd process and waits for it to exit.
// If the process is not running, Stop is a no-op.
func (m *MBUSDProcess) Stop() error {
	if !m.IsRunning() {
		return nil
	}

	m.log.Info().
		Str("bridge_id", m.BridgeID).
		Int("pid", m.cmd.Process.Pid).
		Msg("stopping mbusd process")

	// Send SIGTERM to the entire process group.
	if err := syscall.Kill(-m.cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// Fall back to direct kill.
		if killErr := m.cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill mbusd process: %w", killErr)
		}
	}

	// Wait with timeout — we don't want to block forever.
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && !isExitError(err) {
			m.log.Warn().Err(err).Str("bridge_id", m.BridgeID).Msg("mbusd exited with error")
		}
	case <-time.After(5 * time.Second):
		m.log.Warn().Str("bridge_id", m.BridgeID).Msg("mbusd did not stop gracefully, force killing")
		_ = m.cmd.Process.Kill()
	}

	m.log.Info().Str("bridge_id", m.BridgeID).Msg("mbusd process stopped")
	m.cmd = nil
	return nil
}

// IsRunning reports whether the mbusd process is currently alive.
func (m *MBUSDProcess) IsRunning() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	// Signal 0 checks process existence without sending a real signal.
	err := m.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// PID returns the OS process ID, or -1 if not running.
func (m *MBUSDProcess) PID() int {
	if m.cmd == nil || m.cmd.Process == nil {
		return -1
	}
	return m.cmd.Process.Pid
}

// buildArgs constructs the mbusd command-line arguments.
//
// mbusd flags (common implementation):
//
//	-d          - run in foreground (no daemon)
//	-L /dev/ttyXX - serial device
//	-s <baud>   - baud rate
//	-P <port>   - TCP port to listen on
//	-p <parity> - parity (none/even/odd)
func (m *MBUSDProcess) buildArgs() []string {
	args := []string{
		"-d",
		"-L", m.SerialPort,
		"-s", strconv.Itoa(m.BaudRate),
		"-P", strconv.Itoa(m.TCPPort),
	}

	switch m.Parity {
	case "even":
		args = append(args, "-p", "even")
	case "odd":
		args = append(args, "-p", "odd")
	default:
		args = append(args, "-p", "none")
	}

	return args
}

// checkPortFree returns an error if the TCP port is already in use.
func checkPortFree(port int) error {
	addr := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	_ = l.Close()
	return nil
}

// isExitError returns true for normal process exit errors (which are expected
// after SIGTERM).
func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}
