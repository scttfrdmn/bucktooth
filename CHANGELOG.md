# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-03-17

### Fixed
- `internal/gateway/router.go`: per-user `ConversationalAgent` isolation — single shared instance was not goroutine-safe; each user now gets their own agent with isolated history

### Added
- `internal/gateway/stats.go`: `Stats` with lock-free `atomic.Uint64` message counters, 50-entry recent-message ring buffer (newest-first), and uptime tracking
- `internal/gateway/userprefs.go`: `UserPrefs` store for per-user preferred response channel
- `GET /live` — liveness probe (always 200); `GET /ready` — readiness probe (503 if no channel operational)
- `GET /dashboard/data` — JSON stats endpoint (version, uptime, counters, active users, channels, recent messages)
- Dashboard Basic auth via `gateway.dashboard_auth_password` config / `DASHBOARD_AUTH_PASSWORD` env
- `GET /users/:user_id/preferences`, `POST /users/:user_id/preferences` — cross-channel routing preferences
- `AgentRouter.ActiveUsers() int` — live count of distinct users with active agents
- `/plan <task>` command prefix activates planning agent per-message regardless of `agents.mode`
- `agents.api_base` config field / `ANTHROPIC_API_BASE` env var for Anthropic endpoint override (stored; pending agenkit-go base URL support)

## [0.4.5] - 2026-03-17

### Added
- `StubLLM` implementing `llm.LLM`; echo mode returns `"echo: <input>"`, fixed-response mode via `agents.stub_response` config field
- `TestChannel` with `POST /test/send` and `GET /test/responses` HTTP endpoints for integration testing
- `configs/harness.yaml` — zero-credential harness configuration (stub LLM + test channel + inmemory store)
- `Dockerfile.harness` — alpine-based image with `wget` for healthcheck
- `docker-compose.harness.yml` — single-service harness compose (no credentials, no Redis)
- Integration test suite in `harness/harness_test.go` (`//go:build integration`): `TestHealthEndpoint`, `TestSendAndReceiveEcho`, `TestMultipleMessages`
- `make test-harness` target: spins up harness, runs integration tests, tears down

### Changed
- `internal/gateway/router.go`: `AgentRouter` now accepts `llm.LLM` interface; `NewAgentRouter` switches on `llm_provider: stub` to inject `StubLLM`
- `internal/agents/planning.go`: `ToolStepExecutor.llm` changed from concrete `*llm.AnthropicLLM` to `llm.LLM` interface
- `internal/gateway/http.go`: `HTTPServer` now supports `Handle(pattern, handler)` for programmatic route registration
- `internal/config/config.go`: `GatewayConfig.TestChannel` and `AgentConfig.StubResponse` fields added

## [0.4.1] - 2026-03-17

### Changed
- Replace all `interface{}` with `any` throughout the codebase (Go 1.18+ idiom)
- `internal/config/loader.go`: replace `fmt.Sscanf` with `strconv.Atoi` for env-var port parsing; errors now silently ignored rather than silently wrong
- `internal/gateway/http.go`: check and log errors from `json.Encoder.Encode` in `/health` and `/status` handlers
- `internal/gateway/http.go`, `internal/gateway/websocket.go`: check and log errors from `conn.Close()` in shutdown paths and defer blocks
- `internal/tools/websearch.go`: lowercase error string per Go convention (`ST1005`)
- `cmd/bucktooth/cmd/start.go`: lowercase error string per Go convention (`ST1005`)
- `defer resp.Body.Close()` wrapped as `defer func() { _ = resp.Body.Close() }()` to satisfy errcheck
- `internal/tools/websearch_test.go`: check error from `json.Encoder.Encode` in mock HTTP handler
- Remove unused `mockLLMClient` type from `internal/agents/planning_test.go`
- Apply `gofmt` to 11 files with import-grouping and struct-alignment drift

## [0.4.0] - 2026-03-17

