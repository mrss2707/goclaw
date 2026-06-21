# Google Chat Channel Configuration

## Architecture

```
Google Chat UI → Pub/Sub Topic → Subscription → GoClaw (pull) → Agent
                   ↑                                  ↓
            Chat API publish              Chat API create/patch message
```

GoClaw uses **Pub/Sub pull mode**: messages are published by Google Chat to a Pub/Sub topic, and GoClaw pulls them from a subscription. Responses are sent back via the Chat REST API.

## Two Configuration Paths

| | DB Instance (recommended) | Config-based (`config.json`) |
|---|---|---|
| **Management** | Web UI (`/channels`) | Edit `config.json` directly |
| **Credentials** | Encrypted in DB `credentials` column | Env var `GOCLAW_GOOGLECHAT_SERVICE_ACCOUNT` |
| **Config** | JSONB `config` column per instance | `config.json` → `channels.google_chat` |
| **Multi-instance** | Yes (multiple channels per type) | No (one per channel type) |
| **Reload** | Hot-reload via Web UI | Requires restart |
| **Use case** | Production, multi-tenant | Development, quick testing |

> **Always use DB Instance in production.** Config-based is a legacy path kept for backward compatibility. When a DB instance exists for a channel type, the config-based channel for that type is automatically skipped (see `cmd/gateway_channels_setup.go` — `dbLoaded` guard).

---

## Prerequisites

- Google Workspace domain (e.g. `example.com`)
- `gcloud` CLI installed and authenticated (`gcloud auth login`)
- Billing enabled on GCP project
- GoClaw running with PostgreSQL

---

## Step 1: GCP Project Setup

```bash
# Set your project ID (must be globally unique)
PROJECT_ID="your-project-id"

# Create project
gcloud projects create $PROJECT_ID --name="Your Project Name"

# Set as default
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable chat.googleapis.com pubsub.googleapis.com --project=$PROJECT_ID
```

> **Note**: If creating via GCP Console instead of gcloud, GCP auto-appends a suffix to the project ID (e.g. `your-project-500107`). Use the actual project ID shown in the Console.

## Step 2: Link Billing Account

GCP Console → **Billing** → select billing account → **My Projects** → add the project.

## Step 3: Service Account + IAM

```bash
SA_NAME="your-bot-sa"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Create service account
gcloud iam service-accounts create $SA_NAME \
  --display-name="Chat Bot" \
  --project=$PROJECT_ID

# Grant Pub/Sub subscriber role
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/pubsub.subscriber"

# Download key (store securely!)
gcloud iam service-accounts keys create ~/chat-bot-sa.json \
  --iam-account=${SA_EMAIL} \
  --project=$PROJECT_ID
```

## Step 4: Pub/Sub Topic + Subscription

```bash
# Create topic
gcloud pubsub topics create chat-events --project=$PROJECT_ID

# Create subscription (pull mode)
gcloud pubsub subscriptions create chat-events-sub \
  --topic=chat-events \
  --project=$PROJECT_ID \
  --ack-deadline=60 \
  --message-retention-duration=1d
```

## Step 5: Grant Chat API Publisher on Topic

```bash
gcloud pubsub topics add-iam-policy-binding chat-events \
  --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
  --role="roles/pubsub.publisher" \
  --project=$PROJECT_ID
```

> This allows Google Chat to publish events to your topic. The `chat-api-push@system.gserviceaccount.com` is a Google-managed service account — NOT your own SA.

## Step 6: Google Chat API Console Configuration

> **CRITICAL**: Do NOT check "Build this Chat app as a Workspace add-on" — it locks the configuration permanently and prevents Pub/Sub connection.

1. Open: `https://console.cloud.google.com/apis/api/chat.googleapis.com/hangouts-chat?project=$PROJECT_ID`
2. Go to tab **Configuration**
3. Fill in:
   - **App name**: `Your Bot Name`
   - **Avatar URL**: `https://www.gstatic.com/chat/avatar/placeholder_bot_128.png`
   - **Description**: `AI Agent Gateway`
4. **Connection settings**:
   - Select: **Cloud Pub/Sub**
   - **Topic name**: `projects/$PROJECT_ID/topics/chat-events`
5. **Visibility**: Check "Make this Chat app available to specific people and groups"
   - Add your email
6. **App Status**: Set to **LIVE** (button at top of page)
7. **Save** → wait 5-10 minutes for propagation

### Verify Bot Appears in Google Chat

1. Open https://chat.google.com
2. **New Chat** → search your bot name
3. If not found after 10 minutes, try direct link:
   ```
   https://chat.google.com/u/0/dm/${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com
   ```

## Step 7: GoClaw DB Instance (Web UI)

1. Go to http://localhost:5173/channels → **Add Channel**
2. Select type: **Google Chat**
3. **Credentials tab**:
   - **Service Account JSON**: Paste the entire contents of `~/chat-bot-sa.json`
   - **GCP Project ID**: `$PROJECT_ID`
   - **Pub/Sub Subscription**: `chat-events-sub`
