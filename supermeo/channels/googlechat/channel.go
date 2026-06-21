package googlechat

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub/v2"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/config"
	"github.com/nextlevelbuilder/goclaw/internal/safego"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"google.golang.org/api/chat/v1"
)

const (
	defaultTextChunkLimit   = 4000
	defaultMediaMaxBytes    = 20 * 1024 * 1024 // 20MB
	defaultPushEndpointPath = "/googlechat/events"
	pullReconnectMaxRetries = 10
	pullReconnectBaseDelay  = 1 * time.Second
	pullReconnectMaxDelay   = 5 * time.Minute
	dedupWindowSize         = 1000
)

// Channel connects to Google Chat via Google Cloud Pub/Sub + Chat REST API.
type Channel struct {
	*channels.BaseChannel
	cfg         config.GoogleChatConfig
	chatService *chat.Service
	pubsub      *pubsub.Client
	sub         *pubsub.Subscriber
	cancel      context.CancelFunc
	stopCh      chan struct{}
	httpServer  *http.Server

	// dedup guards against at-least-once Pub/Sub delivery.
	dedupMu  sync.Mutex
	dedupMap map[string]time.Time
}

// Option configures optional Channel dependencies.
type Option func(*Channel)

// New creates a new Google Chat channel.
func New(cfg config.GoogleChatConfig, msgBus *bus.MessageBus, pairingSvc store.PairingStore, pendingStore store.PendingMessageStore, opts ...Option) (*Channel, error) {
	if cfg.ServiceAccountJSON == "" && cfg.ServiceAccountFile == "" {
		return nil, fmt.Errorf("googlechat: service_account_json or service_account_file is required")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("googlechat: project_id is required")
	}
	if cfg.SubscriptionName == "" && cfg.ConnectionMode != "push" {
		return nil, fmt.Errorf("googlechat: subscription_name is required for pull mode")
	}

	if cfg.ConnectionMode == "" {
		cfg.ConnectionMode = "pull"
	}

	base := channels.NewBaseChannel(channels.TypeGoogleChat, msgBus, cfg.AllowFrom)
	base.ValidatePolicy(cfg.DMPolicy, cfg.GroupPolicy)

	historyLimit := cfg.HistoryLimit
	if historyLimit == 0 {
		historyLimit = channels.DefaultGroupHistoryLimit
	}

	requireMention := true
	if cfg.RequireMention != nil {
		requireMention = *cfg.RequireMention
	}

	ch := &Channel{
		BaseChannel: base,
		cfg:         cfg,
		stopCh:      make(chan struct{}),
		dedupMap:    make(map[string]time.Time),
	}
	ch.SetPairingService(pairingSvc)
	ch.SetGroupHistory(channels.MakeHistory(channels.TypeGoogleChat, pendingStore, base.TenantID()))
	ch.SetHistoryLimit(historyLimit)
	ch.SetRequireMention(requireMention)
	for _, opt := range opts {
		opt(ch)
	}
	return ch, nil
}

// Name returns the channel instance name.
func (c *Channel) Name() string { return c.BaseChannel.Name() }

// Type returns the platform type.
func (c *Channel) Type() string { return channels.TypeGoogleChat }

// IsRunning reports whether the channel is active.
func (c *Channel) IsRunning() bool { return c.BaseChannel.IsRunning() }

// IsAllowed checks sender against the allowlist.
func (c *Channel) IsAllowed(senderID string) bool { return c.BaseChannel.IsAllowed(senderID) }

// BlockReplyEnabled returns the per-channel block_reply override.
func (c *Channel) BlockReplyEnabled() *bool { return c.cfg.BlockReply }

// ChatBehaviorConfig returns the per-channel chat_behavior override.
func (c *Channel) ChatBehaviorConfig() *config.ChatBehaviorConfig { return c.cfg.ChatBehavior }

// SetPendingHistoryTenantID propagates tenant_id to pending history for multi-tenant DB operations.
func (c *Channel) SetPendingHistoryTenantID(id uuid.UUID) {
	if gh := c.GroupHistory(); gh != nil {
		gh.SetTenantID(id)
	}
}

// Start begins listening for Google Chat events via Pub/Sub (pull or push mode).
func (c *Channel) Start(ctx context.Context) error {
	c.GroupHistory().StartFlusher()
	slog.Info("starting googlechat bot", "mode", c.cfg.ConnectionMode)

	// Load credentials and initialize API clients.
	creds, err := loadCredentials(c.cfg.ServiceAccountJSON, c.cfg.ServiceAccountFile)
	if err != nil {
		return fmt.Errorf("googlechat start: %w", err)
	}

	chatSvc, err := newChatService(ctx, creds)
	if err != nil {
		return fmt.Errorf("googlechat start: %w", err)
	}
	c.chatService = chatSvc

	switch c.cfg.ConnectionMode {
	case "push":
		return c.startWebhook(ctx)
	default: // "pull"
		psClient, err := newPubSubClient(ctx, c.cfg.ProjectID, creds)
		if err != nil {
			return fmt.Errorf("googlechat start: %w", err)
		}
		c.pubsub = psClient
		c.sub = psClient.Subscriber(c.cfg.SubscriptionName)

		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel
		c.SetRunning(true)

		go func() {
			defer safego.Recover(nil, "component", "googlechat_pull", "channel", c.Name())
			c.pullLoop(ctx)
		}()
		return nil
	}
}

// Stop shuts down the Google Chat channel.
func (c *Channel) Stop(_ context.Context) error {
	c.GroupHistory().StopFlusher()
	slog.Info("stopping googlechat bot")
	close(c.stopCh)

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	if c.httpServer != nil {
		c.httpServer.Close()
		c.httpServer = nil
	}

	c.sub = nil
	c.pubsub = nil
	c.chatService = nil

	c.SetRunning(false)
	return nil
}

// Send delivers an outbound message to a Google Chat space.
func (c *Channel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("googlechat bot not running")
	}
	return c.sendMessage(ctx, msg)
}

// pullLoop runs the Pub/Sub pull subscriber with exponential-backoff reconnect.
func (c *Channel) pullLoop(ctx context.Context) {
	for attempt := 0; attempt < pullReconnectMaxRetries; attempt++ {
		if attempt > 0 {
			delay := pullReconnectBaseDelay * time.Duration(min(1<<uint(attempt-1), int(pullReconnectMaxDelay/pullReconnectBaseDelay)))
			slog.Info("googlechat: reconnecting pull subscriber", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		if c.sub == nil {
			return
		}
		err := c.sub.Receive(ctx, func(recvCtx context.Context, msg *pubsub.Message) {
			c.handlePubSubMessage(recvCtx, msg)
			msg.Ack()
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("googlechat: pull subscriber error, will retry", "error", err)
		}
	}
	slog.Error("googlechat: pull subscriber exhausted retries", "maxRetries", pullReconnectMaxRetries)
}

// Interface assertions.
var (
	_ channels.Channel             = (*Channel)(nil)
	_ channels.WebhookChannel      = (*Channel)(nil)
	_ channels.StreamingChannel    = (*Channel)(nil)
	_ channels.ReactionChannel     = (*Channel)(nil)
	_ channels.BlockReplyChannel   = (*Channel)(nil)
	_ channels.ChatBehaviorChannel = (*Channel)(nil)
)
