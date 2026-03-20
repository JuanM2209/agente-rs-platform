package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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

// Handler holds dependencies for bridge HTTP handlers.
type Handler struct {
	db  *pgxpool.Pool
	hub *ws.AgentHub
}

// NewHandler constructs a bridge Handler.
func NewHandler(db *pgxpool.Pool, hub *ws.AgentHub) *Handler {
	return &Handler{db: db, hub: hub}
}

type startModbusRequest struct {
	SerialPort string `json:"serial_port"`
	BaudRate   int    `json:"baud_rate"`
	Parity     string `json:"parity"`
	StopBits   int    `json:"stop_bits"`
	DataBits   int    `json:"data_bits"`
	TCPPort    int    `json:"tcp_port"`
}

// StartModbusBridge handles POST /api/v1/devices/:deviceId/bridges/modbus-serial.
func (h *Handler) StartModbusBridge(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())

	var req startModbusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	req.SerialPort = strings.TrimSpace(req.SerialPort)
	req.Parity = normalizeParity(req.Parity)

	if req.SerialPort == "" {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "serial_port is required",
		})
		return
	}
	if req.TCPPort == 0 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "tcp_port is required",
		})
		return
	}
	if req.TCPPort < 1024 || req.TCPPort > 65535 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "tcp_port must be between 1024 and 65535",
		})
		return
	}
	if req.BaudRate < 0 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "baud_rate must be positive",
		})
		return
	}
	if req.DataBits != 0 && req.DataBits != 7 && req.DataBits != 8 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "data_bits must be 7 or 8",
		})
		return
	}
	if req.StopBits != 0 && req.StopBits != 1 && req.StopBits != 2 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "stop_bits must be 1 or 2",
		})
		return
	}
	if req.Parity != "N" && req.Parity != "E" && req.Parity != "O" {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "parity must be one of N, E or O",
		})
		return
	}

	// Defaults for optional serial params.
	if req.BaudRate == 0 {
		req.BaudRate = 9600
	}
	if req.DataBits == 0 {
		req.DataBits = 8
	}
	if req.StopBits == 0 {
		req.StopBits = 1
	}
	if req.Parity == "" {
		req.Parity = "N"
	}

	internalID, err := h.resolveInternalID(r.Context(), deviceID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceID).Msg("StartModbusBridge: resolve device")
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

	if active, lookupErr := h.activeBridgeForSerialPort(r.Context(), internalID, req.SerialPort); lookupErr != nil {
		log.Error().Err(lookupErr).Str("device_id", deviceID).Msg("StartModbusBridge: active bridge lookup failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to validate active bridges",
		})
		return
	} else if active {
		writeJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "a serial bridge is already active on this serial port",
		})
		return
	}

	bridgeID := uuid.New().String()
	now := time.Now().UTC()
	endpointID, endpointErr := h.upsertBridgeEndpoint(r.Context(), internalID, req.SerialPort, req.TCPPort)
	if endpointErr != nil {
		log.Error().Err(endpointErr).Str("bridge_id", bridgeID).Msg("StartModbusBridge: endpoint upsert failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to prepare bridge endpoint",
		})
		return
	}

	const insertQ = `
		INSERT INTO bridge_profiles
			(id, device_id, serial_port, baud_rate, parity, stop_bits, data_bits, tcp_port, status, last_started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	if _, dbErr := h.db.Exec(r.Context(), insertQ,
		bridgeID, internalID, req.SerialPort, req.BaudRate,
		req.Parity, req.StopBits, req.DataBits, req.TCPPort, "active", now, now,
	); dbErr != nil {
		log.Error().Err(dbErr).Str("bridge_id", bridgeID).Msg("StartModbusBridge: db insert")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to create bridge profile",
		})
		return
	}

	payloadData, _ := json.Marshal(ws.StartMBUSDPayload{
		BridgeID:   bridgeID,
		SerialPort: req.SerialPort,
		BaudRate:   req.BaudRate,
		Parity:     req.Parity,
		StopBits:   req.StopBits,
		DataBits:   req.DataBits,
		TCPPort:    req.TCPPort,
	})

	cmd := ws.AgentMessage{
		ID:        uuid.New().String(),
		Type:      ws.CmdStartMBUSD,
		Payload:   payloadData,
		Timestamp: now,
	}

	if err := h.hub.SendCommand(deviceID, cmd); err != nil {
		log.Error().Err(err).Str("bridge_id", bridgeID).Msg("StartModbusBridge: send command")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to send bridge command to device",
		})
		return
	}

	bridge := models.BridgeProfile{
		ID:         bridgeID,
		DeviceID:   internalID,
		EndpointID: endpointID,
		SerialPort: req.SerialPort,
		BaudRate:   req.BaudRate,
		Parity:     req.Parity,
		StopBits:   req.StopBits,
		DataBits:   req.DataBits,
		TCPPort:    req.TCPPort,
		Status:     "active",
		CreatedAt:  now,
	}

	writeJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    bridge,
	})
}

// StopBridge handles DELETE /api/v1/bridges/:bridgeId.
func (h *Handler) StopBridge(w http.ResponseWriter, r *http.Request) {
	bridgeID := chi.URLParam(r, "bridgeId")
	tenantID := middleware.GetTenantID(r.Context())

	bridge, deviceStringID, err := h.fetchBridge(r.Context(), bridgeID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "bridge not found",
			})
			return
		}
		log.Error().Err(err).Str("bridge_id", bridgeID).Msg("StopBridge: fetch bridge")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	const updateQ = `UPDATE bridge_profiles SET status = 'idle' WHERE id = $1`
	if _, dbErr := h.db.Exec(r.Context(), updateQ, bridgeID); dbErr != nil {
		log.Error().Err(dbErr).Str("bridge_id", bridgeID).Msg("StopBridge: db update")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to update bridge status",
		})
		return
	}

	if _, dbErr := h.db.Exec(r.Context(), `UPDATE endpoints SET enabled = false WHERE device_id = $1 AND port = $2`, bridge.DeviceID, bridge.TCPPort); dbErr != nil {
		log.Warn().Err(dbErr).Str("bridge_id", bridgeID).Msg("StopBridge: failed to disable bridge endpoint")
	}

	if deviceStringID != "" && h.hub.IsConnected(deviceStringID) {
		payloadData, _ := json.Marshal(ws.StopMBUSDPayload{BridgeID: bridgeID})
		cmd := ws.AgentMessage{
			ID:        uuid.New().String(),
			Type:      ws.CmdStopMBUSD,
			Payload:   payloadData,
			Timestamp: time.Now().UTC(),
		}
		if err := h.hub.SendCommand(deviceStringID, cmd); err != nil {
			log.Warn().Err(err).Str("bridge_id", bridgeID).Msg("StopBridge: agent notify failed")
		}
	}

	_ = bridge
	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "bridge stopped", "bridge_id": bridgeID},
	})
}

// resolveInternalID returns the device's UUID given its string device_id and tenant.
func (h *Handler) resolveInternalID(ctx context.Context, deviceID, tenantID string) (string, error) {
	var id string
	const q = `SELECT id FROM devices WHERE device_id = $1 AND tenant_id = $2 LIMIT 1`
	err := h.db.QueryRow(ctx, q, deviceID, tenantID).Scan(&id)
	return id, err
}

// fetchBridge retrieves a bridge profile and the device's string device_id.
func (h *Handler) fetchBridge(ctx context.Context, bridgeID, tenantID string) (*models.BridgeProfile, string, error) {
	const q = `
		SELECT bp.id, bp.device_id, bp.serial_port, bp.baud_rate, bp.parity,
		       bp.stop_bits, bp.data_bits, bp.tcp_port, bp.status, bp.created_at,
		       d.device_id AS device_string_id
		FROM bridge_profiles bp
		JOIN devices d ON d.id = bp.device_id
		WHERE bp.id = $1 AND d.tenant_id = $2
		LIMIT 1`

	bp := &models.BridgeProfile{}
	var deviceStringID string
	err := h.db.QueryRow(ctx, q, bridgeID, tenantID).Scan(
		&bp.ID, &bp.DeviceID, &bp.SerialPort, &bp.BaudRate, &bp.Parity,
		&bp.StopBits, &bp.DataBits, &bp.TCPPort, &bp.Status, &bp.CreatedAt,
		&deviceStringID,
	)
	return bp, deviceStringID, err
}

func normalizeParity(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "E", "EVEN":
		return "E"
	case "O", "ODD":
		return "O"
	default:
		return "N"
	}
}

func (h *Handler) activeBridgeForSerialPort(ctx context.Context, deviceID, serialPort string) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1
			FROM bridge_profiles
			WHERE device_id = $1 AND serial_port = $2 AND status = 'active'
		)`

	var exists bool
	if err := h.db.QueryRow(ctx, q, deviceID, serialPort).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (h *Handler) upsertBridgeEndpoint(ctx context.Context, deviceID, serialPort string, tcpPort int) (string, error) {
	label := "Serial Modbus Bridge"
	description := "Ephemeral MBUSD bridge for " + serialPort

	const q = `
		INSERT INTO endpoints
			(id, device_id, type, port, label, protocol, description, enabled, discovered_at, created_at)
		VALUES
			($1, $2, 'BRIDGE', $3, $4, 'mbusd', $5, true, NOW(), NOW())
		ON CONFLICT (device_id, port)
		DO UPDATE SET
			type = 'BRIDGE',
			label = EXCLUDED.label,
			protocol = EXCLUDED.protocol,
			description = EXCLUDED.description,
			enabled = true,
			discovered_at = NOW()
		RETURNING id`

	endpointID := uuid.New().String()
	if err := h.db.QueryRow(ctx, q, endpointID, deviceID, tcpPort, label, description).Scan(&endpointID); err != nil {
		return "", err
	}

	return endpointID, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
