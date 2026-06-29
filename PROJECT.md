# GoClaw Gateway — AI Agent Quick Reference

## Target + Vision
**Multi-user AI agent gateway** with user-owned LLM providers. Users bring their own API keys, system provides defaults. Modular architecture enabling seamless upstream sync from `nextlevelbuilder/goclaw`. Desktop (Lite) + Server (Standard) editions.

## Summary
GoClaw is a production-grade agent platform (Go 1.26, v3.x) with ~2,100 Go files. Agents run through an 8-stage pipeline (context→think→prune→tool→observe→checkpoint→memory→finalize), communicate via WebSocket RPC + HTTP API, support 7+ LLM providers, persist to PostgreSQL (Standard) or SQLite (Desktop/Lite). **supermeo/** is the custom feature layer — merges clean from upstream.

## Architecture: Two-Layer Design (Upstream-Safe)

```
github.com/nextlevelbuilder/goclaw/
│
├── internal/          ← UPSTREAM SYNC BOUNDARY (KHÔNG SỬA FILE NÀO)
│   │                    Merge từ nextlevelbuilder/goclaw sạch, không conflict.
│   │                    Chứa: pipeline, providers, gateway, store, channels, tools, ...
│   │
├── supermeo/          ← CUSTOM FEATURE LAYER (toàn bộ code của bạn ở đây)
│   │                    Upstream không có thư mục này → ZERO conflict khi merge.
│   │                    Chứa: userauth, userproviders, userlimits, bridge adapters.
│   │
├── cmd/               ← Thin wiring layer (import cả internal/ và supermeo/)
│   │                    Chỉ thêm import + init, không xóa code cũ. Conflict ~5 dòng.
│   │
├── pkg/protocol/      ← Wire types (public API contract)
├── ui/web/            ← React SPA (public)
└── migrations/        ← DB schema (sequence ≥ 00100 = no conflict)
```

### supermeo/ Directory — The Value Proposition

| Đặc tính | Giá trị |
|----------|--------|
| **Merge sạch từ upstream** | Upstream (`nextlevelbuilder/goclaw`) không có `supermeo/` → mọi file trong đây đều là file mới, không bao giờ conflict khi `git merge upstream/dev` |
| **Module hoá độc lập** | Mỗi subpackage là 1 module riêng: `userauth/`, `userproviders/`, `userlimits/`, `bridge/`. Thêm feature mới = thêm subpackage mới |
| **Không phá vỡ internal/** | `internal/` giữ nguyên 100%. Bridge pattern nối `supermeo/` → `internal/` qua adapter, không sửa code cũ |
| **Tận dụng toàn bộ hạ tầng internal/** | Provider registry, store layer, pipeline, channels, RBAC — tất cả đều được tái sử dụng qua bridge |
| **Mở rộng không giới hạn** | Feature tương lai (billing, team-collab, marketplace, custom tools...) chỉ cần thêm subpackage mới trong `supermeo/` |
| **Go visibility linh hoạt** | Không có `internal/` restriction → package trong `supermeo/` có thể được import bởi plugin hoặc external tooling |

### supermeo/ Package Map

```
supermeo/
├── userauth/          # User identity + authentication
│   ├── model.go       # User, UserSession structs
│   ├── mapper.go      # UserTenantMapper: UserID → TenantID (concurrency-safe UPSERT)
│   ├── jwt.go         # JWT sign/verify (golang-jwt/jwt/v5, HS256)
│   ├── password.go    # bcrypt hash/verify (cost=12)
│   ├── google.go      # Google OAuth2 Sign-In (OpenID Connect)
│   ├── email_verify.go # Email verification via Gmail SMTP
│   ├── smtp.go        # SMTP client (Gmail App Password)
│   └── rate_limiter.go # Auth endpoint brute-force protection
│
├── userproviders/     # User-owned LLM API keys
│   ├── model.go       # UserProviderData struct
│   ├── registry.go    # In-memory map: userID/name → Provider
│   └── loader.go      # Load user providers from DB at startup
│
├── userlimits/        # Per-user resource quotas
│   ├── model.go       # UserQuotaData struct
│   └── policy.go      # CheckLimits() enforcement (fail-closed)
│
└── bridge/            # Adapters: supermeo/ → internal/
    ├── auth_bridge.go      # UserAuthenticator interface → gateway router
    ├── provider_bridge.go  # 3-tier resolution: user → tenant → master
    └── context_bridge.go   # Context keys: goclaw_user_uuid (new UUID key)
```

### Upstream Merge Flow (Đã Verify)

```
git fetch upstream                                    # Lấy code mới từ nextlevelbuilder/goclaw
git merge upstream/dev                                # Merge vào dev branch
# CONFLICT DỰ KIẾN: < 10 file, mỗi file < 5 dòng
#   - internal/gateway/router.go     (Path 0 auth block)
#   - internal/gateway/server.go     (UserAuth field)
#   - internal/store/stores.go       (new store fields)
#   - pkg/protocol/methods.go        (new method consts)
#   - cmd/gateway*.go                (import lines)
# supermeo/** → ZERO conflict
# internal/** (trừ 5 file trên) → ZERO conflict
```

### Adding Agent Presets (Modularized)

To minimize merge conflicts, supermeo-specific agent presets live in a dedicated file:

- **Where:** `ui/web/src/pages/agents/agent-presets-supermeo.ts`
- **Format:** Export an `AgentPreset[]` — each entry has `{ label, prompt, emoji }`
- **Prompts:** Hardcoded in English (no i18n dependency) — keeps locale files untouched and merge surface minimal
- **Wiring:** Import `supermeoPresets` and spread into the array in `agent-presets.ts` (2 lines)
- **Reference:** See existing presets in `agent-presets-supermeo.ts` for structure and tone examples

## System Architecture

### Entry Points
```
cmd/         CLI (Cobra): gateway start, onboard, migrate, TUI, pkg-helper
main.go      Server bootstrap
ui/desktop/  Wails v2 desktop app entry (build tag: sqliteonly)
```

### Core Loop
```
internal/pipeline/    8-stage agent loop (context→think→prune→tool→observe→checkpoint→finalize)
internal/agent/       Agent router, resolver, input guard, types
```

### Providers (LLM + Embedding) — 3-Tier Resolution
```
Tier 1: supermeo/userproviders/     User-owned API keys (user brings own OpenAI/Anthropic key)
Tier 2: internal/providers/         Tenant providers (admin-managed, existing)
Tier 3: internal/providers/         Master defaults (system-wide, existing)
internal/providers/       Anthropic, OpenAI-compat, DashScope/Qwen, Codex, Vertex AI, Claude CLI, ACP
internal/providerresolve/ Plugin adapter registry + model registry
```

### Gateway (WS + HTTP)
```
internal/gateway/          WS + HTTP server, client lifecycle, method router
internal/gateway/methods/  58 RPC handlers (chat, agents, sessions, teams, config, skills, cron...)
internal/http/             HTTP API (/v1/chat/completions, /v1/agents, /v1/skills, files, auth...)
pkg/protocol/              Wire types: frames (req/res/event), methods, events, errors
```

### Store Layer
```
internal/store/stores.go   Interface container (47 stores + new: Users, UserSessions, UserProviders)
internal/store/base/       Dialect interface, BuildMapUpdate, BuildScopeClause, Nil helpers
internal/store/pg/         PostgreSQL 18 + pgvector (Standard)
internal/store/sqlitestore/ SQLite via modernc.org/sqlite (Desktop/Lite)
migrations/                PG migration SQL files (sequence ≥ 00100 cho migration mới)
```

### Memory (3-Tier)
```
L0 Working    → Current session messages (MessageBuffer, auto-injected)
L1 Episodic   → Session summaries via consolidation/episodicWorker (90-day TTL)
L2 Semantic   → Knowledge graph entities via consolidation/semanticWorker
Dreaming      → Long-term fact synthesis from summaries (consolidation/dreamingWorker)
```

### Channels (Existing + Planned)
```
internal/channels/  Telegram, Discord, Slack, WhatsApp, Feishu/Lark, Zalo
internal/channels/googlechat/  NEW: Google Chat (Google Workspace API, HTTP webhook)
internal/channels/gmail/       NEW: Gmail (per-user OAuth2, Pub/Sub watcher)
```

### Tools & Execution
```
internal/tools/       Tool registry, filesystem, exec, web, memory, subagent, MCP bridge
internal/sandbox/     Docker code execution sandbox
internal/skills/      SKILL.md loader + BM25 search
pkg/browser/          Browser automation (Rod + CDP)
```

### Security & Multi-User Isolation
```
internal/permissions/   RBAC (admin/operator/viewer) — tái sử dụng cho user roles
internal/crypto/        AES-256-GCM for API keys + OAuth2 tokens
internal/edition/       Feature gating (Standard vs Lite)
supermeo/userauth/      JWT + bcrypt + Google OAuth2 + email verification
User-as-Tenant mapping  Mỗi user → 1 virtual tenant → tenant isolation tự động qua scopeClause
```

### UI
```
ui/web/         React 19 SPA (pnpm, Vite 6, TypeScript, Tailwind CSS 4, Radix UI, Zustand)
ui/desktop/     Wails v2 desktop app (embedded gateway + React frontend)
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| Database (Standard) | PostgreSQL 18 + pgvector (pgx/v5, database/sql, sqlx) |
| Database (Desktop) | SQLite (modernc.org/sqlite, pure Go) |
| WebSocket | gorilla/websocket |
| CLI | Cobra |
| Config | JSON5 + env vars |
| Auth (new) | golang-jwt/jwt/v5 (HS256), bcrypt, Google OAuth2 (OpenID Connect) |
| Email (new) | Gmail SMTP (App Password, STARTTLS port 587) |
| LLM Token | tiktoken-go |
| Browser | go-rod/rod |
| Desktop | Wails v2 |
| Web UI | React 19, Vite 6, TypeScript, Tailwind CSS 4, Radix UI, Zustand, React Router 7 |
| Channels | telego (Telegram), discordgo, slack-go, whatsmeow |
| Google APIs (new) | Google Chat API, Gmail API, Google OAuth2 |
| MCP | mark3labs/mcp-go |
| Optional | Redis (go-redis), OTel (opentelemetry), Tailscale, goja (JS), CEL |
| Tests | Go integration/scenario/contract tests, vitest (web UI) |

## Current Progress

**Version:** v3.15.0-beta.1 (pre-release on `dev`), v3.4.0 (latest stable)
**Branch:** `develop` (merged from `dev`)
**Upstream:** `nextlevelbuilder/goclaw` — merge sạch nhờ `supermeo/` layer

**Multi-User Plan:** 10 phases, 24 risks audited, 22 deep-dive findings resolved. See `.kilo/plans/multi-user-hybrid-architecture.md`.

**Recent work (Unreleased):**
- Behavior UX sidecar delivery
- Workspace-organizing skill
- Skill agent manage grants
- Packages Update Flow Phase 1 (GitHub) + Phase 2a (pip + npm)

**Feature completeness:** Core stable. Channels, tools, memory, skills, MCP, cron, sandbox, TTS, HTTP API, hooks, teams, delegation, heartbeat all production-ready.

**File count:** ~2,100 Go files, ~4,000 total files, 53 internal packages, 160 PG migrations.
