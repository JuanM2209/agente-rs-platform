package session

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

// Handler holds dependencies for session HTTP handlers.
type Handler struct {
	db  *pgxpool.Pool
	hub *ws.AgentHub
}

// NewHandler constructs a session Handler.
func NewHandler(db *pgxpool.Pool, hub *ws.AgentHub) *Handler {
	return &Handler{db: db, hub: hub}
}

type createSessionRequest struct {
	EndpointID         string `json:"endpoint_id"`
	DeliveryMode       string `json:"delivery_mode"`
	TTLSeconds         int    `json:"ttl_seconds"`
	IdleTimeoutSeconds int    `json:"idle_timeout_seconds"`
}

// CreateSession handles POST /api/v1/devices/:deviceId/sessions.
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	if req.EndpointID == "" {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "endpoint_id is required",
		})
		return
	}

	deliveryMode := models.DeliveryMode(req.DeliveryMode)
	if deliveryMode != models.DeliveryModeWeb && deliveryMode != models.DeliveryModeExport {
		deliveryMode = models.DeliveryModeWeb
	}

	if req.TTLSeconds <= 0 {
		req.TTLSeconds = 3600 // 1 hour default
	}
	if req.IdleTimeoutSeconds <= 0 {
		req.IdleTimeoutSeconds = 900 // 15 min default
	}

	// Verify device and endpoint belong to tenant.
	device, endpoint, err := h.resolveDeviceAndEndpoint(r.Context(), deviceID, req.EndpointID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device or endpoint not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceID).Msg("CreateSession: resolve device/endpoint")
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

	sessionID := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(req.TTLSeconds) * time.Second)

	// Persist session before sending command so it exists even if the ack is lost.
	const insertQ = `
		INSERT INTO sessions
			(id, device_id, endpoint_id, user_id, tenant_id, status, local_port,
			 delivery_mode, ttl_seconds, idle_timeout_seconds, started_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = h.db.Exec(r.Context(), insertQ,
		sessionID,
		device.ID,
		endpoint.ID,
		userID,
		tenantID,
		models.SessionStatusActive,
		endpoint.Port,
		string(deliveryMode),
		req.TTLSeconds,
		req.IdleTimeoutSeconds,
		now,
		expiresAt,
	)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("CreateSession: db insert failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to create session",
		})
		return
	}

	payloadData, _ := json.Marshal(ws.StartSessionPayload{
		SessionID: sessionID,
		Port:      endpoint.Port,
		Protocol:  endpoint.Protocol,
		TTL:       req.TTLSeconds,
	})

	cmd := ws.AgentMessage{
		ID:        uuid.New().String(),
		Type:      ws.CmdStartSession,
		Payload:   payloadData,
		Timestamp: now,
	}

	if err := h.hub.SendCommand(deviceID, cmd); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("CreateSession: send command failed")
		// Session is already in DB; agent will reconcile on reconnect.
	}

	session := models.Session{
		ID:                 sessionID,
		DeviceID:           device.ID,
		EndpointID:         endpoint.ID,
		UserID:             userID,
		TenantID:           tenantID,
		Status:             models.SessionStatusActive,
		LocalPort:          endpoint.Port,
		DeliveryMode:       deliveryMode,
		TTLSeconds:         req.TTLSeconds,
		IdleTimeoutSeconds: req.IdleTimeoutSeconds,
		StartedAt:          now,
		ExpiresAt:          &expiresAt,
	}

	writeJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    session,
	})
}

// StopSession handles DELETE /api/v1/sessions/:sessionId.
func (h *Handler) StopSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	// Fetch session to obtain device_id and verify ownership.
	session, err := h.fetchSession(r.Context(), sessionID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "session not found",
			})
			return
		}
		log.Error().Err(err).Str("session_id", sessionID).Msg("StopSession: fetch session")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	if session.UserID != userID {
		writeJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "forbidden",
		})
		return
	}

	if session.Status != models.SessionStatusActive {
		writeJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "session is not active",
		})
		return
	}

	now := time.Now().UTC()
	const updateQ = `
		UPDATE sessions
		SET status = $1, stopped_at = $2, stop_reason = $3
		WHERE id = $4`

	if _, dbErr := h.db.Exec(r.Context(), updateQ,
		models.SessionStatusStopped, now, "user_requested", sessionID,
	); dbErr != nil {
		log.Error().Err(dbErr).Str("session_id", sessionID).Msg("StopSession: db update failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to stop session",
		})
		return
	}

	// Resolve device string ID for WS command.
	var deviceStringID string
	_ = h.db.QueryRow(r.Context(), `SELECT device_id FROM devices WHERE id = $1`, session.DeviceID).Scan(&deviceStringID)

	if deviceStringID != "" && h.hub.IsConnected(deviceStringID) {
		payloadData, _ := json.Marshal(ws.StopSessionPayload{SessionID: sessionID})
		cmd := ws.AgentMessage{
			ID:        uuid.New().String(),
			Type:      ws.CmdStopSession,
			Payload:   payloadData,
			Timestamp: now,
		}
		if err := h.hub.SendCommand(deviceStringID, cmd); err != nil {
			log.Warn().Err(err).Str("session_id", sessionID).Msg("StopSession: agent notify failed")
		}
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "session stopped", "session_id": sessionID},
	})
}

// ListActiveSessions handles GET /api/v1/me/active-sessions.
func (h *Handler) ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	tenantID := middleware.GetTenantID(r.Context())

	const q = `
		SELECT id, device_id, endpoint_id, user_id, tenant_id, status, local_port,
		       delivery_mode, ttl_seconds, idle_timeout_seconds, started_at, expires_at,
		       stopped_at, stop_reason, audit_data
		FROM sessions
		WHERE user_id = $1 AND tenant_id = $2 AND status = 'active'
		ORDER BY started_at DESC`

	rows, err := h.db.Query(r.Context(), q, userID, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("ListActiveSessions: db error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}
	defer rows.Close()

	sessions := make([]models.Session, 0)
	for rows.Next() {
		var s models.Session
		if scanErr := rows.Scan(
			&s.ID, &s.DeviceID, &s.EndpointID, &s.UserID, &s.TenantID,
			&s.Status, &s.LocalPort, &s.DeliveryMode, &s.TTLSeconds,
			&s.IdleTimeoutSeconds, &s.StartedAt, &s.ExpiresAt,
			&s.StoppedAt, &s.StopReason, &s.AuditData,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("ListActiveSessions: scan error")
			continue
		}
		sessions = append(sessions, s)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("ListActiveSessions: rows error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    sessions,
		Meta:    &models.MetaInfo{Total: len(sessions), Page: 1, Limit: len(sessions)},
	})
}

// ListExportHistory handles GET /api/v1/me/export-history.
func (h *Handler) ListExportHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	tenantID := middleware.GetTenantID(r.Context())

	const q = `
		SELECT id, session_id, user_id, device_id, endpoint_id, tenant_id, site_id,
		       started_at, stopped_at, stop_reason, local_bind_port, delivery_mode, metadata
		FROM export_history
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY started_at DESC
		LIMIT 100`

	rows, err := h.db.Query(r.Context(), q, userID, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("ListExportHistory: db error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}
	defer rows.Close()

	history := make([]models.ExportHistory, 0)
	for rows.Next() {
		var eh models.ExportHistory
		if scanErr := rows.Scan(
			&eh.ID, &eh.SessionID, &eh.UserID, &eh.DeviceID, &eh.EndpointID,
			&eh.TenantID, &eh.SiteID, &eh.StartedAt, &eh.StoppedAt,
			&eh.StopReason, &eh.LocalBindPort, &eh.DeliveryMode, &eh.Metadata,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("ListExportHistory: scan error")
			continue
		}
		history = append(history, eh)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("ListExportHistory: rows error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    history,
		Meta:    &models.MetaInfo{Total: len(history), Page: 1, Limit: 100},
	})
}

// resolveDeviceAndEndpoint fetches the device and endpoint, validating tenant ownership.
func (h *Handler) resolveDeviceAndEndpoint(
	ctx context.Context,
	deviceStringID, endpointID, tenantID string,
) (*models.Device, *models.Endpoint, error) {
	device := &models.Device{}
	const dq = `
		SELECT id, tenant_id, site_id, device_id, display_name, status,
		       last_seen, firmware_version, ip_address, created_at
		FROM devices
		WHERE device_id = $1 AND tenant_id = $2
		LIMIT 1`

	err := h.db.QueryRow(ctx, dq, deviceStringID, tenantID).Scan(
		&device.ID, &device.TenantID, &device.SiteID, &device.DeviceID,
		&device.DisplayName, &device.Status, &device.LastSeen,
		&device.FirmwareVersion, &device.IPAddress, &device.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	endpoint := &models.Endpoint{}
	const eq = `
		SELECT id, device_id, type, port, label, protocol, enabled
		FROM endpoints
		WHERE id = $1 AND device_id = $2
		LIMIT 1`

	err = h.db.QueryRow(ctx, eq, endpointID, device.ID).Scan(
		&endpoint.ID, &endpoint.DeviceID, &endpoint.Type,
		&endpoint.Port, &endpoint.Label, &endpoint.Protocol, &endpoint.Enabled,
	)
	if err != nil {
		return nil, nil, err
	}

	return device, endpoint, nil
}

// fetchSession retrieves a session by ID, scoped to the tenant.
func (h *Handler) fetchSession(ctx context.Context, sessionID, tenantID string) (*models.Session, error) {
	const q = `
		SELECT id, device_id, endpoint_id, user_id, tenant_id, status, local_port,
		       delivery_mode, ttl_seconds, idle_timeout_seconds, started_at, expires_at,
		       stopped_at, stop_reason, audit_data
		FROM sessions
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1`

	s := &models.Session{}
	err := h.db.QueryRow(ctx, q, sessionID, tenantID).Scan(
		&s.ID, &s.DeviceID, &s.EndpointID, &s.UserID, &s.TenantID,
		&s.Status, &s.LocalPort, &s.DeliveryMode, &s.TTLSeconds,
		&s.IdleTimeoutSeconds, &s.StartedAt, &s.ExpiresAt,
		&s.StoppedAt, &s.StopReason, &s.AuditData,
	)
	return s, err
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
