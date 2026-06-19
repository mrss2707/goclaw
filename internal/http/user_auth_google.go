package http

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type GoogleOAuth2Handler struct {
	oauth  *userauth.GoogleOAuth2Handler
	store  *userauth.GoogleStore
	users  store.UserStore
	jwt    *userauth.JWTService
	mapper *userauth.UserTenantMapper
}

func NewGoogleOAuth2Handler(
	oauth *userauth.GoogleOAuth2Handler,
	store *userauth.GoogleStore,
	users store.UserStore,
	jwt *userauth.JWTService,
	mapper *userauth.UserTenantMapper,
) *GoogleOAuth2Handler {
	return &GoogleOAuth2Handler{
		oauth:  oauth,
		store:  store,
		users:  users,
		jwt:    jwt,
		mapper: mapper,
	}
}

func (h *GoogleOAuth2Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/auth/google/login", h.handleLogin)
	mux.HandleFunc("GET /v1/auth/google/callback", h.handleCallback)
}

func (h *GoogleOAuth2Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := userauth.GenerateOAuthState()
	authURL := h.oauth.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *GoogleOAuth2Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code parameter"})
		return
	}

	info, err := h.oauth.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("userauth.google.exchange_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to verify Google identity"})
		return
	}

	user, err := h.store.FindOrCreateUserByGoogle(r.Context(), info)
	if err != nil {
		slog.Error("userauth.google.find_create_user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, _, err := h.jwt.Sign(user.ID, derefStr(user.Email))
	if err != nil {
		slog.Error("userauth.google.sign_jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	tenantID, _ := h.mapper.GetOrCreateTenant(r.Context(), user.ID)

	frontendURL := "http://localhost:5173" // default dev URL
	redirectURL := frontendURL + "/auth/callback?token=" + url.QueryEscape(token) +
		"&tenant_id=" + tenantID.String()

	slog.Info("security.auth.google_login", "user_id", user.ID, "email", maskEmail(derefStr(user.Email)))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
