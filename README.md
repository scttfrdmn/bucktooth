# BuckTooth — Multi-Channel AI Assistant Gateway

**"No Claws All Teeth!"** &nbsp;·&nbsp; v0.14.0 &nbsp;·&nbsp; [![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE) [![CI](https://github.com/scttfrdmn/bucktooth/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/bucktooth/actions/workflows/ci.yml)

BuckTooth is a high-performance single-binary AI assistant gateway written in Go. It connects six messaging platforms to seven LLM providers via the [Agenkit-Go](https://github.com/scttfrdmn/agenkit-go) SDK, with conversation memory, tool use, skills, and production-grade observability — all in a ~25 MB binary.

---

## Features

| Category | Details |
|----------|---------|
| **Channels** | Discord, Telegram, WhatsApp, Slack, Microsoft Teams, Signal |
| **LLM providers** | Anthropic Claude, OpenAI, OpenAI-compatible (vLLM/llama.cpp), Google Gemini, AWS Bedrock, Ollama, LiteLLM |
| **Memory backends** | In-memory, Redis, SQLite, vector (cosine), hybrid BM25+vector |
| **Tools** | Calculator, file system, web search, web fetch, calendar, shell (with approval gates), PDF, image analysis, browser automation |
| **Observability** | Prometheus metrics, OpenTelemetry tracing, LLM cost tracking, real-time dashboard |
| **Reliability** | LLM retry/backoff, fallback provider chains, message deduplication, per-platform chunking |
| **API** | OpenAI-compatible `/v1/chat/completions` + `/v1/models`, WebSocket token streaming |
| **Deployment** | Single binary, Docker, Helm chart, GitHub Actions CI/CD, multi-arch (amd64 + arm64) |

---

## Quick Start

```bash
# 1. Install
go install github.com/scttfrdmn/bucktooth/cmd/bucktooth@latest
# or: git clone https://github.com/scttfrdmn/bucktooth && make build

# 2. Set credentials
export ANTHROPIC_API_KEY=sk-ant-...
export DISCORD_BOT_TOKEN=...

# 3. Start
bucktooth start --config configs/gateway.yaml

# 4. Verify
curl -s localhost:8080/health | jq .
```

The web dashboard is available at **http://localhost:8080**.

---

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | First-time setup walkthrough |
| [Configuration Reference](docs/configuration-reference.md) | Every config key documented |
| [Channel Setup](docs/channels.md) | Per-platform bot setup (Discord, Telegram, WhatsApp, Slack, Teams, Signal) |
| [Architecture](docs/architecture.md) | Component overview and message flow |
| [Docker](docs/docker.md) | Docker Compose deployment |
| [Migration](docs/migration.md) | Upgrading from OpenClaw or earlier BuckTooth versions |

---

## Architecture Overview

BuckTooth is a single event-driven process. Inbound messages from any channel flow through deduplication → agent router → LLM → channel-aware formatting → platform delivery. All components share one in-process event bus.

See [docs/architecture.md](docs/architecture.md) for the full component diagram and message flow.

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Channel health check |
| GET | `/live` | Kubernetes liveness probe |
| GET | `/ready` | Kubernetes readiness probe |
| GET | `/status` | Detailed gateway status |
| GET | `/metrics` | Prometheus metrics |
| GET | `/dashboard/data` | Dashboard JSON snapshot |
| POST | `/v1/chat/completions` | OpenAI-compatible completions (streaming + non-streaming) |
| GET | `/v1/models` | OpenAI-compatible model list |
| GET | `/v1/usage` | LLM cost and token totals |
| GET | `/admin/audit` | Security posture report |
| GET | `/admin/channels/health` | Per-channel health detail |
| POST | `/admin/skills/reload` | Hot-reload agent skills |
| GET | `/cron/jobs` | List scheduled cron jobs |

---

## Building from Source

```bash
# Build
make build          # produces bin/bucktooth

# Test
make test           # go test -v ./...
make test-race      # with race detector

# Quality
make check          # fmt + vet + lint + test
```

---

## Contributing

Issues and pull requests welcome. Please open a GitHub issue before submitting large changes.

---

## License

Copyright 2026 Scott Friedman. Licensed under the [Apache License 2.0](LICENSE).
