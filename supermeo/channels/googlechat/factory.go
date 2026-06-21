package googlechat

import (
	"encoding/json"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/config"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// gcCreds maps the credentials JSON from the channel_instances table.
type gcCreds struct {
	ServiceAccountJSON string `json:"service_account_json,omitempty"`
	ServiceAccountFile string `json:"service_account_file,omitempty"`
	ProjectID          string `json:"project_id,omitempty"`
	SubscriptionName   string `json:"subscription_name,omitempty"`
}

// gcInstanceConfig maps the non-secret config JSONB from the channel_instances table.
type gcInstanceConfig struct {
	PushEndpointPath string                     `json:"push_endpoint_path,omitempty"`
	AllowFrom        []string                   `json:"allow_from,omitempty"`
	DMPolicy         string                     `json:"dm_policy,omitempty"`
	GroupPolicy      string                     `json:"group_policy,omitempty"`
	RequireMention   *bool                      `json:"require_mention,omitempty"`
	HistoryLimit     int                        `json:"history_limit,omitempty"`
	StreamEnabled    *bool                      `json:"stream_enabled,omitempty"`
	ReasoningStream  *bool                      `json:"reasoning_stream,omitempty"`
	ReactionLevel    string                     `json:"reaction_level,omitempty"`
	TextChunkLimit   int                        `json:"text_chunk_limit,omitempty"`
	BlockReply       *bool                      `json:"block_reply,omitempty"`
	ChatBehavior     *config.ChatBehaviorConfig `json:"chat_behavior,omitempty"`
	MediaMaxBytes    int64                      `json:"media_max_bytes,omitempty"`
	ConnectionMode   string                     `json:"connection_mode,omitempty"`
}

// Factory creates a GoogleChat channel from DB instance data.
func Factory(name string, creds json.RawMessage, cfg json.RawMessage,
	msgBus *bus.MessageBus, pairingSvc store.PairingStore) (channels.Channel, error) {

	var c gcCreds
	if len(creds) > 0 {
		if err := json.Unmarshal(creds, &c); err != nil {
			return nil, fmt.Errorf("decode googlechat credentials: %w", err)
		}
	}
	if c.ServiceAccountJSON == "" && c.ServiceAccountFile == "" {
		return nil, fmt.Errorf("googlechat: service_account_json or service_account_file is required")
	}
	if c.ProjectID == "" {
		return nil, fmt.Errorf("googlechat: project_id is required")
	}

	var ic gcInstanceConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &ic); err != nil {
			return nil, fmt.Errorf("decode googlechat config: %w", err)
		}
	}

	gcCfg := config.GoogleChatConfig{
		Enabled:            true,
		ServiceAccountJSON: c.ServiceAccountJSON,
		ServiceAccountFile: c.ServiceAccountFile,
		ProjectID:          c.ProjectID,
		SubscriptionName:   c.SubscriptionName,
		PushEndpointPath:   ic.PushEndpointPath,
		AllowFrom:          ic.AllowFrom,
		DMPolicy:           ic.DMPolicy,
		GroupPolicy:        ic.GroupPolicy,
		RequireMention:     ic.RequireMention,
		HistoryLimit:       ic.HistoryLimit,
		StreamEnabled:      ic.StreamEnabled,
		ReasoningStream:    ic.ReasoningStream,
		ReactionLevel:      ic.ReactionLevel,
		TextChunkLimit:     ic.TextChunkLimit,
		BlockReply:         ic.BlockReply,
		ChatBehavior:       ic.ChatBehavior,
		MediaMaxBytes:      ic.MediaMaxBytes,
		ConnectionMode:     ic.ConnectionMode,
	}

	if gcCfg.ConnectionMode == "" {
		gcCfg.ConnectionMode = "pull"
	}
	if gcCfg.DMPolicy == "" {
		gcCfg.DMPolicy = "pairing"
	}
	if gcCfg.GroupPolicy == "" {
		gcCfg.GroupPolicy = "open"
	}

	ch, err := New(gcCfg, msgBus, pairingSvc, nil)
	if err != nil {
		return nil, err
	}

	ch.SetName(name)
	return ch, nil
}

// FactoryWithPendingStore returns a ChannelFactory with persistent history support.
func FactoryWithPendingStore(pendingStore store.PendingMessageStore) channels.ChannelFactory {
	return func(name string, creds json.RawMessage, cfg json.RawMessage,
		msgBus *bus.MessageBus, pairingSvc store.PairingStore) (channels.Channel, error) {

		var c gcCreds
		if len(creds) > 0 {
			if err := json.Unmarshal(creds, &c); err != nil {
				return nil, fmt.Errorf("decode googlechat credentials: %w", err)
			}
		}
		if c.ServiceAccountJSON == "" && c.ServiceAccountFile == "" {
			return nil, fmt.Errorf("googlechat: service_account_json or service_account_file is required")
		}
		if c.ProjectID == "" {
			return nil, fmt.Errorf("googlechat: project_id is required")
		}

		var ic gcInstanceConfig
		if len(cfg) > 0 {
			if err := json.Unmarshal(cfg, &ic); err != nil {
				return nil, fmt.Errorf("decode googlechat config: %w", err)
			}
		}

		gcCfg := config.GoogleChatConfig{
			Enabled:            true,
			ServiceAccountJSON: c.ServiceAccountJSON,
			ServiceAccountFile: c.ServiceAccountFile,
			ProjectID:          c.ProjectID,
			SubscriptionName:   c.SubscriptionName,
			PushEndpointPath:   ic.PushEndpointPath,
			AllowFrom:          ic.AllowFrom,
			DMPolicy:           ic.DMPolicy,
			GroupPolicy:        ic.GroupPolicy,
			RequireMention:     ic.RequireMention,
			HistoryLimit:       ic.HistoryLimit,
			StreamEnabled:      ic.StreamEnabled,
			ReasoningStream:    ic.ReasoningStream,
			ReactionLevel:      ic.ReactionLevel,
			TextChunkLimit:     ic.TextChunkLimit,
			BlockReply:         ic.BlockReply,
			ChatBehavior:       ic.ChatBehavior,
			MediaMaxBytes:      ic.MediaMaxBytes,
			ConnectionMode:     ic.ConnectionMode,
		}

		if gcCfg.ConnectionMode == "" {
			gcCfg.ConnectionMode = "pull"
		}
		if gcCfg.DMPolicy == "" {
			gcCfg.DMPolicy = "pairing"
		}
		if gcCfg.GroupPolicy == "" {
			gcCfg.GroupPolicy = "open"
		}

		ch, err := New(gcCfg, msgBus, pairingSvc, pendingStore)
		if err != nil {
			return nil, err
		}

		ch.SetName(name)
		return ch, nil
	}
}
