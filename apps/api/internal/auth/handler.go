package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nucleus-portal/api/internal/middleware"
	"github.com/nucleus-portal/api/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// Handler holds dependencies for auth HTTP handlers.
type Handler struct {
	db           *pgxpool.Pool
	redis        *redis.Client
	jwtSecret    string
	jwtExpiryHrs int
}

// NewHandler constructs an auth Handler.
func NewHandler(db *pgxpool.Pool, redisClient *redis.Client, jwtSecret string, jwtExpiryHrs int) *Handler {
	return &Handler{
		db:           db,
		redis:        redisClient,
		jwtSecret:    jwtSecret,
		jwtExpiryHrs: jwtExpiryHrs,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int         `json:"expires_in"`
	User         models.User `json:"user"`
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "email and password are required",
		})
		return
	}

	user, err := h.findUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "invalid credentials",
			})
			return
		}
		log.Error().Err(err).Str("email", req.Email).Msg("login: db lookup failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "invalid credentials",
		})
		return
	}

	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		log.Error().Err(err).Msg("login: token generation failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	refreshToken := uuid.New().String()
	refreshKey := fmt.Sprintf("refresh:%s", refreshToken)
	ttl := time.Duration(h.jwtExpiryHrs*7) * time.Hour // refresh lives 7x longer
	if err := h.redis.Set(r.Context(), refreshKey, user.ID, ttl).Err(); err != nil {
		log.Error().Err(err).Msg("login: failed to store refresh token")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    h.jwtExpiryHrs * 3600,
			User:         *user,
		},
	})
}

// Logout handles POST /api/v1/auth/logout.
// It adds the current token to a Redis denylist so it cannot be reused.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "missing authorization header",
		})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		writeJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "invalid authorization header",
		})
		return
	}

	tokenStr := parts[1]
	denyKey := fmt.Sprintf("denylist:%s", tokenStr)
	ttl := time.Duration(h.jwtExpiryHrs) * time.Hour

	if err := h.redis.Set(r.Context(), denyKey, "1", ttl).Err(); err != nil {
		log.Error().Err(err).Msg("logout: failed to denylist token")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "logged out successfully"},
	})
}

// Me handles GET /api/v1/me — returns the current user from the JWT context.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "not authenticated",
		})
		return
	}

	user, err := h.findUserByID(r.Context(), userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "user not found",
			})
			return
		}
		log.Error().Err(err).Str("user_id", userID).Msg("me: db lookup failed")
		writeJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
	})
}

// generateAccessToken creates a signed JWT for the given user.
func (h *Handler) generateAccessToken(user *models.User) (string, error) {
	expiry := time.Now().Add(time.Duration(h.jwtExpiryHrs) * time.Hour)
	claims := Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *Handler) findUserByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `
		SELECT id, tenant_id, email, password_hash, role, display_name, created_at
		FROM users
		WHERE email = $1
		LIMIT 1`

	user := &models.User{}
	err := h.db.QueryRow(ctx, q, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.DisplayName,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (h *Handler) findUserByID(ctx context.Context, userID string) (*models.User, error) {
	const q = `
		SELECT id, tenant_id, email, password_hash, role, display_name, created_at
		FROM users
		WHERE id = $1
		LIMIT 1`

	user := &models.User{}
	err := h.db.QueryRow(ctx, q, userID).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.DisplayName,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode error")
	}
}
