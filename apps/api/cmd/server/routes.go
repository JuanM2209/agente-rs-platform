package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nucleus-portal/api/internal/audit"
	"github.com/nucleus-portal/api/internal/auth"
	"github.com/nucleus-portal/api/internal/bridge"
	"github.com/nucleus-portal/api/internal/config"
	"github.com/nucleus-portal/api/internal/database"
	"github.com/nucleus-portal/api/internal/device"
	appMiddleware "github.com/nucleus-portal/api/internal/middleware"
	"github.com/nucleus-portal/api/internal/session"
	"github.com/nucleus-portal/api/internal/ws"
)

// buildRouter wires all application routes and returns the root http.Handler.
func buildRouter(cfg *config.Config, hub *ws.AgentHub) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ──────────────────────────────────────────────────────
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.RequestID)
	r.Use(appMiddleware.Logger)

	// CORS — tighten origins per environment in production.
	allowedOrigins := cfg.APICORSOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Dependencies ──────────────────────────────────────────────────────────
	db := database.GetPool()
	redisClient := database.GetRedis()

	authHandler := auth.NewHandler(db, redisClient, cfg.JWTSecret, cfg.JWTExpiryHours)
	deviceHandler := device.NewHandler(db, hub)
	sessionHandler := session.NewHandler(db, hub)
	bridgeHandler := bridge.NewHandler(db, hub)
	auditHandler := audit.NewHandler(db)

	jwtAuth := auth.JWTMiddleware(cfg.JWTSecret)
	requireAdmin := appMiddleware.RequireRole("admin")

	// ── WebSocket agent endpoint ──────────────────────────────────────────────
	r.Get("/ws/agent", agentAuthMiddleware(cfg.AgentWSSecret, hub))

	// ── Health check ─────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"status":"ok"}}`))
	})

	// ── API v1 ────────────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {

		// Public auth routes (no JWT required).
		r.Post("/auth/login", authHandler.Login)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(jwtAuth)

			r.Post("/auth/logout", authHandler.Logout)

			// Current user.
			r.Get("/me", authHandler.Me)
			r.Get("/me/active-sessions", sessionHandler.ListActiveSessions)
			r.Get("/me/export-history", sessionHandler.ListExportHistory)

			// Device routes.
			r.Route("/devices/{deviceId}", func(r chi.Router) {
				r.Get("/", deviceHandler.GetDevice)
				r.Get("/inventory", deviceHandler.GetInventory)
				r.Post("/scan", deviceHandler.ScanDevice)
				r.Post("/sessions", sessionHandler.CreateSession)
				r.Get("/export-history", auditHandler.GetDeviceExportHistory)
				r.Post("/bridges/modbus-serial", bridgeHandler.StartModbusBridge)
			})

			// Session management.
			r.Delete("/sessions/{sessionId}", sessionHandler.StopSession)
			r.Post("/sessions/{sessionId}/telemetry", sessionHandler.UpdateSessionTelemetry)

			// Bridge management.
			r.Delete("/bridges/{bridgeId}", bridgeHandler.StopBridge)

			// Admin-only routes.
			r.Group(func(r chi.Router) {
				r.Use(requireAdmin)
				r.Get("/admin/devices", deviceHandler.ListDevices)
			})
		})
	})

	return r
}

// agentAuthMiddleware validates the AGENT_WS_SECRET before handing off to the hub.
func agentAuthMiddleware(secret string, hub *ws.AgentHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-Agent-Secret")
		if provided == "" {
			provided = r.URL.Query().Get("secret")
		}
		if provided != secret {
			http.Error(w, `{"success":false,"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		hub.HandleAgentConnection(w, r)
	}
}
