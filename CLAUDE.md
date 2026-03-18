# BuckTooth - Claude Code Guide

## Project Overview

BuckTooth is a multi-channel AI assistant gateway (v0.1.0) written in Go. It connects messaging platforms (Discord, WhatsApp, Telegram, Slack) to Claude AI via the Agenkit-Go SDK.

- **GitHub**: https://github.com/scttfrdmn/bucktooth
- **License**: Apache 2.0
- **Go module**: `github.com/scttfrdmn/bucktooth`

## Build & Run Commands

```bash
# Build
make build          # Produces bin/BuckTooth
go build -o bin/BuckTooth ./cmd/gateway

# Run
make run            # go run ./cmd/gateway
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
| `platform:` | Target platform | `platform:discord`, `platform:whatsapp`, `platform:telegram`, `platform:slack` |

### Milestones

- **v0.1.0 - Foundation** (closed) — current codebase
- **v0.2.0 - Tools & Multi-Channel** — tool registry, WhatsApp, Redis memory
- **v0.3.0 - Platform Expansion** — Telegram, Slack, planning agent, observability
- **v1.0.0 - Production Ready** — Docker, Kubernetes, web dashboard, CLI

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
cmd/gateway/main.go          # Entry point
internal/
├── gateway/                 # Core gateway
│   ├── gateway.go           # Gateway struct, lifecycle
│   ├── router.go            # Message routing to agents
│   ├── http.go              # HTTP API server (:8080)
│   ├── websocket.go         # WebSocket server (:18789)
│   └── eventbus.go          # Internal event bus
├── channels/                # Platform adapters
│   ├── channel.go           # Channel interface
│   ├── base.go              # Base channel implementation
│   └── discord/discord.go   # Discord adapter
├── agents/                  # Agent implementations (planned)
├── tools/                   # Tool implementations (planned)
├── memory/                  # Conversation memory
│   ├── memory.go            # Memory interface
│   └── inmemory.go          # In-memory store
└── config/                  # Configuration
    ├── config.go             # Config structs
    └── loader.go             # YAML/env loading
configs/gateway.yaml         # Example configuration
docs/architecture.md         # Architecture documentation
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

- `GET /health` — Health check with channel status
- `GET /status` — Detailed status
- `GET /metrics` — Prometheus metrics
- `WS /ws` — WebSocket (port 18789)

## Configuration

Required environment variables:
- `DISCORD_BOT_TOKEN` — Discord bot token
- `ANTHROPIC_API_KEY` — Anthropic API key

Optional:
- `LOBSTER_GATEWAY_PORT` — HTTP port (default: 8080)
- `LOBSTER_WEBSOCKET_PORT` — WebSocket port (default: 18789)
- `LOBSTER_LOG_LEVEL` — Log level (debug/info/warn/error)

## Key Conventions

- **Error handling**: Wrap errors with `fmt.Errorf("context: %w", err)`
- **Logging**: Use structured logging via `slog`
- **Context**: Always propagate `context.Context` as first parameter
- **Testing**: Table-driven tests; integration tests use real dependencies (no mocks for DB/external systems)
- **Commits**: Conventional commits format (`feat:`, `fix:`, `docs:`, `chore:`, etc.)

## Dependencies

- `github.com/scttfrdmn/agenkit-go` — AI agent framework (Agenkit SDK)
- `github.com/bwmarrin/discordgo` — Discord API
- `gopkg.in/yaml.v3` — YAML config parsing
- `github.com/joho/godotenv` — .env file loading
