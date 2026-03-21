package inventory

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// EndpointType categorises a discovered endpoint.
type EndpointType string

const (
	EndpointTypeWeb     EndpointType = "WEB"
	EndpointTypeProgram EndpointType = "PROGRAM"
	EndpointTypeBridge  EndpointType = "BRIDGE"
)

// Endpoint represents a single discovered network service or bridge interface.
type Endpoint struct {
	Port        int          `json:"port,omitempty"`
	Protocol    string       `json:"protocol,omitempty"`
	Type        EndpointType `json:"type"`
	Label       string       `json:"label"`
	SerialPort  string       `json:"serial_port,omitempty"`
}

// Inventory is the full snapshot of what is running on the device.
type Inventory struct {
	Endpoints    []Endpoint `json:"endpoints"`
	Capabilities []string  `json:"capabilities"`
	LocalIP      string    `json:"local_ip,omitempty"`
	ScannedAt    time.Time `json:"scanned_at"`
}

// wellKnownPorts maps port numbers to (type, label) pairs.
var wellKnownPorts = map[int][2]string{
	80:    {string(EndpointTypeWeb), "HTTP Web Interface"},
	443:   {string(EndpointTypeWeb), "HTTPS Web Interface"},
	1880:  {string(EndpointTypeWeb), "Node-RED Flow Editor"},
	9090:  {string(EndpointTypeWeb), "Device Web UI"},
	502:   {string(EndpointTypeProgram), "Modbus TCP"},
	22:    {string(EndpointTypeProgram), "SSH"},
	44818: {string(EndpointTypeProgram), "EtherNet/IP"},
	2202:  {string(EndpointTypeProgram), "Proprietary Protocol"},
	9999:  {string(EndpointTypeProgram), "Telnet/Debug"},
}

// Scanner periodically scans the device for open ports and serial interfaces.
type Scanner struct {
	interval time.Duration
	log      zerolog.Logger

	mu      sync.RWMutex
	current *Inventory

	stopCh chan struct{}
}

// NewScanner creates a Scanner with the given scan interval.
func NewScanner(interval time.Duration, log zerolog.Logger) *Scanner {
	return &Scanner{
		interval: interval,
		log:      log,
		stopCh:   make(chan struct{}),
	}
}

// Start performs an initial scan and then rescans on the configured interval.
// It blocks until Stop() is called.
func (s *Scanner) Start() {
	s.log.Info().Dur("interval", s.interval).Msg("inventory scanner started")
	s.scan()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scan()
		case <-s.stopCh:
			s.log.Info().Msg("inventory scanner stopped")
			return
		}
	}
}

// Stop signals the scanner goroutine to exit.
func (s *Scanner) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}

// Current returns the most recently collected inventory snapshot.
// Returns nil if no scan has completed yet.
func (s *Scanner) Current() *Inventory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// scan performs one full inventory sweep and caches the result.
func (s *Scanner) scan() {
	s.log.Debug().Msg("starting inventory scan")

	ports, err := openTCPPorts()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to read TCP port list")
		ports = nil
	}

	serialPorts, err := serialDevices()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to enumerate serial devices")
		serialPorts = nil
	}

	inv := buildInventory(ports, serialPorts)
	inv.LocalIP = LocalIP()
	inv.ScannedAt = time.Now().UTC()

	s.mu.Lock()
	s.current = inv
	s.mu.Unlock()

	s.log.Info().
		Int("endpoints", len(inv.Endpoints)).
		Strs("capabilities", inv.Capabilities).
		Msg("inventory scan complete")
}

// openTCPPorts returns the set of locally listening TCP port numbers by
// reading /proc/net/tcp and /proc/net/tcp6.
func openTCPPorts() ([]int, error) {
	portSet := make(map[int]struct{})

	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		ports, err := parseProcNetTCP(path)
		if err != nil {
			// Not all systems expose both files; skip missing ones silently.
			continue
		}
		for _, p := range ports {
			portSet[p] = struct{}{}
		}
	}

	result := make([]int, 0, len(portSet))
	for p := range portSet {
		result = append(result, p)
	}
	return result, nil
}

// parseProcNetTCP reads a /proc/net/tcp or /proc/net/tcp6 file and returns
// the port numbers of entries in the LISTEN state (state == 0x0A).
func parseProcNetTCP(path string) ([]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ports []int
	scanner := bufio.NewScanner(f)

	// Skip header line.
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Field index 3 is the state (hex).
		state := fields[3]
		if state != "0A" { // 0A = TCP_LISTEN
			continue
		}

		// Field index 1 is local_address in host:port hex form.
		localAddr := fields[1]
		colonIdx := strings.LastIndex(localAddr, ":")
		if colonIdx < 0 {
			continue
		}
		portHex := localAddr[colonIdx+1:]
		portBytes, err := hex.DecodeString(portHex)
		if err != nil || len(portBytes) < 2 {
			continue
		}
		port := int(portBytes[0])<<8 | int(portBytes[1])
		ports = append(ports, port)
	}

	return ports, scanner.Err()
}

// serialDevices returns paths to available serial devices by globbing /dev.
func serialDevices() ([]string, error) {
	var devices []string

	patterns := []string{"/dev/ttyS*", "/dev/ttyUSB*", "/dev/ttyACM*", "/dev/ttymxc*"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		devices = append(devices, matches...)
	}

	return devices, nil
}

// buildInventory converts raw scan results into a structured Inventory.
func buildInventory(ports []int, serialPorts []string) *Inventory {
	var endpoints []Endpoint
	capSet := make(map[string]struct{})

	for _, port := range ports {
		info, known := wellKnownPorts[port]
		if !known {
			continue
		}
		endpoints = append(endpoints, Endpoint{
			Port:     port,
			Protocol: "tcp",
			Type:     EndpointType(info[0]),
			Label:    info[1],
		})
	}

	for _, sp := range serialPorts {
		endpoints = append(endpoints, Endpoint{
			Type:       EndpointTypeBridge,
			Label:      fmt.Sprintf("Serial Port %s", sp),
			SerialPort: sp,
		})
		capSet["modbus_rtu"] = struct{}{}
		capSet["serial_bridge"] = struct{}{}
	}

	if hasPort(ports, 502) {
		capSet["modbus_tcp"] = struct{}{}
	}
	if hasPort(ports, 22) {
		capSet["ssh"] = struct{}{}
	}

	capabilities := make([]string, 0, len(capSet))
	for cap := range capSet {
		capabilities = append(capabilities, cap)
	}

	return &Inventory{
		Endpoints:    endpoints,
		Capabilities: capabilities,
	}
}

func hasPort(ports []int, target int) bool {
	for _, p := range ports {
		if p == target {
			return true
		}
	}
	return false
}

// LocalIP returns the primary non-loopback IPv4 address, used for diagnostics.
func LocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return ""
}

// portFromHex converts a hexadecimal string to an integer port number.
// It is used by unit tests to decode /proc/net/tcp port fields directly.
func portFromHex(s string) (int, error) {
	n, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
