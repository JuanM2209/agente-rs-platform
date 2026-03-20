package session

import (
	"context"
	"encoding/json"
	"fmt"
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

type sessionAuditEnvelope struct {
	RemoteHost string                   `json:"remote_host,omitempty"`
	Telemetry  *models.SessionTelemetry `json:"telemetry,omitempty"`
}

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

type updateTelemetryRequest struct {
	ConnectionStatus string     `json:"connection_status"`
	LatencyMS        *int       `json:"latency_ms"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	LastError        string     `json:"last_error"`
	ProbeSource      string     `json:"probe_source"`
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
		req.TTLSeconds = 3600
	}
	if req.IdleTimeoutSeconds <= 0 {
		req.IdleTimeoutSeconds = 900
	}

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

	if !endpoint.Enabled {
		writeJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "endpoint is disabled",
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
	remoteHost := strings.TrimSpace(device.IPAddress)
	telemetry := &models.SessionTelemetry{
		ConnectionStatus: "pending",
		ProbeSource:      "helper",
	}
	if deliveryMode == models.DeliveryModeWeb {
		telemetry.ProbeSource = "system"
	}

	auditData, err := marshalSessionAudit(remoteHost, telemetry)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("CreateSession: marshal audit data")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to build session metadata",
		})
		return
	}

	tunnelURL := ""
	if deliveryMode == models.DeliveryModeWeb {
		tunnelURL = buildWebAccessURL(endpoint.Protocol, remoteHost, endpoint.Port)
	}

	const insertQ = `
		INSERT INTO sessions
			(id, device_id, endpoint_id, user_id, tenant_id, status, local_port,
			 remote_port, delivery_mode, ttl_seconds, idle_timeout_seconds,
			 started_at, expires_at, tunnel_url, audit_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = h.db.Exec(r.Context(), insertQ,
		sessionID,
		device.ID,
		endpoint.ID,
		userID,
		tenantID,
		models.SessionStatusActive,
		endpoint.Port,
		endpoint.Port,
		string(deliveryMode),
		req.TTLSeconds,
		req.IdleTimeoutSeconds,
		now,
		expiresAt,
		tunnelURL,
		auditData,
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
	}

	session := models.Session{
		ID:                 sessionID,
		DeviceID:           device.ID,
		EndpointID:         endpoint.ID,
		UserID:             userID,
		TenantID:           tenantID,
		Status:             models.SessionStatusActive,
		LocalPort:          endpoint.Port,
		RemotePort:         endpoint.Port,
		RemoteHost:         remoteHost,
		DeliveryMode:       deliveryMode,
		TTLSeconds:         req.TTLSeconds,
		IdleTimeoutSeconds: req.IdleTimeoutSeconds,
		StartedAt:          now,
		ExpiresAt:          &expiresAt,
		TunnelURL:          tunnelURL,
		AuditData:          auditData,
		Telemetry:          telemetry,
		Device:             device,
		Endpoint:           endpoint,
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
		models.SessionStatusStopped, now, "user_stopped", sessionID,
	); dbErr != nil {
		log.Error().Err(dbErr).Str("session_id", sessionID).Msg("StopSession: db update failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to stop session",
		})
		return
	}

	if err := h.recordHistory(r.Context(), session, now, "user_stopped"); err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("StopSession: failed to record history")
	}

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

// UpdateSessionTelemetry handles POST /api/v1/sessions/:sessionId/telemetry.
func (h *Handler) UpdateSessionTelemetry(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	session, err := h.fetchSession(r.Context(), sessionID, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "session not found",
			})
			return
		}
		log.Error().Err(err).Str("session_id", sessionID).Msg("UpdateSessionTelemetry: fetch session")
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

	var req updateTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	checkedAt := time.Now().UTC()
	if req.LastCheckedAt != nil {
		checkedAt = req.LastCheckedAt.UTC()
	}

	status := strings.TrimSpace(req.ConnectionStatus)
	if status == "" {
		status = "pending"
	}

	telemetry := &models.SessionTelemetry{
		ConnectionStatus: status,
		LatencyMS:        req.LatencyMS,
		LastCheckedAt:    &checkedAt,
		LastError:        strings.TrimSpace(req.LastError),
		ProbeSource:      strings.TrimSpace(req.ProbeSource),
	}
	if telemetry.ProbeSource == "" {
		telemetry.ProbeSource = "helper"
	}

	audit := parseSessionAudit(session.AuditData)
	audit.Telemetry = telemetry

	rawAudit, err := json.Marshal(audit)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("UpdateSessionTelemetry: marshal audit data")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to encode telemetry",
		})
		return
	}

	const updateQ = `
		UPDATE sessions
		SET audit_data = $1, last_activity_at = $2
		WHERE id = $3`

	if _, err := h.db.Exec(r.Context(), updateQ, rawAudit, checkedAt, sessionID); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("UpdateSessionTelemetry: db update failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "failed to persist telemetry",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    telemetry,
	})
}

