package googlechat

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/pubsub/v2"

	chatpb "google.golang.org/api/chat/v1"

	"github.com/nextlevelbuilder/goclaw/internal/channels"
)

// envelopeAddons is the legacy Google Workspace Add-ons event wrapper.
type envelopeAddons struct {
	Type        string          `json:"type"`
	EventTime   string          `json:"eventTime"`
	Message     json.RawMessage `json:"message"`
	Space       json.RawMessage `json:"space"`
	User        json.RawMessage `json:"user"`
	ConfigToken string          `json:"configToken"`
}

// envelopeChatAPI is the native Google Chat API event wrapper.
type envelopeChatAPI struct {
	Type        string          `json:"type"`
	EventTime   string          `json:"eventTime"`
	Message     *chatMessage    `json:"message,omitempty"`
	Space       *chatSpace      `json:"space,omitempty"`
	User        *chatUser       `json:"user,omitempty"`
	ConfigToken string          `json:"configToken,omitempty"`
	Common      json.RawMessage `json:"common,omitempty"`
}

// chatMessage is a message received from Google Chat.
type chatMessage struct {
	Name         string           `json:"name"`
	Sender       *chatUser        `json:"sender"`
	CreateTime   string           `json:"createTime"`
	Text         string           `json:"text"`
	Thread       *chatThread      `json:"thread,omitempty"`
	Space        *chatSpace       `json:"space,omitempty"`
	ArgumentText string           `json:"argumentText"`
	Attachment   []chatAttachment `json:"attachment,omitempty"`
	SlashCommand *chatSlashCommand `json:"slashCommand,omitempty"`
}

// chatSpace represents a Google Chat space.
type chatSpace struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "DM" or "ROOM"
	DisplayName string `json:"displayName"`
}

// chatUser represents a Google Chat user.
type chatUser struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"` // "HUMAN" or "BOT"
	Email       string `json:"email,omitempty"`
}

// chatThread represents a message thread.
type chatThread struct {
	Name string `json:"name"`
}

// chatAttachment represents a message attachment.
type chatAttachment struct {
	Name              string                `json:"name"`
	ContentName       string                `json:"contentName"`
	ContentType       string                `json:"contentType"`
	AttachmentDataRef *chatAttachmentDataRef `json:"attachmentDataRef,omitempty"`
	DriveDataRef      *chatDriveDataRef      `json:"driveDataRef,omitempty"`
}

// chatAttachmentDataRef is a reference to uploaded attachment data.
type chatAttachmentDataRef struct {
	ResourceName string `json:"resourceName"`
}

// chatDriveDataRef is a reference to a Google Drive file.
type chatDriveDataRef struct {
	DriveFileID string `json:"driveFileId"`
}

// chatSlashCommand represents a slash command invocation.
type chatSlashCommand struct {
	CommandID int64 `json:"commandId"`
}

// parseMessagePayload tries 3 envelope formats: Add-ons, native Chat API, relay flat.
// Returns nil if none match.
func parseMessagePayload(data []byte) *chatMessage {
	// Format 1: Google Workspace Add-ons envelope
	var addons envelopeAddons
	if err := json.Unmarshal(data, &addons); err == nil && addons.Type != "" && addons.Message != nil {
		var msg chatMessage
		if err := json.Unmarshal(addons.Message, &msg); err == nil && msg.Name != "" {
			return &msg
		}
	}

	// Format 2: Native Chat API event envelope
	var api envelopeChatAPI
	if err := json.Unmarshal(data, &api); err == nil && api.Type != "" && api.Message != nil {
		return api.Message
	}

	// Format 3: Flat chatMessage (relayed or test payload)
	var msg chatMessage
	if err := json.Unmarshal(data, &msg); err == nil && msg.Name != "" {
		return &msg
	}

	return nil
}

// extractThreadID extracts the thread ID from a message, falling back to space name.
func extractThreadID(msg *chatMessage) string {
	if msg.Thread != nil && msg.Thread.Name != "" {
		return msg.Thread.Name
	}
	if msg.Space != nil {
		return msg.Space.Name
	}
	return ""
}

// extractSpaceName extracts the space resource name, e.g. "spaces/ABCDE".
func extractSpaceName(msg *chatMessage) string {
	if msg.Space != nil {
		return msg.Space.Name
	}
	// Fallback: parse from message name "spaces/{space}/messages/{id}"
	name := msg.Name
	if i := len("spaces/"); i < len(name) {
		if end := indexSlash(name, i); end > i {
			return "spaces/" + name[i:end]
		}
	}
	return ""
}

