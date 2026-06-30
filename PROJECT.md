# GoClaw Gateway — AI Agent Quick Reference

**Multi-user AI agent gateway** (Go 1.26, v3.x, ~2,100 Go files). PostgreSQL (Standard) / SQLite (Desktop). supermeo/ = custom feature layer, zero-conflict upstream sync from `nextlevelbuilder/goclaw`.

## Architecture: Two-Layer Design

```
internal/          ← UPSTREAM SYNC BOUNDARY — KHÔNG SỬA. Merge sạch từ upstream.
supermeo/          ← CUSTOM LAYER — upstream không có → ZERO conflict. Thêm feature = thêm subpackage.
cmd/               ← Thin wiring (import internal/ + supermeo/). Conflict ~5 dòng.
pkg/protocol/      ← Wire types (public API contract)
migrations/        ← PG schema (sequence ≥ 00100 = no conflict)
```

### supermeo/ Package Map

```
supermeo/
├── userauth/       # JWT + bcrypt + Google OAuth2 + email verify + rate limiter
├── userproviders/  # User-owned LLM API keys (registry + loader)
├── userlimits/     # Per-user quotas (fail-closed enforcement)
└── bridge/         # Adapters: supermeo/ → internal/ (auth, provider 3-tier, context keys)
```

## System Architecture

### Core Pipeline (8-stage)
`context → think → prune → tool → observe → checkpoint → memory → finalize`
- `internal/pipeline/` — agent loop
- `internal/agent/` — router, resolver, input guard

### Provider Resolution (3-Tier)
1. `supermeo/userproviders/` — User-owned API keys
2. `internal/providers/` — Tenant providers (admin-managed)
3. `internal/providers/` — Master defaults (system-wide)
- Supported: Anthropic, OpenAI-compat, DashScope/Qwen, Vertex AI, Claude CLI, ACP

### Gateway
- `internal/gateway/` — WS + HTTP server, method router
- `internal/gateway/methods/` — 58 RPC handlers
- `internal/http/` — HTTP API (`/v1/chat/completions`, `/v1/agents`, files, auth...)
- `pkg/protocol/` — frames (req/res/event), methods, events, errors

### Store Layer
- `internal/store/stores.go` — 47+ store interfaces
- `internal/store/pg/` — PostgreSQL 18 + pgvector (Standard)
- `internal/store/sqlitestore/` — SQLite (Desktop/Lite)
- `migrations/` — PG migrations. New: seq ≥ 00100.
- **Dual-DB:** PG migrations + SQLite `schema.sql` + `schema.go` patches. **Always update both.**

### Security & Isolation
- `internal/permissions/` — RBAC (admin/operator/viewer)
- `internal/crypto/` — AES-256-GCM for API keys + OAuth tokens
- User-as-Tenant: mỗi user → 1 virtual tenant → isolation via scopeClause
- **Tenant-scope guards:** `RoleAdmin` ≠ tenant check. Global tables → `requireMasterScope`. Tenant tables → `WHERE tenant_id = $N`.

## Hard Rules (runtime crash / security if violated)

| Rule | Impact |
|------|--------|
| **i18n:** Add key to `internal/i18n/keys.go` + 3 catalogs (`catalog_{en,vi,zh}.go`) BEFORE handler code. UI: all 3 locale JSONs. | Missing key = runtime crash |
| **Cross-Surface Parity:** Every feature/fix must audit Gateway, API contract, Web UI, CLI. Don't ship backend-only. | Partial feature = broken UX |
| **SQL:** Parameterized queries (`$1, $2`). Check existing indexes. No N+1. | Injection + perf bugs |
| **WS protocol:** First request = `connect`. Params camelCase (`teamId`, not `team_id`). | Connection rejected |
| **Config:** Secrets in `.env.local` or env vars, never in config.json. | Security leak |

## Deploy (Docker — Production)

**⚠️ LUÔN dùng `docker-compose.public.yml`.** Dùng sai sẽ tạo volume rỗng, mất data.

```bash
docker compose -f docker-compose.public.yml up -d --build        # build + start
docker compose -f docker-compose.public.yml up -d --build goclaw  # rebuild app only
docker compose -f docker-compose.public.yml logs -f app           # logs
docker compose -f docker-compose.public.yml --profile maintenance run --rm upgrade  # DB migration
```

| Compose file | Dùng cho | Volume |
|---|---|---|
| `docker-compose.public.yml` | **Production** — self-contained | `goclaw-public_*` (external) |
| `docker-compose.yml` + `.postgres.yml` | Dev only — **KHÔNG dùng production** | `goclaw_*` (rỗng) |
