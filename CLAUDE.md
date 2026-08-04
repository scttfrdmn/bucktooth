# BuckTooth - Claude Code Guide

## Project Overview

BuckTooth is a multi-channel AI assistant gateway (v0.14.0) written in Go. It connects six messaging platforms (Discord, Telegram, WhatsApp, Slack, Microsoft Teams, Signal) to seven LLM providers via the Agenkit-Go SDK.

- **GitHub**: https://github.com/scttfrdmn/bucktooth
- **License**: Apache 2.0
- **Go module**: `github.com/scttfrdmn/bucktooth`
- **Agenkit module**: `github.com/scttfrdmn/agenkit/agenkit-go` (monorepo subdirectory), pinned to a published pseudo-version

### Two agenkit repos — use the monorepo, not the mirror

| Repo | Module path | Use it? |
|------|-------------|---------|
| `scttfrdmn/agenkit` | `github.com/scttfrdmn/agenkit/agenkit-go` | **Yes** — source of truth |
| `scttfrdmn/agenkit-go` | `github.com/scttfrdmn/agenkit-go` | No — distribution mirror, lags behind |

The mirror publishes clean semver tags (`v0.87.0`) and is tempting to pin to,
but it lags the monorepo badly: as of 2026-08-03 its `v0.87.0` dates from
2026-06-20 and still exposes the **old** `patterns` LLM interface (`Chat`),
not the current `Complete`. Pinning it would silently revert the fix in
`internal/gateway/router.go`. The module paths also differ, so it is not a
drop-in substitute.

The monorepo has no subdirectory-scoped tags (Go needs `agenkit-go/vX.Y.Z`),
so a **pseudo-version is the only correct pin** until those are published.
To bump it:

```bash
go get github.com/scttfrdmn/agenkit/agenkit-go@<monorepo-commit-sha>
GOWORK=off go build ./... && go test ./...
```

Confirm the target commit is on `origin/main` first — a pseudo-version for an
unpushed commit resolves locally but fails in CI.

### Local agenkit-go development

`go.work` is intentionally **untracked and gitignored**. Committing it breaks
every clean checkout, CI, and `docker build`, because it references a sibling
`../agenkit/agenkit-go` path that only exists on a dev machine. For the same
reason there is no `replace` directive in `go.mod`.

To iterate against a local agenkit-go checkout, create the workspace locally:

```bash
go work init . ../agenkit/agenkit-go   # local only — never commit
```

Before pushing, verify the build works *without* the workspace: `GOWORK=off go build ./...`

## Build & Run Commands

```bash
# Build
make build          # Produces bin/bucktooth
go build -o bin/bucktooth ./cmd/bucktooth

# Run
make run            # go run ./cmd/bucktooth
make run-debug      # with --log-level debug

# Test
make test           # go test -v ./...
make test-coverage  # with coverage report
make test-race      # with race detector

# Quality
make fmt            # go fmt ./...
make vet            # go vet ./...
make lint           # golangci-lint run ./...
make check          # fmt + vet + lint + test

# Deps
make deps           # go mod download && go mod tidy
```

## CRITICAL: Task Tracking Rules

**Use GitHub for ALL task and project tracking. Never create local tracking files.**

- **Issues**: All bugs, features, enhancements → `gh issue create`
- **Milestones**: Release planning → use existing milestones (v0.2.0, v0.3.0, v1.0.0)
- **Project board**: BuckTooth Roadmap at https://github.com/users/scttfrdmn/projects/
- **Labels**: Use the structured label taxonomy (see below)

**DO NOT create**: TODO.md, TASKS.md, STATUS.md, PLAN.md, NOTES.md, or any other local tracking files. All work tracking belongs in GitHub Issues.

### Label Taxonomy

| Prefix | Purpose | Examples |
|--------|---------|---------|
| `type:` | What kind of work | `type:bug`, `type:feature`, `type:enhancement`, `type:docs`, `type:chore`, `type:test` |
| `area:` | Codebase area | `area:gateway`, `area:channels`, `area:agents`, `area:memory`, `area:config`, `area:observability`, `area:tools`, `area:api` |
| `priority:` | Urgency | `priority:critical`, `priority:high`, `priority:medium`, `priority:low` |
| `status:` | Current state | `status:blocked`, `status:in-progress`, `status:needs-review` |
| `platform:` | Target platform | `platform:discord`, `platform:whatsapp`, `platform:telegram`, `platform:slack`, `platform:teams`, `platform:signal` |

