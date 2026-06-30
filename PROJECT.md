# GoClaw Gateway — AI Agent Quick Reference

**Multi-user AI agent gateway** — Go 1.26, v3.x, ~2,100 Go files, ~4,000 total files.
PostgreSQL 18+pgvector (Standard) / SQLite (Desktop/Lite).
`supermeo/` = custom feature layer, zero-conflict upstream sync from `nextlevelbuilder/goclaw`.

## Architecture Decisions

### Two-Layer Design (merge-safe)

```
internal/    ← UPSTREAM BOUNDARY — KHÔNG SỬA. Merge sạch từ upstream.
supermeo/    ← CUSTOM LAYER — upstream không có → ZERO conflict.
               Thêm feature = thêm subpackage mới. Không sửa internal/.
cmd/         ← Thin wiring (import cả hai). Conflict ~5 dòng.
pkg/protocol/← Wire types (public API contract). Thêm method = thêm const.
migrations/  ← PG schema (seq ≥ 00100 = không đụng upstream).
```

**Why:** Upstream (`nextlevelbuilder/goclaw`) không có `supermeo/` → mọi file trong đó là file mới, không bao giờ conflict khi merge. Bridge pattern (`supermeo/bridge/`) nối supermeo → internal qua adapter, không sửa code cũ.

### supermeo/ Package Map (verified)

```
supermeo/
├── userauth/         JWT(HS256) + bcrypt(cost=12) + Google OAuth2 + email verify + rate limiter
├── userproviders/    User-owned LLM API keys (registry + loader, in-memory map)
├── userlimits/       Per-user quotas (fail-closed)
├── bridge/           Adapters → internal/ (auth, provider 3-tier, context keys)
├── admin/            Provider config, service CRUD, tenant user management, setup wizard
└── channels/googlechat/  Google Chat (GCP SA auth, Pub/Sub inbound, webhook, SSRF-guarded media)
```

### Pipeline (8-stage)
`context → think → prune → tool → observe → checkpoint → memory → finalize`
- `internal/pipeline/` — 21 files: stages, MessageBuffer, run state, message buffer
- `internal/agent/` — router, resolver, input guard, intent classification

### Provider Resolution (3-Tier, fail-through)
1. `supermeo/userproviders/` → User-owned API keys
2. `internal/providers/` → Tenant providers (admin-managed)
3. `internal/providers/` → Master defaults (system-wide)

Adapters: **anthropic, openai** (ChatGPT+Gemini-compat), **dashscope** (Qwen), **codex** (Claude CLI).
Also: Vertex AI, Voyage embeddings, ACP bridge, failover, retry, cooldown, caching middleware, model fallback, reasoning extraction.

## Tech Stack (decision-critical)

| Domain | Choice | Import / Note |
|--------|--------|---------------|
| **WS** | gorilla/websocket v1.5 | `github.com/gorilla/websocket` |
| **CLI** | Cobra v1.10 + Bubble Tea TUI | `github.com/spf13/cobra` |
| **Config** | JSON5 + env vars | `github.com/titanous/json5` |
| **PG driver** | pgx/v5 + sqlx | `github.com/jackc/pgx/v5`, `github.com/jmoiron/sqlx` |
| **SQLite** | modernc.org/sqlite (pure Go) | `modernc.org/sqlite` |
| **Migrations** | golang-migrate v4 | `github.com/golang-migrate/migrate/v4` |
| **Redis** | go-redis v9 | `github.com/redis/go-redis/v9` |
| **JWT** | golang-jwt/jwt/v5 (HS256) | `github.com/golang-jwt/jwt/v5` |
| **Crypto** | bcrypt (cost=12), AES-256-GCM | `golang.org/x/crypto`, `internal/crypto/` |
| **OAuth2** | golang.org/x/oauth2 + Google API | `golang.org/x/oauth2`, `google.golang.org/api` |
| **LLM tokens** | tiktoken-go | `github.com/pkoukk/tiktoken-go` |
| **Browser** | go-rod/rod (CDP) | `github.com/go-rod/rod` |
| **Channels** | discordgo, telego, slack-go, whatsmeow | `github.com/bwmarrin/discordgo`, etc. |
| **Desktop** | Wails v2.12 | `github.com/wailsapp/wails/v2` |
| **MCP** | mark3labs/mcp-go v0.44 | `github.com/mark3labs/mcp-go` |
| **OTel** | opentelemetry v1.43 (OTLP) | `go.opentelemetry.io/otel` |
| **S3** | AWS SDK v2 | `github.com/aws/aws-sdk-go-v2/service/s3` |
| **JS runtime** | goja | `github.com/dop251/goja` |
| **Policy** | CEL (Common Expression Language) | `github.com/google/cel-go` |
| **Keyring** | zalando/go-keyring | `github.com/zalando/go-keyring` |
| **Frontend** | React 19, Vite 8, TypeScript 6, Tailwind 4, Radix UI, Zustand 5, TanStack Query 5, React Router 7, i18next 26, Recharts 3, Sigma 3 (graph viz), Mermaid 11, DnD Kit, Zod 4 | `ui/web/package.json` |
| **Tests** | Go integration/contract/scenario, Vitest 4 | `make test-critical` |

## System Map