4. **Config tab**:
   - **Connection Mode**: `Pull (Pub/Sub)`
   - **DM Policy**: `Open` (or `Pairing` if you want pairing flow)
   - **Group Policy**: `Open`
   - **Require Mention**: `false` (DM has no @mention concept)
   - **Stream Enabled**: `true`
   - **Reasoning Stream**: `false` (avoids slow character-by-character display)
   - **History Limit**: `50`
5. **Allow From** (if DM Policy is not `Open`): Add your Google Chat sender ID
   - Find it by sending a message to the bot — it will reply with your sender ID
   - Format: `users/123456789012345678901`
6. Click **Create** → GoClaw auto-reloads the instance

## Step 8: Verify

```bash
# Check APIs enabled
gcloud services list --project=$PROJECT_ID --enabled | grep -E "chat|pubsub"

# Test Pub/Sub round-trip (publish + check subscription)
gcloud pubsub topics publish chat-events \
  --message='{"test":true}' \
  --project=$PROJECT_ID

gcloud pubsub subscriptions pull chat-events-sub \
  --project=$PROJECT_ID --limit=1 --auto-ack

# Check GoClaw log (should show channels=[your-instance-name])
grep "channels=\[" /tmp/goclaw.log
```

## Current Working Config (Reference)

### DB Instance Config (JSONB)

```json
{
  "dm_policy": "open",
  "group_policy": "open",
  "require_mention": false,
  "dm_stream": true,
  "stream_enabled": true,
  "reasoning_stream": false,
  "reasoning_delivery": "off",
  "connection_mode": "pull",
  "history_limit": 50,
  "reaction_level": "full",
  "draft_transport": true,
  "mention_mode": "strict",
  "media_max_mb": 20,
  "link_preview": true,
  "allow_from": [
    "user-uuid-here",
    "users/123456789012345678901"
  ]
}
```

### Pub/Sub Resources

| Resource | Name |
|----------|------|
| Topic | `projects/$PROJECT_ID/topics/chat-events` |
| Subscription | `projects/$PROJECT_ID/subscriptions/chat-events-sub` |
| Ack Deadline | 60s |
| Message Retention | 1 day |

---

## Troubleshooting

### Bot not appearing in Google Chat

- Verify App Status is **LIVE** in Console
- Verify your email is in the visibility allowlist
- Wait up to 15 minutes for propagation
- Verify Chat API is enabled: `gcloud services list --enabled | grep chat`

### Messages not reaching GoClaw

- **Check**: `gcloud pubsub subscriptions pull chat-events-sub --limit=5`
  - If messages appear here but not in GoClaw logs → GoClaw pull loop issue
  - If no messages → Google Chat not publishing to topic

- **Verify topic IAM**: `gcloud pubsub topics get-iam-policy chat-events`
  - Must have `chat-api-push@system.gserviceaccount.com` with `roles/pubsub.publisher`

- **Verify Console**: Connection settings → topic name must be exact: `projects/$PROJECT_ID/topics/chat-events`

- **Enable debug logging**: Set `GOCLAW_LOG_LEVEL=debug` env var, restart GoClaw

### "This bot requires pairing" message

- This means `dm_policy` is `"pairing"` and the sender is not paired
- Send `/pair` in DM to the bot to initiate pairing
- Or change `dm_policy` to `"open"` in the DB instance config

### `/pair` command not working

- Fixed in code: `/pair` now bypasses DM policy check and reaches the pairing engine
- Requires the DB instance to have its agent properly linked and active

### Slow character-by-character output (streaming)

- Root cause: Each LLM text chunk triggers a `_patchMessage` API call, causing rate-limit storms
- Fix: Throttled streaming implemented — buffers chunks and flushes every 500ms
- Config: Keep `stream_enabled: true` and `reasoning_stream: false`

### Duplicate messages

- Root cause: Stream creates a message, then `sendMessage` creates another for the final reply
- Fix: `sendMessage` checks `inflightStream` map and patches the existing stream message instead

### "no provider available" / agent not responding

- Verify the agent linked to the channel has an API key configured
- Check GoClaw log for `resolved agent from DB` — confirms agent loaded successfully

---

## gcloud Quick Reference

```bash
# Project
gcloud projects list
gcloud config set project PROJECT_ID

# Pub/Sub
gcloud pubsub topics list
gcloud pubsub subscriptions list
gcloud pubsub topics publish TOPIC --message='{"key":"value"}'
gcloud pubsub subscriptions pull SUB --limit=5 --auto-ack

# Service Accounts
gcloud iam service-accounts list
gcloud iam service-accounts keys list --iam-account=SA_EMAIL

# APIs
gcloud services list --enabled
gcloud services enable API_NAME
```