### Milestones

Releases v0.1.0 through v0.14.0 have shipped; see `CHANGELOG.md` for the
per-version breakdown and `gh api repos/scttfrdmn/bucktooth/milestones` for
current state. Recent themes:

- **v0.9.0 - Connections & Tooling** — rate limiting, BM25 hybrid memory, PDF/image tools, cron, admin API
- **v0.10.0 - Reliability & Quality** — chunking, dedup, retry/backoff, fallback chain
- **v0.11.0 - Streaming & Intelligence** — WS token streaming, `/v1/chat/completions`, temporal decay
- **v0.12.0 - Ecosystem & Automation** — SKILL.md dep validation, browser automation, `/v1/models`
- **v0.13.0 - Signal, Observability & CI/CD** — Signal channel, cost tracking, GitHub Actions
- **v0.14.0 - Docs, Deployment & Security Hardening** — user docs, multi-arch GHCR, Helm, `/admin/audit`

**When cutting a release, close its milestone's issues.** Issues from shipped
milestones were repeatedly left open; `make release-tag` does not close them.

## CRITICAL: Parity Tracking

**Every milestone plan must include a gap audit against OpenClaw and RustyNail.**

BuckTooth tracks feature parity with two sibling projects:
- **RustyNail** — Rust implementation of the same gateway (same version cadence)
- **OpenClaw** — Reference implementation / ClawHub ecosystem standard

Before proposing or planning any milestone:
1. Run a gap audit: identify features in RustyNail and OpenClaw that BuckTooth lacks.
2. Include a parity section in the milestone plan listing gaps, their effort/value ratio, and which are targeted for this sprint vs deferred.
3. Create GitHub issues with label `type:enhancement` and reference the sibling project (e.g. "parity: RustyNail v0.8.0").
4. The comparison doc lives at `docs/openclaw-comparison-2026-03.md` — update it when parity changes.

**Do not plan a milestone without a gap audit first.**

## Architecture

```
cmd/bucktooth/               # Entry point (cobra CLI)
│   main.go
│   cmd/                     # start, status, config, mcp, version, completion
internal/
├── gateway/                 # Core gateway
│   ├── gateway.go           # Gateway struct, lifecycle, backend wiring
│   ├── router.go            # AgentRouter — per-user agent isolation
│   ├── http.go              # HTTP API server (:8080), admin + /v1 endpoints
│   ├── websocket.go         # WebSocket server (:18789), token streaming
│   ├── eventbus.go          # Internal event bus
│   ├── stats.go             # Atomic counters + recent-message ring buffer
│   ├── chunker.go           # Per-platform long-message splitting
│   ├── deduplicator.go      # SHA-256 dedup ring buffer
│   ├── formatter.go         # Channel-aware markdown formatting
│   ├── retryllm.go          # Exponential backoff + jitter
│   ├── fallbackllm.go       # Multi-provider fallback chain
│   ├── stubllm.go           # Zero-credential test LLM
│   ├── ratelimiter.go       # Per-user token bucket
│   ├── costtracker.go       # LLM USD cost accounting
│   └── userprefs.go         # Per-user channel + system-prompt overrides
├── channels/                # Platform adapters (one package each)
│   ├── channel.go           # Channel interface
│   ├── base.go              # Base channel implementation
│   ├── discord/ telegram/ whatsapp/ slack/ teams/ signal/
│   ├── testchan/            # Harness test channel
│   └── webhook/verify.go    # HMAC-SHA256 verification middleware
├── agents/planning.go       # PlanningAgent + ToolStepExecutor
├── tools/                   # calculator, filesystem, websearch, webfetch,
│                            # calendar, shell, pdf, image, browser, message
│   ├── tool.go              # Tool interface + Registry
│   └── from_config.go       # Shared by gateway and `mcp serve`
├── memory/                  # inmemory, redis, sqlite, vector, hybrid (BM25)
│   ├── memory.go            # Store interface
│   └── summarizer.go        # Async LLM conversation compaction
├── skills/depcheck.go       # ClawHub SKILL.md dependency validation
├── cron/scheduler.go        # Ticker-based job engine
├── observability/           # Prometheus metrics + OTel tracing
└── config/                  # config.go (structs), loader.go (YAML/env)
harness/                     # Integration tests (//go:build integration)
bench/                       # Benchmarks
charts/bucktooth/            # Helm chart
web/                         # Embedded dashboard assets
configs/gateway.yaml         # Example configuration
docs/                        # getting-started, channels, configuration-reference,
                             # architecture, docker, migration, benchmarks, parity
```

