package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nextlevelbuilder/goclaw/supermeo/admin"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type AdminSetupHandler struct {
	svc         *admin.AdminSetupService
	rateLimiter *userauth.AuthRateLimiter
}

func NewAdminSetupHandler(svc *admin.AdminSetupService, rateLimiter *userauth.AuthRateLimiter) *AdminSetupHandler {
	return &AdminSetupHandler{svc: svc, rateLimiter: rateLimiter}
}

func (h *AdminSetupHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/admin/setup", h.handleSetup)
	mux.HandleFunc("GET /v1/admin/setup", h.handleCheckSetup)
}

func (h *AdminSetupHandler) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.AllowRegister(r.RemoteAddr) {
		slog.Warn("security.admin.setup_rate_limited", "ip", r.RemoteAddr)
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded; try again in 15 minutes"})
		return
	}

	var req admin.CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	resp, err := h.svc.CreateFirstAdmin(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		msg := err.Error()
		switch msg {
		case "admin account already exists":
			status = http.StatusForbidden
		case "email is required", "password is required":
			status = http.StatusBadRequest
		case "email already in use":
			status = http.StatusConflict
		}
		if strings.HasPrefix(msg, "password") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AdminSetupHandler) handleCheckSetup(w http.ResponseWriter, r *http.Request) {
	complete, err := h.svc.CheckSetup(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"setup_complete": complete})
}
