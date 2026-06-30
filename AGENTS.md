# GoClaw Gateway

PostgreSQL multi-tenant AI agent gateway with WebSocket RPC + HTTP API.

> **ARCHITECTURE:** See `PROJECT.md` for system architecture, supermeo/ layer, package map, tech stack, memory tiers, provider resolution, store layer, channels, tools, key numbers, hard rules.

## Language

Respond in the same language as the user: Vietnamese ↔ tiếng Việt, English ↔ English.

## Key Conventions

- **WS protocol:** Frames `req`/`res`/`event`. First request must be `connect`. All method params **camelCase** (`teamId`, `taskId`, `sessionKey`) — match Go `json:"..."` tags.
- **Config:** JSON5 at `GOCLAW_CONFIG` env. Secrets in `.env.local` or env vars, never in config.json.
- **Security:** Rate limiting, input guard (detection-only), CORS, shell deny patterns, SSRF protection, path traversal prevention, AES-256-GCM encryption. Security logs: `slog.Warn("security.*")`.
- **Telegram:** LLM output → `SanitizeAssistantContent()` → `markdownToTelegramHTML()` → `chunkHTML()` → `sendHTML()`. Tables as ASCII in `<pre>`.
- **i18n:** Web UI: `i18next`, locale files in `ui/web/src/i18n/locales/{lang}/`. Backend: `i18n.T(locale, key, args...)`. Locale via `store.WithLocale(ctx)` — WS `connect.locale`, HTTP `Accept-Language`. Supported: en (default), vi, zh. New strings: add key to `internal/i18n/keys.go` + 3 catalogs. New UI strings: add to all 3 locale dirs. Bootstrap templates (SOUL.md) stay English-only.
- **Context:** `store.WithAgentType`, `WithUserID`, `WithAgentID`, `WithLocale`, `WithTenantID`.

## Cross-Surface Parity

Every feature/fix must audit 4 surfaces: **Gateway server** (handlers, WS methods, stores, migrations), **API contract** (`pkg/protocol`, OpenAPI), **Web UI** (`ui/web`), **CLI/runtime** (`cmd`). Don't ship backend-only when UI/CLI/API must change. If unaffected: `Surface parity: <surface> N/A because ...`.

## Running

```bash
go build -o goclaw . && ./goclaw onboard && source .env.local && ./goclaw
./goclaw migrate up

# Integration tests (pgvector pg18 on port 5433)
docker run -d --name pgtest -p 5433:5432 -e POSTGRES_PASSWORD=test -e POSTGRES_DB=goclaw_test pgvector/pgvector:pg18
TEST_DATABASE_URL="postgres://postgres:test@localhost:5433/goclaw_test?sslmode=disable" \
  go test -v -tags integration ./tests/integration/

# Layered tests
make test-invariants    # P0 - tenant isolation (blocking)
make test-contracts     # P1 - API schemas (requires server)
make test-scenarios     # P2 - user journeys (requires server)
make test-critical      # P0 + P1 (pre-merge)

cd ui/web && pnpm install && pnpm dev          # Web dashboard
cd ui/desktop && wails dev -tags sqliteonly    # Desktop (Wails + SQLite)
make desktop-build VERSION=0.1.0               # Build .app/.exe
```

## CI/CD & Releases

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yaml` | push main, PR→main/dev | Build+test+vet |
| `dev-beta-release.yaml` | push dev | Auto beta prerelease + zuey deploy |
| `release.yaml` | tag `vX.Y.Z` (clean semver) | Binaries + Docker (4 variants) |
| `release-beta.yaml` | tag `v*-beta*`/`v*-rc*` | Beta binaries + Docker |
| `release-desktop.yaml` | tag `lite-v*` | Desktop app (macOS+Windows) |

**Release commands:** Standard: `git tag v3.0.0 && git push origin v3.0.0`. Beta: `git push origin dev` (auto) or `git tag v2.67.0-beta.1 && git push origin v2.67.0-beta.1` (manual). Desktop: `git tag lite-v1.1.0 && git push origin lite-v1.1.0`.

**Docker:** GHCR (`ghcr.io/nextlevelbuilder/goclaw`) + Docker Hub (`digitop/goclaw`). Variants: `:latest` (backend+web+Python), `:base` (backend only), `:full` (all runtimes+skills), `-web:latest` (Nginx), `:beta`.

**Tag safety:** `release.yaml` only matches clean `v[0-9]+.[0-9]+.[0-9]+`, never beta/rc. `dev-beta-release.yaml` is branch-triggered (push dev). `lite-` prefix prevents desktop overlap.

## Desktop Edition (Lite)

`//go:build sqliteonly` — SQLite only. `internal/edition/edition.go` auto-selects `Lite` preset. Entry: `ui/desktop/main.go` (Wails v2). Port: 18790 (localhost, `GOCLAW_PORT`). Secrets: OS keyring (`go-keyring`) + file fallback `~/.goclaw/secrets/`. Data: `~/.goclaw/data/`, workspace: `~/.goclaw/workspace/`. Version: `cmd.Version` via `-ldflags`. Auto-update: `internal/updater/updater.go` checks `lite-v*` tags.

