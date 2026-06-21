package googlechat

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/googleapi"
)

const (
	sendRetryMaxAttempts = 3
	sendRetryBaseDelay   = 500 * time.Millisecond
	sendRetryMaxDelay    = 30 * time.Second
)

// sendMessage is the internal Send implementation.
func (c *Channel) sendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	if c.chatService == nil {
		return fmt.Errorf("googlechat: chat service not initialized")
	}

	chatID := msg.ChatID
	if chatID == "" {
		return fmt.Errorf("googlechat: empty chat ID")
	}

	text := msg.Content
	if text == "" && len(msg.Media) == 0 {
		return nil
	}

	// Format text for Google Chat dialect
	formatted := formatGoogleChat(text)

	// Determine chunk limit
	chunkLimit := c.cfg.TextChunkLimit
	if chunkLimit <= 0 {
		chunkLimit = defaultTextChunkLimit
	}

	// Resolve thread
	threadKey := resolveThreadKey(msg)

	// Send text in chunks
	chunks := chunkText(formatted, chunkLimit)
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		// First chunk: check if we should patch the typing card
		if i == 0 {
			c.patchTypingCard(ctx, chatID, chunk)
		}

		// If stream already delivered a message, patch it instead of creating a duplicate.
		if i == 0 {
			if msgID, ok := inflightStream.LoadAndDelete(chatID); ok {
				if _, err := c._patchMessage(ctx, msgID.(string), &chat.Message{Text: chunk}); err != nil {
					slog.Warn("googlechat: patch stream message failed, creating new", "error", err)
				} else {
					continue
				}
			}
		}

		if _, err := c._createMessage(ctx, chatID, &chat.Message{Text: chunk}, threadKey); err != nil {
			return fmt.Errorf("googlechat send: %w", err)
		}
	}

	return nil
}

// _createMessage posts a new message to a space via the Chat REST API.
func (c *Channel) _createMessage(ctx context.Context, spaceName string, msg *chat.Message, threadKey string) (*chat.Message, error) {
	if msg == nil {
		msg = &chat.Message{}
	}
	if threadKey != "" {
		msg.Thread = &chat.Thread{Name: threadKey}
	}

	call := c.chatService.Spaces.Messages.Create(spaceName, msg)
	if threadKey != "" {
		call = call.MessageReplyOption("REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD")
	}

	return c.retryCreate(ctx, call)
}

// _patchMessage updates an existing message (used for streaming).
func (c *Channel) _patchMessage(ctx context.Context, name string, msg *chat.Message) (*chat.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("googlechat: nil message for patch")
	}
	return c.retryPatch(ctx, name, msg)
}

// resolveThreadKey extracts the thread key from outbound message metadata.
func resolveThreadKey(msg bus.OutboundMessage) string {
	if threadID := msg.Metadata["googlechat_thread_id"]; threadID != "" {
		return threadID
	}
	return ""
}

// chunkText splits text at newline boundaries near maxChars.
func chunkText(text string, maxChars int) []string {
	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > maxChars {
		splitPoint := maxChars
		searchStart := maxChars - maxChars/5
		if searchStart < 0 {
			searchStart = 0
		}
		if idx := strings.LastIndex(remaining[searchStart:maxChars], "\n"); idx >= 0 {
			splitPoint = searchStart + idx + 1
		} else if idx := strings.LastIndex(remaining[searchStart:maxChars], ". "); idx >= 0 {
			splitPoint = searchStart + idx + 2
		} else if idx := strings.LastIndex(remaining[searchStart:maxChars], " "); idx >= 0 {
			splitPoint = searchStart + idx + 1
		}

		chunks = append(chunks, remaining[:splitPoint])
		remaining = remaining[splitPoint:]
	}

	if len(remaining) > 0 {
		chunks = append(chunks, remaining)
	}
	return chunks
}

// retryCreate wraps Create API calls with retry logic.
func (c *Channel) retryCreate(ctx context.Context, call *chat.SpacesMessagesCreateCall) (*chat.Message, error) {
	var lastErr error
	for attempt := 0; attempt < sendRetryMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := computeRetryDelay(attempt)
			slog.Debug("googlechat: retrying create API call", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := call.Do()
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !c.isRetryable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("googlechat: exhausted retries: %w", lastErr)
}

// retryPatch wraps Patch API calls with retry logic.
func (c *Channel) retryPatch(ctx context.Context, name string, msg *chat.Message) (*chat.Message, error) {
	var lastErr error
	for attempt := 0; attempt < sendRetryMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := computeRetryDelay(attempt)
			slog.Debug("googlechat: retrying patch API call", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		call := c.chatService.Spaces.Messages.Patch(name, msg).UpdateMask("text")
		resp, err := call.Do()
		if err == nil {
			return resp, nil
		}

		lastErr = err
		// Special: 404 on PATCH means message was deleted — don't retry
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			return nil, fmt.Errorf("googlechat: message deleted: %w", err)
		}
		if !c.isRetryable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("googlechat: exhausted retries: %w", lastErr)
}

// isRetryable returns true for transient errors (429, 5xx).
func (c *Channel) isRetryable(err error) bool {
	if gerr, ok := err.(*googleapi.Error); ok {
		switch gerr.Code {
		case 403:
			return false // fatal: permission denied
		case 404:
			return false // fatal: not found
		case 429:
			return true // rate limit
		case 500, 502, 503, 504:
			return true // server error
		default:
			return gerr.Code >= 500
		}
	}
	return true // non-API errors are retryable
}

// computeRetryDelay returns jittered exponential backoff delay.
func computeRetryDelay(attempt int) time.Duration {
	delay := sendRetryBaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > sendRetryMaxDelay {
		delay = sendRetryMaxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(delay / 2)))
	return delay + jitter
}
