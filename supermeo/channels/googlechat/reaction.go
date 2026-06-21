package googlechat

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/api/chat/v1"
)

// statusEmoji maps agent status to Google Chat emoji.
var statusEmoji = map[string]string{
	"thinking": "💭",
	"tool":     "🔧",
	"done":     "✅",
	"error":    "❌",
	"stall":    "⏳",
}

// OnReactionEvent adds a status reaction on a user's message.
func (c *Channel) OnReactionEvent(ctx context.Context, chatID string, messageID string, status string) error {
	if c.chatService == nil {
		return fmt.Errorf("googlechat: chat service not initialized")
	}
	if c.cfg.ReactionLevel == "" || c.cfg.ReactionLevel == "off" {
		return nil
	}
	if c.cfg.ReactionLevel == "minimal" && status != "done" && status != "error" {
		return nil
	}

	emoji, ok := statusEmoji[status]
	if !ok {
		return nil
	}

	reaction := &chat.Reaction{
		Emoji: &chat.Emoji{
			Unicode: emoji,
		},
	}

	_, err := c.chatService.Spaces.Messages.Reactions.Create(messageID, reaction).Do()
	if err != nil {
		slog.Debug("googlechat: failed to create reaction", "message_id", messageID, "status", status, "error", err)
		return fmt.Errorf("googlechat create reaction: %w", err)
	}

	return nil
}

// ClearReaction removes all reactions from a message.
func (c *Channel) ClearReaction(ctx context.Context, chatID string, messageID string) error {
	if c.chatService == nil {
		return fmt.Errorf("googlechat: chat service not initialized")
	}
	if c.cfg.ReactionLevel == "" || c.cfg.ReactionLevel == "off" {
		return nil
	}

	resp, err := c.chatService.Spaces.Messages.Reactions.List(messageID).Do()
	if err != nil {
		slog.Debug("googlechat: failed to list reactions for clearing", "message_id", messageID, "error", err)
		return fmt.Errorf("googlechat list reactions: %w", err)
	}

	for _, r := range resp.Reactions {
		if _, err := c.chatService.Spaces.Messages.Reactions.Delete(r.Name).Do(); err != nil {
			slog.Debug("googlechat: failed to delete reaction", "reaction_id", r.Name, "error", err)
		}
	}

	return nil
}
