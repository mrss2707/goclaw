package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type EmailVerifyHandler struct {
	verifier *userauth.EmailVerifier
	mapper   *userauth.UserTenantMapper
	users    store.UserStore
	jwt      *userauth.JWTService
}

func NewEmailVerifyHandler(
	verifier *userauth.EmailVerifier,
	mapper *userauth.UserTenantMapper,
	users store.UserStore,
	jwt *userauth.JWTService,
) *EmailVerifyHandler {
	return &EmailVerifyHandler{
		verifier: verifier,
		mapper:   mapper,
		users:    users,
		jwt:      jwt,
	}
}

func (h *EmailVerifyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/verify-email", h.handleVerifyEmail)
	mux.HandleFunc("POST /v1/auth/resend-code", h.handleResendCode)
	mux.HandleFunc("POST /v1/auth/forgot-password", h.handleForgotPassword)
	mux.HandleFunc("POST /v1/auth/reset-password", h.handleResetPassword)
}

func (h *EmailVerifyHandler) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
		return
	}

	token, err := h.verifier.VerifyCode(r.Context(), userID, req.Code)
	if err != nil {
		slog.Warn("security.auth.verify_failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	tenantID, _ := h.mapper.GetOrCreateTenant(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]any{
		"token":     token,
		"tenant_id": tenantID,
		"user_id":   userID,
	})
}

func (h *EmailVerifyHandler) handleResendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
		return
	}

	if err := h.verifier.SendVerificationCode(r.Context(), userID); err != nil {
		slog.Warn("userauth.resend_code_failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "verification code sent"})
}

func (h *EmailVerifyHandler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := h.users.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset code has been sent"})
		return
	}

	if err := h.verifier.SendVerificationCode(r.Context(), user.ID); err != nil {
		slog.Warn("userauth.forgot_password_failed", "user_id", user.ID, "error", err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset code has been sent"})
		return
	}

	slog.Info("security.auth.forgot_password", "user_id", user.ID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset code has been sent"})
}

func (h *EmailVerifyHandler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"user_id"`
		Code    string `json:"code"`
		NewPass string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
		return
	}

	if err := userauth.ValidatePassword(req.NewPass); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	_, err = h.verifier.VerifyCode(r.Context(), userID, req.Code)
	if err != nil {
		slog.Warn("security.auth.reset_password_verify_failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	hash, err := userauth.HashPassword(req.NewPass)
	if err != nil {
		slog.Error("userauth.reset_password.hash", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := h.users.UpdatePassword(r.Context(), userID, hash); err != nil {
		slog.Error("userauth.reset_password.update", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	slog.Info("security.auth.password_reset", "user_id", userID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password reset successful"})
}