// ListActiveSessions handles GET /api/v1/me/active-sessions.
func (h *Handler) ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	tenantID := middleware.GetTenantID(r.Context())

	const q = `
		SELECT s.id, s.device_id, s.endpoint_id, s.user_id, s.tenant_id, s.status,
		       s.local_port, s.remote_port, s.delivery_mode, s.ttl_seconds,
		       s.idle_timeout_seconds, s.started_at, s.expires_at, s.last_activity_at,
		       s.stopped_at, s.stop_reason, s.tunnel_url, s.audit_data,
		       d.id, d.tenant_id, d.site_id, d.device_id, d.display_name, d.status,
		       d.last_seen, d.firmware_version, d.ip_address, d.created_at,
		       e.id, e.device_id, e.type, e.port, e.label, e.protocol, e.enabled
		FROM sessions s
		JOIN devices d ON d.id = s.device_id
		JOIN endpoints e ON e.id = s.endpoint_id
		WHERE s.user_id = $1 AND s.tenant_id = $2 AND s.status = 'active'
		ORDER BY s.started_at DESC`

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
		var d models.Device
		var e models.Endpoint

		if scanErr := rows.Scan(
			&s.ID, &s.DeviceID, &s.EndpointID, &s.UserID, &s.TenantID, &s.Status,
			&s.LocalPort, &s.RemotePort, &s.DeliveryMode, &s.TTLSeconds,
			&s.IdleTimeoutSeconds, &s.StartedAt, &s.ExpiresAt, &s.LastActivityAt,
			&s.StoppedAt, &s.StopReason, &s.TunnelURL, &s.AuditData,
			&d.ID, &d.TenantID, &d.SiteID, &d.DeviceID, &d.DisplayName, &d.Status,
			&d.LastSeen, &d.FirmwareVersion, &d.IPAddress, &d.CreatedAt,
			&e.ID, &e.DeviceID, &e.Type, &e.Port, &e.Label, &e.Protocol, &e.Enabled,
		); scanErr != nil {
			log.Error().Err(scanErr).Msg("ListActiveSessions: scan error")
			continue
		}

		audit := parseSessionAudit(s.AuditData)
		s.RemoteHost = audit.RemoteHost
		s.Telemetry = audit.Telemetry
		s.Device = &d
		s.Endpoint = &e
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

		if telemetry := parseHistoryTelemetry(eh.Metadata); telemetry != nil {
			eh.Telemetry = telemetry
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
		       remote_port, delivery_mode, ttl_seconds, idle_timeout_seconds,
		       started_at, expires_at, last_activity_at, stopped_at, stop_reason,
		       tunnel_url, audit_data
		FROM sessions
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1`

	s := &models.Session{}
	err := h.db.QueryRow(ctx, q, sessionID, tenantID).Scan(
		&s.ID, &s.DeviceID, &s.EndpointID, &s.UserID, &s.TenantID,
		&s.Status, &s.LocalPort, &s.RemotePort, &s.DeliveryMode,
		&s.TTLSeconds, &s.IdleTimeoutSeconds, &s.StartedAt, &s.ExpiresAt,
		&s.LastActivityAt, &s.StoppedAt, &s.StopReason, &s.TunnelURL, &s.AuditData,
	)
	if err != nil {
		return nil, err
	}

	audit := parseSessionAudit(s.AuditData)
	s.RemoteHost = audit.RemoteHost
	s.Telemetry = audit.Telemetry
	return s, nil
}

func (h *Handler) recordHistory(
	ctx context.Context,
	session *models.Session,
	stoppedAt time.Time,
	stopReason string,
) error {
	durationSeconds := int(stoppedAt.Sub(session.StartedAt).Seconds())
	if durationSeconds < 0 {
		durationSeconds = 0
	}

	var siteID string
	_ = h.db.QueryRow(ctx, `SELECT COALESCE(site_id::text, '') FROM devices WHERE id = $1`, session.DeviceID).Scan(&siteID)

	const q = `
		INSERT INTO export_history
			(id, session_id, user_id, device_id, endpoint_id, tenant_id, site_id,
			 started_at, stopped_at, stop_reason, local_bind_port, delivery_mode,
			 duration_seconds, bytes_transferred, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := h.db.Exec(ctx, q,
		uuid.New().String(),
		session.ID,
		session.UserID,
		session.DeviceID,
		session.EndpointID,
		session.TenantID,
		siteID,
		session.StartedAt,
		stoppedAt,
		stopReason,
		session.LocalPort,
		string(session.DeliveryMode),
		durationSeconds,
		0,
		session.AuditData,
	)
	return err
}

func buildWebAccessURL(protocol, remoteHost string, port int) string {
	if remoteHost == "" || port <= 0 {
		return ""
	}
	scheme := "http"
	if strings.EqualFold(protocol, "https") {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, remoteHost, port)
}

func marshalSessionAudit(remoteHost string, telemetry *models.SessionTelemetry) (json.RawMessage, error) {
	return json.Marshal(sessionAuditEnvelope{
		RemoteHost: remoteHost,
		Telemetry:  telemetry,
	})
}

func parseSessionAudit(raw json.RawMessage) sessionAuditEnvelope {
	if len(raw) == 0 {
		return sessionAuditEnvelope{}
	}
	var envelope sessionAuditEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return sessionAuditEnvelope{}
	}
	return envelope
}

func parseHistoryTelemetry(raw json.RawMessage) *models.SessionTelemetry {
	if len(raw) == 0 {
		return nil
	}
	var payload struct {
		Telemetry *models.SessionTelemetry `json:"telemetry"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return payload.Telemetry
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
