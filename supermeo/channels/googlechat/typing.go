package googlechat

import (
	"context"
	"log/slog"
	"sync"

	chatpb "google.golang.org/api/chat/v1"
)

// typingCard tracks an in-flight typing indicator message.
type typingCard struct {
	messageID string // "spaces/{space}/messages/{id}"
	spaceName string
	threadKey string
}

// typingMu prevents duplicate typing cards per space.
var typingMu sync.Map // spaceName → *sync.Mutex

// inflightTyping tracks the current typing message per space so we can patch-in-place.
var inflightTyping sync.Map // spaceName → *typingCard

// inflightStream tracks the stream message ID per space so sendMessage can
// patch-in-place instead of creating a duplicate message.
var inflightStream sync.Map // spaceName → string (messageID)

// sendTyping creates or patches a "thinking" indicator message.
// On first call: creates a "💭 Thinking..." message.
// On subsequent calls: patches the existing card if still inflight.
func (c *Channel) sendTyping(ctx context.Context, spaceName, threadKey string) {
	if c.chatService == nil {
		return
	}

	// Guard against duplicate typing cards
	mu := &sync.Mutex{}
	actual, _ := typingMu.LoadOrStore(spaceName, mu)
	mu = actual.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Check if there's already an inflight typing card
	if existing, ok := inflightTyping.Load(spaceName); ok {
		card := existing.(*typingCard)
		// Patch the existing card (still thinking)
		if _, err := c._patchMessage(ctx, card.messageID, &chatpb.Message{Text: "💭 *Thinking...*"}); err != nil {
			slog.Debug("googlechat: patch typing card failed", "error", err)
		}
		return
	}

	// Create a new typing card
	msg, err := c._createMessage(ctx, spaceName, &chatpb.Message{Text: "💭 *Thinking...*"}, threadKey)
	if err != nil {
		slog.Debug("googlechat: create typing card failed", "error", err)
		return
	}

	inflightTyping.Store(spaceName, &typingCard{
		messageID: msg.Name,
		spaceName: spaceName,
		threadKey: threadKey,
	})
}

// clearTyping deletes the typing card and removes it from both inflight maps.
func (c *Channel) clearTyping(ctx context.Context, spaceName string) {
	if card, ok := inflightTyping.LoadAndDelete(spaceName); ok {
		tc := card.(*typingCard)
		if _, err := c.chatService.Spaces.Messages.Delete(tc.messageID).Do(); err != nil {
			slog.Debug("googlechat: delete typing card failed", "message_id", tc.messageID, "error", err)
		}
	}
	inflightStream.Delete(spaceName)
}

// patchTypingCard replaces the typing card content (for first actual chunk, patch-in-place).
func (c *Channel) patchTypingCard(ctx context.Context, spaceName, newText string) {
	if card, ok := inflightTyping.LoadAndDelete(spaceName); ok {
		tc := card.(*typingCard)
		if _, err := c._patchMessage(ctx, tc.messageID, &chatpb.Message{Text: newText}); err != nil {
			slog.Debug("googlechat: patch typing card failed, falling back to create", "error", err)
			// Fallback: create new message
			c._createMessage(ctx, spaceName, &chatpb.Message{Text: newText}, tc.threadKey)
		}
	}
}
