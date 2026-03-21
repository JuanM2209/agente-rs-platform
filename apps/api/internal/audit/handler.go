package audit

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nucleus-portal/api/internal/middleware"
	"github.com/nucleus-portal/api/internal/models"
	"github.com/rs/zerolog/log"
)

// Handler holds dependencies for audit HTTP handlers.
type Handler struct {
	db *pgxpool.Pool
}

// NewHandler constructs an audit Handler.
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

// GetDeviceExportHistory handles GET /api/v1/devices/:deviceId/export-history.
func (h *Handler) GetDeviceExportHistory(w http.ResponseWriter, r *http.Request) {
	deviceStringID := chi.URLParam(r, "deviceId")
	tenantID := middleware.GetTenantID(r.Context())

	// Resolve internal device UUID, confirming tenant ownership.
	var internalDeviceID string
	const resolveQ = `SELECT id FROM devices WHERE device_id = $1 AND tenant_id = $2 LIMIT 1`
	if err := h.db.QueryRow(r.Context(), resolveQ, deviceStringID, tenantID).Scan(&internalDeviceID); err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "device not found",
			})
			return
		}
		log.Error().Err(err).Str("device_id", deviceStringID).Msg("GetDeviceExportHistory: resolve device")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	const q = `
		SELECT id, COALESCE(session_id::text, ''), user_id, device_id, COALESCE(endpoint_id::text, ''), tenant_id, COALESCE(site_id::text, ''),
		       started_at, stopped_at, COALESCE(stop_reason, ''), COALESCE(local_bind_port, 0), COALESCE(delivery_mode, ''), COALESCE(metadata, '{}'::jsonb)
		FROM export_history
		WHERE device_id = $1 AND tenant_id = $2
		ORDER BY started_at DESC
		LIMIT 200`

	rows, err := h.db.Query(r.Context(), q, internalDeviceID, tenantID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceStringID).Msg("GetDeviceExportHistory: db error")
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
			log.Error().Err(scanErr).Msg("GetDeviceExportHistory: scan error")
			continue
		}
		history = append(history, eh)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("GetDeviceExportHistory: rows error")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    history,
		Meta:    &models.MetaInfo{Total: len(history), Page: 1, Limit: 200},
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
