package admin

import (
	"context"
	"log/slog"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type SupermeoConfigService struct {
	sysConfigs store.SystemConfigStore
}

func NewSupermeoConfigService(sysConfigs store.SystemConfigStore) *SupermeoConfigService {
	return &SupermeoConfigService{sysConfigs: sysConfigs}
}

type GoogleOAuth2Settings struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	Enabled      bool   `json:"enabled"`
}

type SMTPSettings struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	FromName string `json:"from_name"`
	Enabled  bool   `json:"enabled"`
}

type SupermeoConfig struct {
	Google *GoogleOAuth2Settings `json:"google"`
	SMTP   *SMTPSettings        `json:"smtp"`
}

func (s *SupermeoConfigService) GetConfig(ctx context.Context) (*SupermeoConfig, error) {
	getVal := func(key string) string {
		v, _ := s.sysConfigs.Get(ctx, key)
		return v
	}

	cfg := &SupermeoConfig{
		Google: &GoogleOAuth2Settings{
			ClientID:     getVal("supermeo.google.client_id"),
			ClientSecret: maskSecret(getVal("supermeo.google.client_secret")),
			RedirectURL:  getVal("supermeo.google.redirect_url"),
			Enabled:      getVal("supermeo.google.client_id") != "",
		},
		SMTP: &SMTPSettings{
			Host:     getVal("supermeo.smtp.host"),
			Port:     getVal("supermeo.smtp.port"),
			Username: getVal("supermeo.smtp.username"),
			Password: maskSecret(getVal("supermeo.smtp.password")),
			FromName: getVal("supermeo.smtp.from_name"),
			Enabled:  getVal("supermeo.smtp.host") != "",
		},
	}
	return cfg, nil
}

func (s *SupermeoConfigService) SetGoogleConfig(ctx context.Context, settings *GoogleOAuth2Settings) error {
	if settings.ClientID != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.google.client_id", settings.ClientID); err != nil {
			return err
		}
	}
	if settings.ClientSecret != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.google.client_secret", settings.ClientSecret); err != nil {
			return err
		}
	}
	if settings.RedirectURL != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.google.redirect_url", settings.RedirectURL); err != nil {
			return err
		}
	}
	slog.Info("admin.config.google_oauth2_updated")
	return nil
}

func (s *SupermeoConfigService) SetSMTPConfig(ctx context.Context, settings *SMTPSettings) error {
	if settings.Host != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.smtp.host", settings.Host); err != nil {
			return err
		}
	}
	if settings.Port != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.smtp.port", settings.Port); err != nil {
			return err
		}
	}
	if settings.Username != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.smtp.username", settings.Username); err != nil {
			return err
		}
	}
	if settings.Password != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.smtp.password", settings.Password); err != nil {
			return err
		}
	}
	if settings.FromName != "" {
		if err := s.sysConfigs.Set(ctx, "supermeo.smtp.from_name", settings.FromName); err != nil {
			return err
		}
	}
	slog.Info("admin.config.smtp_updated")
	return nil
}

func (s *SupermeoConfigService) GetGoogleOAuth2Config(ctx context.Context) *userauth.GoogleOAuth2Config {
	getVal := func(key string) string {
		v, _ := s.sysConfigs.Get(ctx, key)
		return v
	}
	return &userauth.GoogleOAuth2Config{
		ClientID:     getVal("supermeo.google.client_id"),
		ClientSecret: getVal("supermeo.google.client_secret"),
		RedirectURL:  getVal("supermeo.google.redirect_url"),
	}
}

func (s *SupermeoConfigService) GetSMTPConfig(ctx context.Context) userauth.SMTPConfig {
	getVal := func(key string) string {
		v, _ := s.sysConfigs.Get(ctx, key)
		return v
	}
	port := getVal("supermeo.smtp.port")
	if port == "" {
		port = "587"
	}
	return userauth.SMTPConfig{
		Host:     getVal("supermeo.smtp.host"),
		Port:     port,
		Username: getVal("supermeo.smtp.username"),
		Password: getVal("supermeo.smtp.password"),
		FromName: getVal("supermeo.smtp.from_name"),
	}
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}
