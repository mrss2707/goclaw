package admin

import (
	"context"
	"log/slog"

	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

func (s *SupermeoConfigService) NewGoogleOAuth2HandlerFromStore(ctx context.Context) *userauth.GoogleOAuth2Handler {
	cfg := s.GetGoogleOAuth2Config(ctx)
	slog.Info("admin.config.google_oauth2_from_store", "configured", cfg.ClientID != "")
	return userauth.NewGoogleOAuth2Handler(cfg)
}

func (s *SupermeoConfigService) NewSMTPClientFromStore(ctx context.Context) *userauth.SMTPClient {
	cfg := s.GetSMTPConfig(ctx)
	slog.Info("admin.config.smtp_from_store", "configured", cfg.Host != "")
	return userauth.NewSMTPClient(cfg)
}
