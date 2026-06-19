package userauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type EmailVerifier struct {
	users    store.UserStore
	smtp     *SMTPClient
	sessions store.UserSessionStore
	jwt      *JWTService
	mapper   *UserTenantMapper
	limiter  *AuthRateLimiter
}

func NewEmailVerifier(
	users store.UserStore,
	smtp *SMTPClient,
	sessions store.UserSessionStore,
	jwt *JWTService,
	mapper *UserTenantMapper,
	limiter *AuthRateLimiter,
) *EmailVerifier {
	return &EmailVerifier{
		users:    users,
		smtp:     smtp,
		sessions: sessions,
		jwt:      jwt,
		mapper:   mapper,
		limiter:  limiter,
	}
}

func (v *EmailVerifier) SendVerificationCode(ctx context.Context, userID uuid.UUID) error {
	user, err := v.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("get user: %w", err)
	}
	if user.Email == nil || *user.Email == "" {
		return fmt.Errorf("user has no email")
	}
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	if !v.limiter.AllowVerify(userID.String()) {
		return fmt.Errorf("rate limit exceeded")
	}

	code := GenerateVerificationCode()
	exp := time.Now().Add(15 * time.Minute)
	if err := v.users.SetVerificationCode(ctx, userID, code, exp); err != nil {
		return fmt.Errorf("set code: %w", err)
	}

	if v.smtp != nil {
		if err := v.smtp.SendVerificationCode(*user.Email, code); err != nil {
			slog.Warn("userauth.verify.send_email_failed", "user_id", userID, "error", err)
		}
	}
	slog.Info("userauth.verify.code_sent", "user_id", userID)
	return nil
}

func (v *EmailVerifier) VerifyCode(ctx context.Context, userID uuid.UUID, code string) (string, error) {
	if !v.limiter.AllowVerify(userID.String()) {
		return "", fmt.Errorf("rate limit exceeded")
	}

	user, err := v.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("user not found")
		}
		return "", fmt.Errorf("get user: %w", err)
	}

	if user.VerificationCode == nil || user.VerificationExp == nil {
		return "", fmt.Errorf("no verification code pending")
	}
	if time.Now().After(*user.VerificationExp) {
		return "", fmt.Errorf("verification code expired")
	}
	if !ConstantTimeCompare(code, *user.VerificationCode) {
		slog.Warn("security.auth.verify_failed", "user_id", userID)
		return "", fmt.Errorf("invalid verification code")
	}

	if err := v.users.VerifyEmail(ctx, userID); err != nil {
		return "", fmt.Errorf("verify email: %w", err)
	}

	token, _, err := v.jwt.Sign(userID, derefStr(user.Email))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return token, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