### Gateway
- `internal/gateway/` — WS+HTTP server, client lifecycle, method router, rate limiter
- `internal/gateway/methods/` — 42 handler files, ~90 RPC method constants in `pkg/protocol/methods.go`
- `internal/http/` — REST API: `/v1/chat/completions`, `/v1/agents`, files, auth, admin, export/import
- `pkg/protocol/` — frames (`req`/`res`/`event`), ProtocolVersion=3, `MethodConnect="connect"`

### Store Layer (Dual-DB)
- `internal/store/stores.go` — 38 store fields in `Stores` struct (Users, UserSessions, UserProviders added)
- `internal/store/pg/` — PostgreSQL 18 + pgvector (119 files)
- `internal/store/sqlitestore/` — SQLite via modernc.org/sqlite (112 files)
- `internal/store/base/` — Dialect interface, BuildScopeClause, Nil helpers
- **Dual-DB rule:** PG: `migrations/` + bump `RequiredSchemaVersion`. SQLite: update `schema.sql` + add patch in `schema.go` `migrations` map + bump `SchemaVersion`. **Always update both.**
- PG migrations: 87 sequenced pairs, max seq=102. SQLite SchemaVersion=53.

### Memory Architecture (3-Tier, event-driven)

| Tier | Store | Trigger → Worker | Output |
|------|-------|------------------|--------|
| **L0 Working** | SessionStore + MessageBuffer | Compaction → consolidation | Session messages (auto-injected) |
| **L1 Episodic** | EpisodicStore | `session.completed` → episodicWorker | LLM summary + L0 abstract (~50 tokens), 90d TTL |
| **L2 Semantic** | KnowledgeGraphStore + MemoryStore | `episodic.created` → semanticWorker → dedupWorker; + dreamingWorker (debounced) | KG entities/relations; long-term `_system/dreaming/` docs |

Recall scoring: `0.30×frequency + 0.35×relevance + 0.20×recency + 0.15×freshness`.
Workers in `internal/consolidation/` (12 files). Register via `workers.Register()`.

### Channels (verified)
`internal/channels/`: **discord, slack, telegram, whatsapp, facebook, feishu, bitrix24, zalo, pancake**
`supermeo/channels/googlechat/`: **Google Chat** (GCP SA auth, Pub/Sub inbound, webhook)

### Tools (categories)
`internal/tools/` (200+ files): filesystem, shell (deny patterns), web fetch/search (Brave/DDG/Exa/Tavily), browser (Rod), image/video/audio gen, memory, KG, cron, subagent, MCP bridge, sandbox, skill manage/search/publish, sessions, TTS, vault, editor, heartbeat, team tasks, workspace, credentialed exec (git, psql), rate limiter.

### Security & Isolation
- `internal/permissions/` — RBAC: owner/admin/operator/viewer, 5-layer policy engine
- `internal/crypto/` — AES-256-GCM (API keys+OAuth tokens), SHA-256 API key hash (`goclaw_` prefix)
- User-as-Tenant: UserID → TenantID (via `userauth/mapper.go`, concurrency-safe UPSERT), isolation via scopeClause
- **Tenant-scope guards:** `RoleAdmin` ≠ tenant check. Global tables → `requireMasterScope`. Tenant tables → `WHERE tenant_id = $N`. Predicate: `store.IsMasterScope(ctx)`.
- `internal/edition/` — Standard (unlimited) vs Lite (5 agents, 1 team, no channels/RBAC/KG/vector)

### Desktop (Lite Edition)
`//go:build sqliteonly` | Entry: `ui/desktop/main.go` (Wails v2) | Port: 18790 | Secrets: OS keyring + `~/.goclaw/secrets/` | Data: `~/.goclaw/data/` | Auto-update: `internal/updater/` checks `lite-v*` tags | Limits: 5 agents, 1 team, 5 members, 50 sessions.

## Key Numbers (single-glance)

| Metric | Value |
|--------|-------|
| Go version | 1.26 |
| Protocol version | 3 |
| RequiredSchemaVersion | 102 |
| SQLite SchemaVersion | 53 |
| Store interfaces | 38 fields in Stores struct |
| Gateway method handlers | 42 non-test files, ~90 RPC consts |
| i18n keys | 179 (keys.go) + 3 catalogs (en/vi/zh, 371 lines each) |
| PG migrations | 87 pairs, max seq=102 |
| supermeo/ packages | 6 (userauth, userproviders, userlimits, bridge, admin, googlechat) |
| Channels | 10 implemented |
| UI locales | en, vi, zh |

## Hard Rules (runtime crash / security leak if violated)

| Rule | Impact |
|------|--------|
| **i18n:** Add key to `internal/i18n/keys.go` + `catalog_{en,vi,zh}.go` BEFORE handler. UI: all 3 locale JSONs. | Missing key = runtime crash |
| **Surface parity:** Every feature/fix → Gateway server + API contract + Web UI + CLI. | Partial = broken UX |
| **SQL:** Parameterized queries (`$1, $2`). Check indexes. No N+1. | Injection, perf |
| **WS protocol:** First frame = `connect`. Params camelCase (`teamId`, not `team_id`). | Connection rejected |
| **Config:** Secrets in `.env.local` or env vars. Never in config.json. | Security leak |
| **Dual-DB:** PG migration + SQLite schema.sql + schema.go patch. Always both. | Schema drift |
| **Tenant scope:** `RoleAdmin` ≠ tenant check. Use `requireMasterScope` for global tables. | Data leak |
