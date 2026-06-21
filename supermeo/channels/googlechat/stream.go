package googlechat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	chatpb "google.golang.org/api/chat/v1"

	"github.com/nextlevelbuilder/goclaw/internal/channels"
)

const streamFlushInterval = 500 * time.Millisecond

// gcStream implements channels.ChannelStream for Google Chat.
// Buffers incoming text and flushes via a single API call every streamFlushInterval,
// avoiding rate-limit storms from per-chunk API calls.
type gcStream struct {
	mu        sync.Mutex
	channel   *Channel
	chatID    string
	threadKey string
	messageID string // "spaces/{space}/messages/{id}"
	n         int    // dummy message ID counter
	closed    bool
	created   bool

	// Throttle state
	pending    string // accumulated text not yet flushed
	flushed    string // last text sent to the API
	flushCh    chan struct{}
	stopCh     chan struct{}
	flushDone  chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

// Update buffers incoming text for periodic flush.
func (s *gcStream) Update(_ context.Context, text string) {
	s.mu.Lock()
	s.pending = text
	needPoke := s.flushCh != nil
	s.mu.Unlock()
	if needPoke {
		select {
		case s.flushCh <- struct{}{}:
		default:
		}
	}
}

// Stop finalizes the stream: flushes remaining text and shuts down the ticker.
func (s *gcStream) Stop(_ context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	// Final flush of any remaining pending text
	pending := s.pending
	s.mu.Unlock()

	if pending != "" {
		s.doFlush(pending)
	}

	// Shutdown the flush goroutine
	if s.cancel != nil {
		s.cancel()
	}
	if s.flushDone != nil {
		<-s.flushDone
	}
	return nil
}

// startFlushLoop launches the periodic flush goroutine.
func (s *gcStream) startFlushLoop() {
	s.flushCh = make(chan struct{}, 1)
	s.stopCh = make(chan struct{})
	s.flushDone = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	go func() {
		defer close(s.flushDone)
		ticker := time.NewTicker(streamFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			case <-s.flushCh:
				s.mu.Lock()
				pending := s.pending
				s.mu.Unlock()
				if pending != "" && pending != s.flushed {
					s.doFlush(pending)
				}
			case <-ticker.C:
				s.mu.Lock()
				pending := s.pending
				s.mu.Unlock()
				if pending != "" && pending != s.flushed {
					s.doFlush(pending)
				}
			}
		}
	}()
}

// doFlush sends one API call with the accumulated text.
func (s *gcStream) doFlush(text string) {
	if s.channel == nil || s.channel.chatService == nil {
		return
	}

	s.mu.Lock()
	if !s.created {
		// Reuse the inflight typing card to avoid duplicate messages.
		if card, ok := inflightTyping.LoadAndDelete(s.chatID); ok {
			tc := card.(*typingCard)
			s.messageID = tc.messageID
			s.created = true
		}
	}
	created := s.created
	messageID := s.messageID
	s.mu.Unlock()

	if !created {
		msg, err := s.channel._createMessage(s.ctx, s.chatID, &chatpb.Message{Text: text}, s.threadKey)
		if err != nil {
			slog.Warn("googlechat: stream create failed", "chat_id", s.chatID, "error", err)
			return
		}
		s.mu.Lock()
		s.messageID = msg.Name
		s.created = true
		s.flushed = text
		s.mu.Unlock()
		// Store so sendMessage can patch instead of creating a duplicate.
		inflightStream.Store(s.chatID, msg.Name)
	} else {
		if _, err := s.channel._patchMessage(s.ctx, messageID, &chatpb.Message{Text: text}); err != nil {
			slog.Warn("googlechat: stream patch failed", "message_id", messageID, "error", err)
			return
		}
		s.mu.Lock()
		s.flushed = text
		s.mu.Unlock()
	}
}

// MessageID returns a dummy sequence number (Google Chat uses string IDs).
func (s *gcStream) MessageID() int {
	return s.n
}

// StreamEnabled reports whether streaming is enabled.
func (c *Channel) StreamEnabled(isGroup bool) bool {
	if c.cfg.StreamEnabled == nil {
		return true
	}
	return *c.cfg.StreamEnabled
}

// ReasoningStreamEnabled reports whether reasoning should be shown as a separate message.
func (c *Channel) ReasoningStreamEnabled() bool {
	if c.cfg.ReasoningStream != nil {
		return *c.cfg.ReasoningStream
	}
	return true
}

// CreateStream creates a new per-run streaming handle with throttled flush.
func (c *Channel) CreateStream(ctx context.Context, chatID string, firstStream bool) (channels.ChannelStream, error) {
	if c.chatService == nil {
		return nil, fmt.Errorf("googlechat: chat service not initialized for streaming")
	}

	stream := &gcStream{
		channel:   c,
		chatID:    chatID,
	}
	stream.startFlushLoop()
	return stream, nil
}

// FinalizeStream is called after the stream has been stopped.
func (c *Channel) FinalizeStream(ctx context.Context, chatID string, stream channels.ChannelStream) {
	gs, ok := stream.(*gcStream)
	if !ok {
		slog.Warn("googlechat: FinalizeStream received non-gcStream")
		return
	}
	// Stop already handles final flush and cleanup
	_ = gs.Stop(ctx)
}

// ChannelStream interface assertion.
var _ channels.ChannelStream = (*gcStream)(nil)
