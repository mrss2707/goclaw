package googlechat

import (
	"encoding/json"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/config"
)

func TestNew_Validation(t *testing.T) {
	msgBus := bus.New()

	tests := []struct {
		name    string
		cfg     config.GoogleChatConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pull config",
			cfg: config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				ProjectID:          "my-project",
				SubscriptionName:   "my-sub",
			},
			wantErr: false,
		},
		{
			name: "valid push config (no subscription needed)",
			cfg: config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				ProjectID:          "my-project",
				ConnectionMode:     "push",
			},
			wantErr: false,
		},
		{
			name: "missing service account",
			cfg: config.GoogleChatConfig{
				ProjectID:        "my-project",
				SubscriptionName: "my-sub",
			},
			wantErr: true,
			errMsg:  "service_account_json or service_account_file is required",
		},
		{
			name: "missing project ID",
			cfg: config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				SubscriptionName:   "my-sub",
			},
			wantErr: true,
			errMsg:  "project_id is required",
		},
		{
			name: "pull mode missing subscription",
			cfg: config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				ProjectID:          "my-project",
			},
			wantErr: true,
			errMsg:  "subscription_name is required for pull mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg, msgBus, nil, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !containsStr(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNew_Defaults(t *testing.T) {
	msgBus := bus.New()
	cfg := config.GoogleChatConfig{
		ServiceAccountJSON: `{"type":"service_account"}`,
		ProjectID:          "my-project",
		SubscriptionName:   "my-sub",
	}

	ch, err := New(cfg, msgBus, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check defaults
	if ch.cfg.ConnectionMode != "pull" {
		t.Errorf("expected default connection_mode 'pull', got %q", ch.cfg.ConnectionMode)
	}
	if ch.Name() != channels.TypeGoogleChat {
		t.Errorf("expected name %q, got %q", channels.TypeGoogleChat, ch.Name())
	}
	if ch.Type() != channels.TypeGoogleChat {
		t.Errorf("expected type %q, got %q", channels.TypeGoogleChat, ch.Type())
	}
	if ch.HistoryLimit() != channels.DefaultGroupHistoryLimit {
		t.Errorf("expected history limit %d, got %d", channels.DefaultGroupHistoryLimit, ch.HistoryLimit())
	}
	if !ch.RequireMention() {
		t.Error("expected require_mention to default to true")
	}
}

func TestNew_WithOptions(t *testing.T) {
	msgBus := bus.New()
	cfg := config.GoogleChatConfig{
		ServiceAccountJSON: `{"type":"service_account"}`,
		ProjectID:          "my-project",
		SubscriptionName:   "my-sub",
	}

	ch, err := New(cfg, msgBus, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check it's not running
	if ch.IsRunning() {
		t.Error("channel should not be running before Start()")
	}
}

func TestChannel_InterfaceAssertions(t *testing.T) {
	cfg := config.GoogleChatConfig{
		ServiceAccountJSON: `{"type":"service_account"}`,
		ProjectID:          "my-project",
		SubscriptionName:   "my-sub",
	}

	ch, err := New(cfg, bus.New(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify channel implements core interface
	var _ channels.Channel = ch
	if ch.IsAllowed("test") != true {
		t.Error("empty allowlist should allow all")
	}
	if ch.IsRunning() != false {
		t.Error("channel should not be running")
	}
}

func TestConfiguration_BlockReply(t *testing.T) {
	msgBus := bus.New()

	tests := []struct {
		name      string
		blockReply *bool
		wantNil    bool
		wantVal    bool
	}{
		{
			name:      "nil block_reply",
			blockReply: nil,
			wantNil:    true,
		},
		{
			name:      "true block_reply",
			blockReply: boolPtr(true),
			wantNil:    false,
			wantVal:    true,
		},
		{
			name:      "false block_reply",
			blockReply: boolPtr(false),
			wantNil:    false,
			wantVal:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				ProjectID:          "my-project",
				SubscriptionName:   "my-sub",
				BlockReply:         tt.blockReply,
			}
			ch, err := New(cfg, msgBus, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := ch.BlockReplyEnabled()
			if tt.wantNil && got != nil {
				t.Errorf("expected nil, got %v", *got)
			}
			if !tt.wantNil && (got == nil || *got != tt.wantVal) {
				t.Errorf("expected %v, got %v", tt.wantVal, got)
			}
		})
	}
}

func TestRequireMention_Config(t *testing.T) {
	msgBus := bus.New()

	tests := []struct {
		name     string
		rm       *bool
		wantBool bool
	}{
		{"nil defaults true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GoogleChatConfig{
				ServiceAccountJSON: `{"type":"service_account"}`,
				ProjectID:          "my-project",
				SubscriptionName:   "my-sub",
				RequireMention:     tt.rm,
			}
			ch, err := New(cfg, msgBus, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ch.RequireMention() != tt.wantBool {
				t.Errorf("requireMention = %v, want %v", ch.RequireMention(), tt.wantBool)
			}
		})
	}
}

func TestFactory(t *testing.T) {
	msgBus := bus.New()

	ch, err := Factory("test-gc", json.RawMessage(`{"service_account_json":"{}", "project_id":"p", "subscription_name":"s"}`), json.RawMessage(`{}`), msgBus, nil)
	if err != nil {
		t.Fatalf("factory returned error: %v", err)
	}
	if ch.Name() != "test-gc" {
		t.Errorf("expected name 'test-gc', got %q", ch.Name())
	}
}

func TestFactory_MissingCredentials(t *testing.T) {
	msgBus := bus.New()

	_, err := Factory("test", json.RawMessage(`{}`), json.RawMessage(`{}`), msgBus, nil)
	if err == nil {
		t.Error("expected error for missing credentials")
	}
}

func boolPtr(b bool) *bool { return &b }
