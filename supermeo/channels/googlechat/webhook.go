package googlechat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// webhookHandler serves the PUSH webhook endpoint for Pub/Sub push subscriptions.
type webhookHandler struct {
	channel *Channel
}

// ServeHTTP handles Pub/Sub push delivery.
func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB cap
	if err != nil {
		slog.Warn("googlechat webhook: read body failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Pub/Sub push body (PushConfig wraps message in JSON)
	var pushMsg struct {
		Message struct {
			Data       string            `json:"data"`
			MessageID  string            `json:"messageId"`
			Attributes map[string]string `json:"attributes"`
		} `json:"message"`
		Subscription string `json:"subscription"`
	}
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		slog.Warn("googlechat webhook: parse push body failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(pushMsg.Message.Data)
	if err != nil {
		slog.Warn("googlechat webhook: base64 decode failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Parse and dispatch the message
	msg := parseMessagePayload(data)
	if msg == nil {
		slog.Debug("googlechat webhook: unknown payload format", "message_id", pushMsg.Message.MessageID)
		w.WriteHeader(http.StatusOK)
		return
	}

	h.channel.dispatchMessage(r.Context(), msg)
	w.WriteHeader(http.StatusOK)
}

// WebhookHandler returns the HTTP handler and mount path for the PUSH webhook endpoint.
// Returns ("", nil) when connection_mode is "pull" (default).
func (c *Channel) WebhookHandler() (string, http.Handler) {
	if c.cfg.ConnectionMode != "push" {
		return "", nil
	}

	path := c.cfg.PushEndpointPath
	if path == "" {
		path = defaultPushEndpointPath
	}

	handler := &webhookHandler{channel: c}
	return path, handler
}

// startWebhook starts the webhook server (for PUSH mode).
func (c *Channel) startWebhook(ctx context.Context) error {
	_, handler := c.WebhookHandler()
	if handler == nil {
		return fmt.Errorf("googlechat: push mode requires webhook path")
	}

	slog.Info("googlechat: webhook endpoint ready for mounting")
	c.SetRunning(true)
	return nil
}
