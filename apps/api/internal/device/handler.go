package device

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nucleus-portal/api/internal/middleware"
	"github.com/nucleus-portal/api/internal/models"
	"github.com/nucleus-portal/api/internal/ws"
	"github.com/rs/zerolog/log"
)

// Handler holds dependencies for device HTTP handlers.
type Handler struct {
	db  *pgxpool.Pool
	hub *ws.AgentHub
}

const (
	defaultModbusSerialPort = "/dev/ttymxc5"
	inventoryStaleAfter     = 10 * time.Minute
)

type inventoryEndpointGroups struct {
	Web     []models.Endpoint `json:"web"`
	Program []models.Endpoint `json:"program"`
	Bridge  []models.Endpoint `json:"bridge"`
}

type inventoryCapabilities struct {
	HasSerial          bool     `json:"has_serial"`
	SerialPorts        []string `json:"serial_ports"`
	ModbusSerialPort   string   `json:"modbus_serial_port,omitempty"`
	ActivationWarning  string   `json:"activation_warning,omitempty"`
	BundledBridgeBinary string  `json:"bundled_bridge_binary,omitempty"`
}

type inventoryFreshness struct {
	LastScan time.Time `json:"last_scan"`
	IsStale  bool      `json:"is_stale"`
}

type inventoryResponse struct {
	Device       *models.Device        `json:"device"`
	Endpoints    inventoryEndpointGroups `json:"endpoints"`
	Capabilities inventoryCapabilities `json:"capabilities"`
	Freshness    inventoryFreshness    `json:"freshness"`
}

// NewHandler constructs a device Handler.
func NewHandler(db *pgxpool.Pool, hub *ws.AgentHub) *Handler {
	return &Handler{db: db, hub: hub}
}

// GetDevice handles GET /api/v1/devices/:deviceId.
// Fetches a device by its string device_id, scoped to the caller's tenant.
func (h *Handler) GetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())

	const q = `
		SELECT id, tenant_id, COALESCE(site_id::text, ''), device_id, display_name, status,
		       last_seen, COALESCE(firmware_version, ''), COALESCE(ip_address::text, ''), COALESCE(hardware_model, ''),
		       inventory_updated_at, created_at
		FROM devices
		WHERE device_id = $1 AND tenant_id = $2
		LIMIT 1`

	device := &models.Device{}
	err := h.db.QueryRow(r.Context(), q, deviceID, tenantID).Scan(
		&device.ID,
		&device.TenantID,
		&device.SiteID,
		&device.DeviceID,
		&device.DisplayName,
		&device.Status,
		&device.LastSeen,
		&device.FirmwareVersion,
		&device.IPAddress,
		&device.HardwareModel,
		&device.InventoryUpdatedAt,
		&device.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceID).Msg("GetDevice: db error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	// Reflect live connectivity status from the hub when the DB says online.
	if !h.hub.IsConnected(deviceID) && device.Status == models.DeviceStatusOnline {
		device.Status = models.DeviceStatusOffline
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    device,
	})
}

// GetInventory handles GET /api/v1/devices/:deviceId/inventory.
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())

	device, err := h.fetchDevice(r.Context(), deviceID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceID).Msg("GetInventory: resolve device")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	internalID := device.ID

	const q = `
		SELECT id, device_id, type, port, label, protocol, COALESCE(description, ''), enabled, discovered_at
		FROM endpoints
		WHERE device_id = $1 AND enabled = true
		ORDER BY type ASC, port ASC`

	rows, err := h.db.Query(r.Context(), q, internalID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("GetInventory: db error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}
	defer rows.Close()

	groups := inventoryEndpointGroups{
		Web:     make([]models.Endpoint, 0),
		Program: make([]models.Endpoint, 0),
		Bridge:  make([]models.Endpoint, 0),
	}
	for rows.Next() {
		var ep models.Endpoint
		if scanErr := rows.Scan(
			&ep.ID,
			&ep.DeviceID,
			&ep.Type,
			&ep.Port,
			&ep.Label,
			&ep.Protocol,
			&ep.Description,
			&ep.Enabled,
			&ep.DiscoveredAt,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("GetInventory: row scan error")
			continue
		}
		switch ep.Type {
		case models.EndpointTypeWeb:
			groups.Web = append(groups.Web, ep)
		case models.EndpointTypeBridge:
			groups.Bridge = append(groups.Bridge, ep)
		default:
			groups.Program = append(groups.Program, ep)
		}
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("GetInventory: rows iteration error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	serialPorts, bridgeErr := h.fetchSerialPorts(r.Context(), internalID)
	if bridgeErr != nil {
		log.Warn().Err(bridgeErr).Str("device_id", deviceID).Msg("GetInventory: serial port lookup failed")
	}
	if len(serialPorts) == 0 {
		serialPorts = []string{defaultModbusSerialPort}
	}

	lastScan := time.Now().UTC()
	if device.InventoryUpdatedAt != nil && !device.InventoryUpdatedAt.IsZero() {
		lastScan = device.InventoryUpdatedAt.UTC()
	}

	if !h.hub.IsConnected(deviceID) && device.Status == models.DeviceStatusOnline {
		device.Status = models.DeviceStatusOffline
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: inventoryResponse{
			Device:    device,
			Endpoints: groups,
			Capabilities: inventoryCapabilities{
				HasSerial:           len(serialPorts) > 0,
				SerialPorts:         serialPorts,
				ModbusSerialPort:    defaultModbusSerialPort,
				ActivationWarning:   "Activating MBUSD on /dev/ttymxc5 temporarily interrupts Node-RED Modbus serial communication while the serial bridge is active.",
				BundledBridgeBinary: "mbusd (bundled for ARMv7 Nucleus devices)",
			},
			Freshness: inventoryFreshness{
				LastScan: lastScan,
				IsStale:  time.Since(lastScan) > inventoryStaleAfter,
			},
		},
	})
}

// ScanDevice handles POST /api/v1/devices/:deviceId/scan.
func (h *Handler) ScanDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())

	if _, err := h.resolveInternalID(r.Context(), deviceID, tenantID); err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceID).Msg("ScanDevice: resolve device")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	if !h.hub.IsConnected(deviceID) {
		writeJSON(w, http.StatusServiceUnavailable, models.APIResponse{
			Success: false,
			Error:   "device is not connected",
		})
		return
	}

	cmd := ws.AgentMessage{
		ID:        uuid.New().String(),
		Type:      ws.CmdSyncInventory,
		Timestamp: time.Now().UTC(),
	}

	if err := h.hub.SendCommand(deviceID, cmd); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("ScanDevice: send command failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to send scan command to device",
		})
		return
	}

	writeJSON(w, http.StatusAccepted, models.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "scan initiated", "command_id": cmd.ID},
	})
}

