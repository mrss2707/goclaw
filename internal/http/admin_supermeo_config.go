package http

import (
	"encoding/json"
	"net/http"

	"github.com/nextlevelbuilder/goclaw/supermeo/admin"
)

type AdminSupermeoConfigHandler struct {
	svc *admin.SupermeoConfigService
}

func NewAdminSupermeoConfigHandler(svc *admin.SupermeoConfigService) *AdminSupermeoConfigHandler {
	return &AdminSupermeoConfigHandler{svc: svc}
}

func (h *AdminSupermeoConfigHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/supermeo-config", requireAuth("owner", h.handleGetConfig))
	mux.HandleFunc("PUT /v1/admin/supermeo-config", requireAuth("owner", h.handleUpdateConfig))
}

func (h *AdminSupermeoConfigHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.GetConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *AdminSupermeoConfigHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Google *admin.GoogleOAuth2Settings `json:"google"`
		SMTP   *admin.SMTPSettings        `json:"smtp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Google != nil {
		if err := h.svc.SetGoogleConfig(r.Context(), req.Google); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.SMTP != nil {
		if err := h.svc.SetSMTPConfig(r.Context(), req.SMTP); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	cfg, err := h.svc.GetConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}
