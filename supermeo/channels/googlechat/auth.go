// Package googlechat implements the Google Chat channel using Google Cloud Pub/Sub + Chat REST API.
// Supports: DM + Group (spaces), Pub/Sub PULL (primary) + PUSH webhook (optional),
// streaming via message PATCH, emoji reactions, media attachments.
package googlechat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/pubsub/v2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

// redactEmail strips the email portion from error messages to avoid leaking SA identity.
func redactEmail(s string) string {
	// Simple heuristic: replace anything that looks like an email
	for _, prefix := range []string{"service_account_email", "client_email", "email"} {
		if idx := strings.Index(strings.ToLower(s), prefix); idx >= 0 {
			return s[:idx+len(prefix)] + " [redacted]"
		}
	}
	return s
}

// loadCredentials resolves Google service account credentials with 3-tier priority:
// 1. Inline JSON from config/env (ServiceAccountJSON)
// 2. File path from config/env (ServiceAccountFile)
// 3. Application Default Credentials (ADC)
// Credentials are scoped for both Chat API and Pub/Sub (pull mode).
func loadCredentials(serviceAccountJSON, serviceAccountFile string) (*google.Credentials, error) {
	// Combined scopes: Chat API for messaging + Pub/Sub for event delivery (pull mode).
	scopes := []string{chat.ChatBotScope, pubsub.ScopePubSub}

	// Tier 1: Inline JSON
	if serviceAccountJSON != "" {
		creds, err := google.CredentialsFromJSON(context.Background(), []byte(serviceAccountJSON), scopes...)
		if err != nil {
			return nil, fmt.Errorf("googlechat: invalid service_account_json: %w", err)
		}
		return creds, nil
	}

	// Tier 2: File path
	if serviceAccountFile != "" {
		data, err := os.ReadFile(serviceAccountFile)
		if err != nil {
			return nil, fmt.Errorf("googlechat: read service_account_file %s: %w", serviceAccountFile, err)
		}
		creds, err := google.CredentialsFromJSON(context.Background(), data, scopes...)
		if err != nil {
			return nil, fmt.Errorf("googlechat: invalid service_account_file: %w", err)
		}
		return creds, nil
	}

	// Tier 3: ADC
	creds, err := google.FindDefaultCredentials(context.Background(), scopes...)
	if err != nil {
		return nil, fmt.Errorf("googlechat: no credentials found (set service_account_json, service_account_file, or GOOGLE_APPLICATION_CREDENTIALS): %w", err)
	}
	return creds, nil
}

// newChatService creates a Google Chat REST API client.
func newChatService(ctx context.Context, creds *google.Credentials) (*chat.Service, error) {
	svc, err := chat.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("googlechat: create Chat API client: %w", err)
	}
	return svc, nil
}

// newPubSubClient creates a Google Cloud Pub/Sub subscriber client.
func newPubSubClient(ctx context.Context, projectID string, creds *google.Credentials) (*pubsub.Client, error) {
	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("googlechat: create Pub/Sub client (project=%s): %w", projectID, err)
	}
	return client, nil
}
