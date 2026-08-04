# BuckTooth Architecture

## Component Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  BuckTooth Gateway (Single Go Binary ~25 MB)                                  │
│                                                                                │
│  ┌──────────────────────────────┐  ┌──────────────────────────────────────┐  │
│  │  HTTP Server (:8080)         │  │  WebSocket Server (:18789)           │  │
│  │  REST API · Dashboard        │  │  Token streaming · Client events     │  │
│  │  OpenAI-compat completions   │  └──────────────────────────────────────┘  │
│  │  Prometheus metrics          │                                             │
│  └──────────────────────────────┘                                             │
│                                                                                │
│  ┌──────────────────────────────────────────────────────────────────────────┐ │
│  │  Event Bus                                                               │ │
│  │  message.received · message.sent · channel.connected · agent.*          │ │
│  └──────────────────────┬─────────────────────────────────────────────────┘ │
│                         │                                                      │
│  ┌──────┬──────┬────────┬──────┬─────────┬────────┐                          │
│  │Discord│Telegram│WhatsApp│Slack│  Teams  │Signal  │  ← Channel adapters     │
│  └──────┴──────┴────────┴──────┴─────────┴────────┘                          │
│                         │                                                      │
│  ┌──────────────────────▼──────────────────────────────────────────────────┐ │
│  │  Agent Router                                                           │ │
│  │  Deduplication · Rate limiting · Stats · User preferences              │ │
│  └──────────────────────┬──────────────────────────────────────────────────┘ │
│                         │                                                      │
│  ┌──────────────────────▼──────────────────────────────────────────────────┐ │
│  │  LLM (7 providers + retry/backoff + fallback chain)                     │ │
│  │  Anthropic · OpenAI · Gemini · Bedrock · Ollama · LiteLLM · Stub       │ │
│  └──────────────────────┬──────────────────────────────────────────────────┘ │
│                         │                                                      │
│  ┌──────────┐  ┌────────┴───────┐  ┌──────────────┐  ┌────────────────────┐ │
│  │  Memory  │  │  Tool Registry │  │  Skill       │  │  Cost Tracker      │ │
│  │  5 types │  │  11 tools      │  │  Registry    │  │  per-model USD     │ │
│  └──────────┘  └────────────────┘  └──────────────┘  └────────────────────┘ │
│                                                                                │
│  ┌──────────────────────┐  ┌─────────────────┐  ┌───────────────────────┐   │
│  │  Cron Scheduler      │  │  MCP Clients    │  │  Chunker / Formatter  │   │
│  │  scheduled messages  │  │  tool bridge    │  │  per-platform split   │   │
│  └──────────────────────┘  └─────────────────┘  └───────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Components

### Gateway (`internal/gateway/gateway.go`)

The top-level orchestrator. Initialises all subsystems, manages their lifecycle, and routes the `message.received` event through the processing pipeline.

### Event Bus (`internal/gateway/eventbus.go`)

An in-process publish/subscribe bus. Event types:

- `message.received` — inbound user message
- `message.sent` — outbound response delivered
- `channel.connected` / `channel.disconnected`
- `agent.started` / `agent.completed` / `agent.error`

### Channels (`internal/channels/`)

Platform adapters. Each implements the `Channel` interface:

```go
type Channel interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect() error
    SendMessage(ctx context.Context, msg *Message) error
    ReceiveMessages(ctx context.Context) (<-chan *Message, error)
    Health() HealthStatus
}
```

Supported: Discord, Telegram, WhatsApp (whatsmeow), Slack (Socket Mode), Microsoft Teams (Bot Framework), Signal (signal-cli JSON-RPC).

### Agent Router (`internal/gateway/router.go`)

Routes messages to the configured agent pattern (conversational / ReAct / planning), manages LLM instances, and wires optional middleware:

- `RateLimiter` — per-user token-bucket
- `CostTracker` — per-model USD accounting
- `Summarizer` — LLM-based history compaction

### LLM Layer

Backed by agenkit-go adapters. Supports retry with exponential backoff (configurable attempts + initial backoff) and a fallback provider chain — if the primary LLM fails with a non-retryable error, the next provider in `fallback_providers` is tried.

### Memory Store (`internal/memory/`)

Five pluggable backends:

| Backend | Description |
|---------|-------------|
| `inmemory` | In-process map (default, lost on restart) |
| `redis` | Persistent, distributed |
| `sqlite` | Local file, no external service required |
| `vector` | Cosine-similarity semantic search |
| `hybrid` | BM25 + semantic search with configurable weight |

Optional: memory summarization (message-count or token-count trigger), temporal recency decay.

### Tool Registry (`internal/tools/`)

11 tools registered at startup: calculator, file system (sandboxed), web search (Brave), web fetch, calendar, shell (with approval gates), PDF extraction, image analysis, browser automation (Chrome CDP), message (cross-channel), inter-channel message. Additional tools are registered dynamically from MCP servers.

### Skill Registry

Discovers `SKILL.md` files from configured `search_paths` and injects skill instructions into agent prompts. Optional dependency validation ensures required binaries and env vars are available before injection.

### HTTP Server (`internal/gateway/http.go`)

Serves the web dashboard (embedded static assets), REST API endpoints, Prometheus metrics, and OpenAI-compatible completions. Optional Bearer token auth and Basic auth for dashboard routes.

### WebSocket Server (`internal/gateway/websocket.go`)

Provides token-by-token streaming for long responses and broadcasts gateway events to the dashboard's real-time feed.

### Cron Scheduler (`internal/cron/`)

Injects scheduled messages into the event bus at configured intervals.

### MCP Clients

Connects to external Model Context Protocol servers (stdio subprocess or HTTP) at startup and registers their tools into the tool registry.

### Chunker / Formatter (`internal/gateway/chunker.go`, `formatter.go`)

The Chunker splits long responses into platform-appropriate segments (e.g. Discord 2 000 chars, Slack 4 000 chars). The Formatter applies channel-aware Markdown transformations before delivery.

---

## Message Flow

```
1.  User sends message on a platform
2.  Channel adapter receives it and emits a channels.Message
3.  Gateway publishes message.received event
4.  Deduplicator checks ring buffer — drops duplicate
5.  Stats recorder increments inbound counter
6.  Rate limiter checks per-user token bucket
7.  Agent router retrieves conversation history from memory store
8.  Router passes message + history to LLM (with retry + fallback)
9.  LLM returns response text
10. Cost tracker records token usage and USD cost
11. Memory store appends message + response
12. Formatter applies channel-aware Markdown
13. Chunker splits into platform-sized pieces
14. Channel adapter delivers each chunk (100 ms gap between chunks)
15. Stats recorder increments outbound counter
16. Gateway publishes message.sent and agent.completed events
17. Dashboard WebSocket clients receive real-time update
```

---

## Configuration Loading

Priority order (highest wins):

1. CLI flags (`--flag`)
2. Environment variables (`LOBSTER_*`, `ANTHROPIC_API_KEY`, etc.)
3. YAML config file (default: `configs/gateway.yaml`)
4. Built-in defaults (`config.DefaultConfig()`)

`${VAR_NAME}` substitution is performed on YAML values at load time.

---

## Observability

- **Logging**: structured JSON via zerolog; per-component loggers; configurable level
- **Metrics**: Prometheus counters and histograms at `GET /metrics`
- **Tracing**: OpenTelemetry spans exported via OTLP HTTP
- **Cost tracking**: per-model USD totals via `GET /v1/usage`
- **Health probes**: `/live` (always 200), `/ready` (503 when no channel healthy), `/health` (channel details)
- **Security audit**: `GET /admin/audit` returns scored security posture report
