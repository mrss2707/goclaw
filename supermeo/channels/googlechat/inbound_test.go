package googlechat

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseMessagePayload_AddonsEnvelope(t *testing.T) {
	inner := chatMessage{
		Name: "spaces/ABC/messages/xyz",
		Sender: &chatUser{
			Name:        "users/123",
			DisplayName: "Test User",
			Type:        "HUMAN",
		},
		Text: "Hello bot!",
		Space: &chatSpace{
			Name: "spaces/ABC",
			Type: "DM",
		},
	}
	innerJSON, _ := json.Marshal(inner)

	addons := envelopeAddons{
		Type:    "MESSAGE",
		Message: innerJSON,
	}
	addonsJSON, _ := json.Marshal(addons)

	msg := parseMessagePayload(addonsJSON)
	if msg == nil {
		t.Fatal("parseMessagePayload returned nil for Add-ons envelope")
	}
	if msg.Text != "Hello bot!" {
		t.Errorf("expected 'Hello bot!', got %q", msg.Text)
	}
	if msg.Sender.Type != "HUMAN" {
		t.Errorf("expected HUMAN sender, got %s", msg.Sender.Type)
	}
}

func TestParseMessagePayload_ChatAPIEnvelope(t *testing.T) {
	msg := &chatMessage{
		Name: "spaces/ABC/messages/xyz",
		Sender: &chatUser{
			Name:        "users/123",
			DisplayName: "Test User",
			Type:        "HUMAN",
		},
		Text: "Hello from Chat API!",
		Space: &chatSpace{
			Name: "spaces/ABC",
			Type: "ROOM",
		},
	}

	api := envelopeChatAPI{
		Type:    "MESSAGE",
		Message: msg,
	}
	apiJSON, _ := json.Marshal(api)

	parsed := parseMessagePayload(apiJSON)
	if parsed == nil {
		t.Fatal("parseMessagePayload returned nil for Chat API envelope")
	}
	if parsed.Text != "Hello from Chat API!" {
		t.Errorf("expected 'Hello from Chat API!', got %q", parsed.Text)
	}
}

func TestParseMessagePayload_FlatMessage(t *testing.T) {
	msg := chatMessage{
		Name: "spaces/ABC/messages/xyz",
		Sender: &chatUser{
			Name: "users/123",
			Type: "HUMAN",
		},
		Text: "Flat message",
		Space: &chatSpace{
			Name: "spaces/ABC",
			Type: "DM",
		},
	}
	msgJSON, _ := json.Marshal(msg)

	parsed := parseMessagePayload(msgJSON)
	if parsed == nil {
		t.Fatal("parseMessagePayload returned nil for flat message")
	}
	if parsed.Text != "Flat message" {
		t.Errorf("expected 'Flat message', got %q", parsed.Text)
	}
}

func TestParseMessagePayload_InvalidJSON(t *testing.T) {
	msg := parseMessagePayload([]byte("not json"))
	if msg != nil {
		t.Error("parseMessagePayload should return nil for invalid JSON")
	}
}

func TestParseMessagePayload_EmptyData(t *testing.T) {
	msg := parseMessagePayload([]byte("{}"))
	if msg != nil {
		t.Error("parseMessagePayload should return nil for empty JSON")
	}
}

func TestParseMessagePayload_BotSender(t *testing.T) {
	msg := chatMessage{
		Name: "spaces/ABC/messages/xyz",
		Sender: &chatUser{
			Name: "users/bot-123",
			Type: "BOT",
		},
		Text: "Bot message",
		Space: &chatSpace{
			Name: "spaces/ABC",
			Type: "DM",
		},
	}
	msgJSON, _ := json.Marshal(msg)

	parsed := parseMessagePayload(msgJSON)
	if parsed == nil {
		t.Fatal("parseMessagePayload should parse bot messages (self-filter is in dispatchMessage)")
	}
	if parsed.Sender.Type != "BOT" {
		t.Errorf("expected BOT sender, got %s", parsed.Sender.Type)
	}
}

func TestExtractSpaceName(t *testing.T) {
	tests := []struct {
		name string
		msg  *chatMessage
		want string
	}{
		{
			name: "from space field",
			msg: &chatMessage{
				Name: "spaces/ABC/messages/xyz",
				Space: &chatSpace{Name: "spaces/ABC"},
			},
			want: "spaces/ABC",
		},
		{
			name: "fallback from message name",
			msg: &chatMessage{
				Name: "spaces/ABC/messages/xyz",
			},
			want: "spaces/ABC",
		},
		{
			name: "no space info",
			msg:  &chatMessage{Name: "unknown"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSpaceName(tt.msg)
			if got != tt.want {
				t.Errorf("extractSpaceName = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractThreadID(t *testing.T) {
	tests := []struct {
		name string
		msg  *chatMessage
		want string
	}{
		{
			name: "with thread",
			msg: &chatMessage{
				Thread: &chatThread{Name: "spaces/ABC/messages/thread123"},
				Space:  &chatSpace{Name: "spaces/ABC"},
			},
			want: "spaces/ABC/messages/thread123",
		},
		{
			name: "fallback to space",
			msg: &chatMessage{
				Space: &chatSpace{Name: "spaces/ABC"},
			},
			want: "spaces/ABC",
		},
		{
			name: "no space or thread",
			msg:  &chatMessage{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractThreadID(tt.msg)
			if got != tt.want {
				t.Errorf("extractThreadID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDedupCheck(t *testing.T) {
	ch := &Channel{
		dedupMap: make(map[string]time.Time),
	}

	// First call should be true (new)
	if !ch.dedupCheck("msg1") {
		t.Error("dedupCheck should return true for new message")
	}

	// Second call should be false (duplicate)
	if ch.dedupCheck("msg1") {
		t.Error("dedupCheck should return false for duplicate message")
	}

	// Different message should be true
	if !ch.dedupCheck("msg2") {
		t.Error("dedupCheck should return true for different message")
	}
}