**Limits:** 5 agents, 1 team, 5 members, 50 sessions. No channels, heartbeat, file storage UI, skill self-manage, KG, RBAC, multi-tenant.

**Tool gating:** `TeamActionPolicy` blocks comment/review/approve/reject/attach/ask_user. `skill_manage`/`publish_skill` not registered.

**File serving:** 2-layer path isolation (`internal/http/files.go`) — workspace boundary + tenant scope (standard only).

## Plan Verification Rules

Verify before finalizing any multi-phase plan (scout → planner → audit-verify → report):

1. Re-grep every claim, path, endpoint against code. Don't copy scout summaries.
2. Trace control flow when referencing existing code — identify WHEN and under WHAT conditions each field mutates.
3. Every symbol must cite `file:line`. No fabricated wrappers. Use `go doc <pkg>` to verify exported surface.
4. Verify struct lifetime (per-request/session/agent/process) before adding fields. Shared-instance state leaks.
5. List all early-returns before asserting "feature X triggers independently of Y".
6. Match upstream config shape exactly; flag any divergence with rationale.
7. Verify external API endpoints — sibling APIs often use different roots.
8. `grep -rn '<symbol>' .` whole repo for delete scope. Enumerate ALL call sites.
9. List all callers explicitly on signature changes — "update all callers" is insufficient.
10. Scout `ui/desktop/frontend/` and `ui/web/` separately.
11. Re-scout if phase promotes from deferred → active.
12. Write characterization test BEFORE migration — not optional.
13. Add i18n key + 3 catalogs BEFORE handler code — missing key = runtime crash.
14. Fresh explore/grep audit after rewrite — don't trust self-validation.

**Red-team practice:** After planner completes, audit mode: "spot-check 15+ claims vs live codebase". Past catches: fabricated `crypto.Keyring`/`tracing.StartSpan`, inverted TS-port semantics, wrong struct scope, misread early-return gate. See `plans/*/reports/audit-*.md`.

## Post-Implementation Checklist

```bash
go fix ./... && go build ./... && go build -tags sqliteonly ./... && go vet ./...
go test -race ./tests/integration/
```

**Project-specific conventions:**
- **Dual-DB migrations:** PG: `migrations/` + bump `RequiredSchemaVersion`. SQLite: update `schema.sql` + add patch in `schema.go` `migrations` map + bump `SchemaVersion`. Always update both.
- **i18n:** Add key to `internal/i18n/keys.go` + `catalog_{en,vi,zh}.go`. UI: all locale JSONs.
- **SQL safety:** Parameterized queries (`$1, $2`). Check existing indexes. No N+1 queries.
- **DB query reuse:** Prefer passing resolved data through context/events/params over re-querying.
- **Tenant-scope guards:** `RoleAdmin` ≠ tenant check. Global tables → `requireMasterScope`. Tenant tables → `requireTenantAdmin` + `WHERE tenant_id = $N`. Predicate: `store.IsMasterScope(ctx)`. See `CONTRIBUTING.md`.
- **No load/benchmark tests** — flake on shared CI. Use unit + integration + chaos instead.

## Mobile UI/UX

| Rule | Requirement |
|------|-------------|
| Viewport | `h-dvh` (never `h-screen`) |
| Inputs | `text-base md:text-sm` (≥16px on mobile, prevents iOS zoom) |
| Safe areas | `viewport-fit=cover`, `safe-{top,bottom,left,right}` classes |
| Touch targets | ≥44px via `@media (pointer: coarse)` with `::after` |
| Tables | Wrap in `overflow-x-auto`, `min-w-[600px]` |
| Grids | `grid-cols-1 sm:grid-cols-2 lg:grid-cols-N` |
| Dialogs | Full-screen mobile (`max-sm:inset-0`), centered desktop (`sm:max-w-lg`) |
| Keyboard | `useVirtualKeyboard()` + `var(--keyboard-height)` |
| Scroll | `overscroll-contain`, smooth auto-scroll for incoming, instant on send |
| Landscape | `landscape-compact` class (`max-height: 500px`) |
| Portals | Custom portals in dialogs: add `pointer-events-auto` |
| Timezone | `formatBucketTz()` with native `Intl.DateTimeFormat` |
| ErrorBoundary | `stableErrorBoundaryKey(pathname)` strips dynamic segments. Never `key={location.pathname}` on Outlet wrapper. |
| Route params | Derive from `useParams()`, never duplicate into `useState`. Use optional params. |
