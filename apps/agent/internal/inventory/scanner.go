package inventory

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	Port       int          `json:"port,omitempty"`
	Protocol   string       `json:"protocol,omitempty"`
	Type       EndpointType `json:"type"`
	Label      string       `json:"label"`
	SerialPort string       `json:"serial_port,omitempty"`
}

// Inventory is the full snapshot of what is running on the device.
type Inventory struct {
	Endpoints    []Endpoint `json:"endpoints"`
	Capabilities []string   `json:"capabilities"`
	LocalIP      string     `json:"local_ip,omitempty"`
	ScannedAt    time.Time  `json:"scanned_at"`
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
	interval              time.Duration
	controlPlaneURL       string
	localIPOverride       string
	preferredLANInterface string
	log                   zerolog.Logger

	mu      sync.RWMutex
	current *Inventory

	stopCh chan struct{}
}

// NewScanner creates a Scanner with the given scan interval.
func NewScanner(interval time.Duration, controlPlaneURL, localIPOverride, preferredLANInterface string, log zerolog.Logger) *Scanner {
	return &Scanner{
		interval:              interval,
		controlPlaneURL:       strings.TrimSpace(controlPlaneURL),
		localIPOverride:       strings.TrimSpace(localIPOverride),
		preferredLANInterface: strings.TrimSpace(preferredLANInterface),
		log:                   log,
		stopCh:                make(chan struct{}),
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
	inv.LocalIP = LocalIP(s.controlPlaneURL, s.localIPOverride, s.preferredLANInterface)
	inv.ScannedAt = time.Now().UTC()

	s.mu.Lock()
	previous := s.current
	s.current = inv
	s.mu.Unlock()

	logger := s.log.Debug()
	if inventoryChanged(previous, inv) {
		logger = s.log.Info()
	}

	logger.
		Int("endpoints", len(inv.Endpoints)).
		Strs("capabilities", inv.Capabilities).
		Str("local_ip", inv.LocalIP).
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
	sort.Ints(result)
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
	sort.Strings(capabilities)

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

func inventoryChanged(previous, current *Inventory) bool {
	if previous == nil || current == nil {
		return true
	}
	if previous.LocalIP != current.LocalIP {
		return true
	}
	if len(previous.Endpoints) != len(current.Endpoints) || len(previous.Capabilities) != len(current.Capabilities) {
		return true
	}
	for i := range previous.Capabilities {
		if previous.Capabilities[i] != current.Capabilities[i] {
			return true
		}
	}
	for i := range previous.Endpoints {
		if previous.Endpoints[i] != current.Endpoints[i] {
			return true
		}
	}
	return false
}

// LocalIP returns the best LAN-facing IPv4 address for the Nucleus. It prefers:
// 1. explicit override, 2. an interface chosen by routing to the control plane,
// 3. the default route interface, 4. scored interface enumeration.
func LocalIP(controlPlaneURL, override, preferredInterface string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}

	if preferred := strings.TrimSpace(preferredInterface); preferred != "" {
		if ip := interfaceIPv4(preferred); ip != "" {
			return ip
		}
	}

	if ip := localIPViaDial(controlPlaneURL); ip != "" {
		return ip
	}

	if defaultIface := defaultRouteInterface(); defaultIface != "" {
		if ip := interfaceIPv4(defaultIface); ip != "" {
			return ip
		}
	}

	type candidate struct {
		name  string
		ip    string
		score int
	}

	var best candidate
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
			ipv4 := ip.To4()
			if ipv4 == nil {
				continue
			}

			score := scoreInterfaceAddress(iface.Name, ipv4, preferredInterface)
			if score > best.score {
				best = candidate{
					name:  iface.Name,
					ip:    ipv4.String(),
					score: score,
				}
			}
		}
	}
	return best.ip
}

func localIPViaDial(controlPlaneURL string) string {
	targetHost := controlPlaneHost(controlPlaneURL)
	if targetHost == "" {
		targetHost = "1.1.1.1"
	}

	conn, err := net.DialTimeout("udp", net.JoinHostPort(targetHost, "443"), 2*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil {
		if ipv4 := addr.IP.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ""
}

func controlPlaneHost(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func defaultRouteInterface() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		if fields[1] == "00000000" && fields[3] != "0000" {
			return fields[0]
		}
	}
	return ""
}

func interfaceIPv4(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil {
			continue
		}
		if ipv4 := ip.To4(); ipv4 != nil && !ipv4.IsLoopback() {
			return ipv4.String()
		}
	}
	return ""
}

func scoreInterfaceAddress(name string, ip net.IP, preferredInterface string) int {
	score := 0
	ipv4 := ip.To4()
	if ipv4 == nil {
		return score
	}

	if strings.EqualFold(name, preferredInterface) {
		score += 100
	}
	if isPreferredLANInterfaceName(name) {
		score += 35
	}
	if isVirtualInterfaceName(name) {
		score -= 80
	}
	if isRFC1918(ipv4) {
		score += 60
	}
	if ipv4.IsGlobalUnicast() {
		score += 10
	}
	if ipv4[0] == 169 && ipv4[1] == 254 {
		score -= 40
	}
	return score
}

func isPreferredLANInterfaceName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"eth", "en", "wlan", "wl", "usb", "ppp"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func isVirtualInterfaceName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"docker", "br-", "veth", "cni", "flannel", "tailscale", "zt", "tun"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func isRFC1918(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false
	}
	switch {
	case ipv4[0] == 10:
		return true
	case ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31:
		return true
	case ipv4[0] == 192 && ipv4[1] == 168:
		return true
	default:
		return false
	}
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