### Added
- Enhanced CLI with cobra: `start`, `status`, `config validate`, `version` subcommands; shell completion auto-provided by cobra (#17)
- Web dashboard at `/` (embedded, no build step) with live WebSocket feed at `/api/ws` showing message and channel events in real time (#18)
- Multi-stage distroless Dockerfile; `bucktooth status` used as `HEALTHCHECK CMD` (#15)
- `docker-compose.yml` with BuckTooth + Redis (appendonly, health-checked) (#15)
- Helm chart `charts/bucktooth/` with Deployment, Service, ConfigMap, Secret, Ingress, HPA templates (#16)
- Benchmark suite in `bench/` covering EventBus, InMemoryStore, HTTPServer, and Config parse (#20)
- `docs/benchmarks.md` — methodology and baseline results table (#20)
- `docs/migration.md` — OpenClaw → BuckTooth feature comparison and step-by-step migration guide (#19)
- `docs/docker.md` — Docker quick start, env var table, Redis memory config (#15)

### Changed
- `BINARY_NAME` and `MAIN_PATH` in Makefile updated to `bucktooth` / `./cmd/bucktooth`
- `run-debug` Makefile target updated to `start --log-level debug`
- `bench` Makefile target scoped to `./bench/...` with `-benchtime=5s`
- `docker-compose-up` and `docker-compose-down` targets added to Makefile

### Removed
- `cmd/gateway/main.go` — replaced by `cmd/bucktooth/`

## [0.3.0] - 2026-03-17

### Added
- Telegram channel integration via long-polling (#9)
- Slack channel integration via Socket Mode (#10)
- Planning agent with tool-backed step executor, activated by `agents.mode: planning` (#11)
- Web search tool using Brave Search API (#13)
- Calendar tool with local JSON event store (#12)
- OpenTelemetry tracing with OTLP HTTP exporter, spans on message handling and agent processing (#14)

### Fixed
- Service name default corrected from "lobster" to "bucktooth" in tracing config

## [0.2.0] - 2026-03-17

### Added
- WhatsApp channel integration via whatsmeow library with QR-code authentication and SQLite session storage
- Redis-backed conversation memory store with configurable TTL and history trimming
- Tool registry with dynamic tool registration and discovery
- Calculator tool for arithmetic operations (add, subtract, multiply, divide, modulo)
- Message formatter tool for platform-specific text formatting (Discord, plain, Markdown)
- Sandboxed filesystem tool for file read/write/list/delete operations with path traversal protection
- Tool-augmented agent path using ReAct pattern (agenkit ReActAgent) with automatic fallback to conversational
- Unit tests for calculator and filesystem tools

## [0.1.0] - 2026-02-01

### Added
- Initial release of BuckTooth multi-channel AI assistant gateway
- Single Go binary deployment (25MB)
- Discord channel integration with full message handling
- Event-driven architecture with pub-sub event bus
- Conversational AI powered by Claude Sonnet 4.5 (claude-sonnet-4-5-20250220)
- In-memory conversation storage with per-user history
- HTTP API server with `/health`, `/status`, and `/metrics` endpoints
- WebSocket server for real-time client connections (port 18789)
- Prometheus metrics integration for observability
- Structured JSON logging using zerolog
- Configuration via YAML files, environment variables, and CLI flags
- Graceful shutdown with proper cleanup
- Health monitoring for channels
- OpenTelemetry integration for tracing (configurable)
- Channel abstraction layer for multi-platform support
- Agent router for message processing
- Base channel implementation with health tracking
- Configuration validation and defaults
- Unit tests for configuration module
- Comprehensive documentation (README, QUICKSTART, architecture docs)
- Development tooling (Makefile, .gitignore, .env.example)

### Technical Details
- Go 1.24.0 (requires Go 1.23+)
- Agenkit-Go framework for AI orchestration
- discordgo v0.28.1 for Discord integration
- gorilla/websocket v1.5.3 for WebSocket support
- prometheus/client_golang v1.20.5 for metrics
- zerolog v1.32.0 for structured logging
- OpenTelemetry v1.38.0 for observability

### Performance
- Binary size: 25MB
- Memory footprint: <100MB at startup
- Agent processing overhead: ~2ms
- Target throughput: 1,000 messages/second (Phase 4 goal)

[unreleased]: https://github.com/scttfrdmn/bucktooth/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.4.5...v0.5.0
[0.4.5]: https://github.com/scttfrdmn/bucktooth/compare/v0.4.1...v0.4.5
[0.4.1]: https://github.com/scttfrdmn/bucktooth/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/bucktooth/releases/tag/v0.1.0