// ListDevices handles GET /api/v1/admin/devices.
func (h *Handler) ListDevices(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	const q = `
		SELECT id, tenant_id, COALESCE(site_id::text, ''), device_id, display_name, status,
		       last_seen, COALESCE(firmware_version, ''), COALESCE(ip_address::text, ''), created_at
		FROM devices
		WHERE tenant_id = $1
		ORDER BY display_name ASC`

	rows, err := h.db.Query(r.Context(), q, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("ListDevices: db error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		var d models.Device
		if scanErr := rows.Scan(
			&d.ID,
			&d.TenantID,
			&d.SiteID,
			&d.DeviceID,
			&d.DisplayName,
			&d.Status,
			&d.LastSeen,
			&d.FirmwareVersion,
			&d.IPAddress,
			&d.CreatedAt,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("ListDevices: scan error")
			continue
		}
		devices = append(devices, d)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("ListDevices: rows error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    devices,
		Meta: &models.MetaInfo{
			Total: len(devices),
			Page:  1,
			Limit: len(devices),
		},
	})
}

// resolveInternalID looks up the internal UUID for a device_id string,
// confirming the device belongs to the given tenant.
func (h *Handler) resolveInternalID(ctx context.Context, deviceID, tenantID string) (string, error) {
	var id string
	const q = `SELECT id FROM devices WHERE device_id = $1 AND tenant_id = $2 LIMIT 1`
	err := h.db.QueryRow(ctx, q, deviceID, tenantID).Scan(&id)
	return id, err
}

func (h *Handler) fetchDevice(ctx context.Context, deviceID, tenantID string) (*models.Device, error) {
	const q = `
		SELECT d.id, d.tenant_id, COALESCE(d.site_id::text, ''), d.device_id, d.display_name, d.status,
		       d.last_seen, COALESCE(d.firmware_version, ''), COALESCE(d.ip_address::text, ''), COALESCE(d.hardware_model, ''),
		       d.inventory_updated_at, d.created_at,
		       COALESCE(s.id::text, ''), COALESCE(s.tenant_id::text, ''),
		       COALESCE(s.name, ''), COALESCE(s.location, ''), COALESCE(s.timezone, '')
		FROM devices d
		LEFT JOIN sites s ON s.id = d.site_id
		WHERE d.device_id = $1 AND d.tenant_id = $2
		LIMIT 1`

	device := &models.Device{}
	var siteID string
	var siteTenantID string
	var siteName string
	var siteLocation string
	var siteTimezone string

	if err := h.db.QueryRow(ctx, q, deviceID, tenantID).Scan(
		&device.ID,
		&device.TenantID,
		&device.SiteID,
		&device.DeviceID,
		&device.DisplayName,
		&device.Status,
		&device.LastSeen,
		&device.FirmwareVersion,
		&device.IPAddress,
		&device.HardwareModel,
		&device.InventoryUpdatedAt,
		&device.CreatedAt,
		&siteID,
		&siteTenantID,
		&siteName,
		&siteLocation,
		&siteTimezone,
	); err != nil {
		return nil, err
	}

	if siteID != "" {
		device.Site = &models.Site{
			ID:       siteID,
			TenantID: siteTenantID,
			Name:     siteName,
			Location: siteLocation,
			Timezone: siteTimezone,
		}
	}

	return device, nil
}

func (h *Handler) fetchSerialPorts(ctx context.Context, deviceInternalID string) ([]string, error) {
	const q = `
		SELECT serial_port
		FROM bridge_profiles
		WHERE device_id = $1
		ORDER BY created_at DESC`

	rows, err := h.db.Query(ctx, q, deviceInternalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := make([]string, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		var serialPort string
		if scanErr := rows.Scan(&serialPort); scanErr != nil {
			continue
		}
		if serialPort == "" {
			continue
		}
		if _, ok := seen[serialPort]; ok {
			continue
		}
		seen[serialPort] = struct{}{}
		ports = append(ports, serialPort)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return ports, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