func indexSlash(s string, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// handlePubSubMessage processes a Pub/Sub message from the pull subscription.
func (c *Channel) handlePubSubMessage(ctx context.Context, msg *pubsub.Message) {
	msgID := msg.ID

	// Dedup guard: Pub/Sub is at-least-once delivery
	if !c.dedupCheck(msgID) {
		slog.Debug("googlechat: duplicate message skipped", "message_id", msgID)
		return
	}

	// Parse the data payload
	data := msg.Data
	if len(data) == 0 {
		// Try attributes for relayed messages
		if attrData := msg.Attributes["data"]; attrData != "" {
			data = []byte(attrData)
		}
	}

	chatMsg := parseMessagePayload(data)
	if chatMsg == nil {
		slog.Warn("googlechat: unknown or empty payload", "message_id", msgID)
		return
	}

	c.dispatchMessage(ctx, chatMsg)
}

// dispatchMessage processes a parsed chatMessage and forwards it to the agent.
func (c *Channel) dispatchMessage(ctx context.Context, msg *chatMessage) {
	if msg == nil {
		return
	}

	// Bot self-filter: drop messages from BOT type senders
	if msg.Sender != nil && strings.EqualFold(msg.Sender.Type, "BOT") {
		slog.Debug("googlechat: ignoring bot message", "sender", msg.Sender.Name)
		return
	}

	// Extract identity
	senderID := ""
	senderName := ""
	if msg.Sender != nil {
		senderID = msg.Sender.Name // "users/1234567890"
		senderName = msg.Sender.DisplayName
	}
	if senderID == "" {
		slog.Debug("googlechat: message without sender, skipping")
		return
	}

	// Extract space and thread
	spaceName := extractSpaceName(msg)
	if spaceName == "" {
		slog.Debug("googlechat: message without space, skipping")
		return
	}

	threadKey := extractThreadID(msg)

	// Determine peer kind
	peerKind := "group"
	if msg.Space != nil && strings.EqualFold(msg.Space.Type, "DM") {
		peerKind = "direct"
	}

	// Check DM/Group policy (bypassed for /pair command)
	isPairCmd := strings.TrimSpace(msg.Text) == "/pair" || strings.TrimSpace(msg.ArgumentText) == "/pair"
	if !isPairCmd {
	if peerKind == "direct" {
		result := c.CheckDMPolicy(ctx, senderID, c.cfg.DMPolicy)
		switch result {
		case channels.PolicyDeny:
			return
		case channels.PolicyNeedsPairing:
			c.handlePairingRequest(ctx, spaceName, senderID)
			return
		}
	} else {
		result := c.CheckGroupPolicy(ctx, senderID, spaceName, c.cfg.GroupPolicy)
		switch result {
		case channels.PolicyDeny:
			return
		case channels.PolicyNeedsPairing:
			c.handlePairingRequest(ctx, spaceName, senderID)
			return
		}
	}
	}

	// Extract content
	content := msg.Text
	if content == "" && msg.ArgumentText != "" {
		content = msg.ArgumentText
	}

	// Handle attachments
	var mediaPaths []string
	for _, att := range msg.Attachment {
		if attText := c.handleInboundAttachment(ctx, &att); attText != "" {
			if content == "" {
				content = attText
			} else {
				content += "\n" + attText
			}
		}
	}

	// Build metadata
	metadata := map[string]string{
		"googlechat_space":        spaceName,
		"googlechat_sender_name":  senderName,
	}
	if threadKey != "" {
		metadata["googlechat_thread_id"] = threadKey
	}

	// Add merged sender ID (name|displayName format for compound matching)
	compoundSender := senderID
	if senderName != "" {
		compoundSender = senderID + "|" + senderName
	}

	c.HandleMessage(compoundSender, spaceName, content, mediaPaths, metadata, peerKind)
}

// handlePairingRequest sends a pairing code prompt to an unauthenticated sender.
func (c *Channel) handlePairingRequest(ctx context.Context, spaceName, senderID string) {
	if !c.CanSendPairingNotif(senderID, 60*time.Second) {
		return
	}

	text := "🔒 *Access Required*\n\n"
	text += "This bot requires pairing before use. Please use the `/pair` command in a direct message or contact the administrator."
	text += "\n\nYour sender ID: `" + senderID + "`"

	if c.chatService != nil {
		if _, err := c._createMessage(ctx, spaceName, &chatpb.Message{Text: text}, ""); err != nil {
			slog.Warn("googlechat: failed to send pairing message", "space", spaceName, "error", err)
		}
	}

	c.MarkPairingNotifSent(senderID)
}

// dedupCheck returns true if the message ID is new (not seen recently).
func (c *Channel) dedupCheck(msgID string) bool {
	c.dedupMu.Lock()
	defer c.dedupMu.Unlock()

	// Cleanup old entries periodically
	if len(c.dedupMap) > dedupWindowSize {
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, t := range c.dedupMap {
			if t.Before(cutoff) {
				delete(c.dedupMap, k)
			}
		}
	}

	if _, seen := c.dedupMap[msgID]; seen {
		return false
	}
	c.dedupMap[msgID] = time.Now()
	return true
}
