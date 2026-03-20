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
		SELECT id, tenant_id, site_id, device_id, display_name, status,
		       last_seen, firmware_version, ip_address, created_at
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

	internalID, err := h.resolveInternalID(r.Context(), deviceID, tenantID)
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

	const q = `
		SELECT id, device_id, type, port, label, protocol, enabled
		FROM endpoints
		WHERE device_id = $1
		ORDER BY port ASC`

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

	endpoints := make([]models.Endpoint, 0)
	for rows.Next() {
		var ep models.Endpoint
		if scanErr := rows.Scan(
			&ep.ID,
			&ep.DeviceID,
			&ep.Type,
			&ep.Port,
			&ep.Label,
			&ep.Protocol,
			&ep.Enabled,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("GetInventory: row scan error")
			continue
		}
		endpoints = append(endpoints, ep)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("GetInventory: rows iteration error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    endpoints,
		Meta: &models.MetaInfo{
			Total: len(endpoints),
			Page:  1,
			Limit: len(endpoints),
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
		SELECT id, tenant_id, site_id, device_id, display_name, status,
		       last_seen, firmware_version, ip_address, created_at
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
