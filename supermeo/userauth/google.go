package userauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type GoogleOAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GoogleOAuth2Handler struct {
	config   *oauth2.Config
	stateTTL int
}

type GoogleUserInfo struct {
	GoogleID      string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

func NewGoogleOAuth2Handler(cfg *GoogleOAuth2Config) *GoogleOAuth2Handler {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &GoogleOAuth2Handler{
		config:   oauthCfg,
		stateTTL: 300,
	}
}

func NewGoogleOAuth2HandlerFromEnv() *GoogleOAuth2Handler {
	return NewGoogleOAuth2Handler(&GoogleOAuth2Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	})
}

func (h *GoogleOAuth2Handler) AuthCodeURL(state string) string {
	return h.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (h *GoogleOAuth2Handler) Exchange(ctx context.Context, code string) (*GoogleUserInfo, error) {
	token, err := h.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	idTokenRaw, ok := token.Extra("id_token").(string)
	if !ok || idTokenRaw == "" {
		return nil, fmt.Errorf("no id_token in response")
	}

	payload, err := idtoken.Validate(ctx, idTokenRaw, h.config.ClientID)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}

	info := &GoogleUserInfo{
		GoogleID:      payload.Subject,
		EmailVerified: true,
	}
	if email, ok := payload.Claims["email"].(string); ok {
		info.Email = email
	}
	if name, ok := payload.Claims["name"].(string); ok {
		info.Name = name
	}
	if picture, ok := payload.Claims["picture"].(string); ok {
		info.Picture = picture
	}
	return info, nil
}

func GenerateOAuthState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *GoogleOAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	info, err := h.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Authenticated as %s (%s)", info.Name, info.Email)
}
