package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type UserAuthHandler struct {
	users       store.UserStore
	jwt         *userauth.JWTService
	mapper      *userauth.UserTenantMapper
	rateLimiter *userauth.AuthRateLimiter
}

func NewUserAuthHandler(users store.UserStore, jwt *userauth.JWTService, mapper *userauth.UserTenantMapper, limiter *userauth.AuthRateLimiter) *UserAuthHandler {
	return &UserAuthHandler{
		users:       users,
		jwt:         jwt,
		mapper:      mapper,
		rateLimiter: limiter,
	}
}

func (h *UserAuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /v1/auth/register", h.handleRegister)
}

func (h *UserAuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !h.rateLimiter.AllowLogin(req.Email) {
		slog.Warn("security.auth.login_rate_limited", "email", maskEmail(req.Email))
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded; try again in 60 seconds"})
		return
	}

	user, err := h.users.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("security.auth.login_failed", "email", maskEmail(req.Email), "reason", "not_found")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		slog.Error("userauth.login.get_user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if user.PasswordHash == nil || !userauth.VerifyPassword(*user.PasswordHash, req.Password) {
		slog.Warn("security.auth.login_failed", "email", maskEmail(req.Email), "reason", "password_mismatch")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if !user.EmailVerified {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "email not verified", "user_id": user.ID.String()})
		return
	}

	token, _, err := h.jwt.Sign(user.ID, derefStr(user.Email))
	if err != nil {
		slog.Error("userauth.login.sign_jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	tenantID, _ := h.mapper.GetOrCreateTenant(r.Context(), user.ID)
	identity := map[string]any{
		"user_id":   user.ID,
		"tenant_id": tenantID,
		"role":      "admin",
		"locale":    user.Locale,
		"email":     derefStr(user.Email),
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": identity})
}

func (h *UserAuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email required"})
		return
	}

	if err := userauth.ValidatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if !h.rateLimiter.AllowRegister(r.RemoteAddr) {
		slog.Warn("security.auth.register_rate_limited", "email", maskEmail(req.Email))
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded; try again in 60 seconds"})
		return
	}

	_, err := h.users.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		slog.Error("userauth.register.check_email", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	hash, err := userauth.HashPassword(req.Password)
	if err != nil {
		slog.Error("userauth.register.hash", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	user := &store.User{
		Email:         &req.Email,
		PasswordHash:  &hash,
		DisplayName:   &req.DisplayName,
		EmailVerified: false,
		Locale:        "en",
		Metadata:      []byte("{}"),
	}
	if err := h.users.CreateUser(r.Context(), user); err != nil {
		if store.IsDuplicateKeyErr(err) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		slog.Error("userauth.register.create", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	code := userauth.GenerateVerificationCode()
	if err := h.users.SetVerificationCode(r.Context(), user.ID, code, time.Now().Add(15*time.Minute)); err != nil {
		slog.Warn("userauth.register.set_code", "error", err)
	}

	slog.Info("security.auth.registered", "user_id", user.ID, "email", maskEmail(req.Email))
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": user.ID, "message": "registration successful. check email for verification code."})
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) <= 2 {
		return email
	}
	return parts[0][:2] + "***@" + parts[1]
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