### Key Interfaces

```go
// Channel — each platform implements this
type Channel interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg Message) error
    ID() string
    Type() ChannelType
}

// Memory — conversation history store
type Store interface {
    GetHistory(ctx context.Context, channelID, userID string) ([]Message, error)
    AddMessage(ctx context.Context, channelID, userID string, msg Message) error
    ClearHistory(ctx context.Context, channelID, userID string) error
}
```

## API Endpoints

Probes (always unauthenticated):
- `GET /live` — Liveness (always 200)
- `GET /ready` — Readiness (503 if no channel operational)

Core:
- `GET /health` — Health check with channel status
- `GET /status` — Detailed status
- `GET /metrics` — Prometheus metrics
- `WS /ws` — WebSocket (port 18789), supports token streaming
- `GET /dashboard`, `GET /dashboard/data` — Dashboard (optional Basic auth)

Admin (Bearer-token protected when `api_token` is set):
- `GET /admin/audit` — Security posture report
- `GET /admin/memory/{user_id}/flush`, `POST /admin/skills/reload`
- `GET /admin/skills/deps`, `GET /admin/channels/health`

OpenAI-compatible:
- `POST /v1/chat/completions` (JSON + SSE), `GET /v1/models`, `GET /v1/usage`

See `docs/configuration-reference.md` for the full list.

## Configuration

Required environment variables:
- `ANTHROPIC_API_KEY` — Anthropic API key (or the equivalent for another provider)
- At least one channel credential, e.g. `DISCORD_BOT_TOKEN`

Optional:
- `BUCKTOOTH_API_TOKEN` — Bearer token for API auth
- `BUCKTOOTH_RATE_LIMIT_RPM`, `BUCKTOOTH_SKILLS_PATH`, `BUCKTOOTH_WS_ALLOWED_ORIGINS`
- `DASHBOARD_AUTH_PASSWORD` — Enables dashboard Basic auth
- `LOBSTER_GATEWAY_PORT` — HTTP port (default: 8080)
- `LOBSTER_WEBSOCKET_PORT` — WebSocket port (default: 18789)
- `LOBSTER_LOG_LEVEL` — Log level (debug/info/warn/error)
- `LOBSTER_LLM_MODEL` — Model override

Note: the four `LOBSTER_*` names are pre-rename leftovers that are still what
the loader reads. Renaming them to `BUCKTOOTH_*` is a breaking change and needs
a deprecation cycle — don't "fix" them in docs without changing `internal/config/loader.go`.

## Key Conventions

- **Error handling**: Wrap errors with `fmt.Errorf("context: %w", err)`
- **Logging**: Structured logging via `github.com/rs/zerolog` (not `slog`); pass a `zerolog.Logger` down through constructors
- **Context**: Always propagate `context.Context` as first parameter
- **Testing**: Table-driven tests; integration tests use real dependencies (no mocks for DB/external systems) and sit behind `//go:build integration`
- **Commits**: Conventional commits format (`feat:`, `fix:`, `docs:`, `chore:`, etc.)

## Dependencies

- `github.com/scttfrdmn/agenkit/agenkit-go` — AI agent framework (Agenkit SDK)
- `github.com/bwmarrin/discordgo` — Discord API
- `github.com/rs/zerolog` — Structured logging
- `github.com/spf13/cobra` — CLI
- `github.com/prometheus/client_golang`, `go.opentelemetry.io/otel` — Observability
- `modernc.org/sqlite` — Pure-Go SQLite (no CGO)
- `gopkg.in/yaml.v3` — YAML config parsing
