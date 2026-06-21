package googlechat

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// trustedHosts for media download (SSRF guard).
var trustedHosts = map[string]bool{
	"chat.googleapis.com":    true,
	"googleapis.com":         true,
	"storage.googleapis.com": true,
}

// mediaMaxBytes returns the configured max download size, default 20MB.
func (c *Channel) mediaMaxBytes() int64 {
	if c.cfg.MediaMaxBytes > 0 {
		return c.cfg.MediaMaxBytes
	}
	return defaultMediaMaxBytes
}

// downloadFromURL downloads a file from a trusted URL with SSRF protection.
func (c *Channel) downloadFromURL(ctx context.Context, downloadURL string) ([]byte, string, error) {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return nil, "", fmt.Errorf("googlechat: invalid download URL: %w", err)
	}

	if !isTrustedHost(u.Host) {
		return nil, "", fmt.Errorf("googlechat: untrusted download host: %s", u.Host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("googlechat: download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("googlechat: download returned %d", resp.StatusCode)
	}

	maxBytes := c.mediaMaxBytes()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("googlechat: attachment exceeds max size %d bytes", maxBytes)
	}

	contentType := resp.Header.Get("Content-Type")
	return body, contentType, nil
}

// isTrustedHost checks if a host is in the trusted allowlist.
func isTrustedHost(host string) bool {
	host = strings.ToLower(host)

	if trustedHosts[host] {
		return true
	}

	// Wildcard match (*.example.com)
	parts := strings.SplitN(host, ".", 2)
	if len(parts) == 2 {
		wildcard := "*." + parts[1]
		if trustedHosts[wildcard] {
			return true
		}
	}

	return false
}

// handleInboundAttachment logs and summarizes an inbound attachment.
// Phase 1: text-only attachment notification. Phase 2: native download via Chat API.
func (c *Channel) handleInboundAttachment(ctx context.Context, att *chatAttachment) string {
	if att == nil {
		return ""
	}
	contentType := att.ContentType
	if contentType == "" {
		contentType = "unknown"
	}
	slog.Debug("googlechat: inbound attachment",
		"content_name", att.ContentName,
		"content_type", contentType,
	)
	return fmt.Sprintf("[Attachment: %s (%s)]", att.ContentName, contentType)
}
