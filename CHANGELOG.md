# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.14.0] - 2026-08-03

### Added
- User documentation set: `docs/getting-started.md`, `docs/channels.md`, `docs/configuration-reference.md`; README rewritten around a feature matrix and quick start (#70)
- Multi-arch Docker builds (`linux/amd64`, `linux/arm64`) published to GHCR via QEMU + Buildx in the release workflow (#71)
- Helm chart updated to expose all current config options; chart version bumped (#72)
- `GET /admin/audit` security posture endpoint: scored report over `api_token`, `dashboard_auth`, `rate_limiting`, `shell_approval`, `message_dedup`, and `tls` checks; `HTTPServer.SetConfig` wires gateway config into the handler (#73)

### Changed
- `docs/openclaw-comparison-2026-03.md` updated to v0.14.0; exec approval gates and security audit endpoint marked resolved (#74)

### Fixed
- `internal/gateway/router.go`: `llmClientAdapter` implements `patterns.LLMClient.Complete(ctx, messages, ...agenkit.CallOption)` instead of the removed `Chat` method — upstream agenkit converged its LLM protocols on `Complete`, which broke compilation
- Build no longer depends on a sibling `../agenkit/agenkit-go` checkout: the `replace` directive is dropped in favour of a pinned published version, and `go.work`/`go.work.sum` are untracked and gitignored. Restores clean-checkout CI, `docker build`, and `go install ...@latest`

## [0.13.0] - 2026-03-18

### Added
- Signal channel via signal-cli JSON-RPC WebSocket bridge (requires signal-cli ≥ 0.13.0) (#65)
- LLM USD cost tracking with a static pricing table; `GET /v1/usage` endpoint; Prometheus counter `bucktooth_llm_cost_dollars_total{provider,model}` (#66)
- GitHub Actions CI workflow (push/PR) and release workflow (`v*` tag → GHCR image + GitHub release) (#67)
- `make ci` and `make release-tag` Makefile targets (#67)
- Harness integration tests: `TestModelsEndpoint`, `TestSkillsDepsEndpoint`, `TestUsageEndpoint`, `TestSignalChannelUnavailable` (#68)

### Changed
- `docs/openclaw-comparison-2026-03.md` updated to v0.13.0 (#69)

## [0.12.0] - 2026-03-18

### Added
- ClawHub `SKILL.md` dependency validation: new `internal/skills` package with `DepChecker`, `ParseSkillMeta`, `CheckAll`; validates binaries via `exec.LookPath` and env vars from SKILL.md frontmatter; skills with unmet deps are dropped with a warn log; `GET /admin/skills/deps` returns `[]DepResult`; gated by `skills.dep_check_enabled` (#61)
- Browser automation tool via Chrome CDP (`github.com/chromedp/chromedp`): `navigate`, `click`, `type`, `screenshot` (base64 PNG), `extract`; stateless per-call context; gated by `tools.browser_enabled` / `tools.browser_timeout` (#62)
- OpenAI-compatible `GET /v1/models` endpoint returning a static model list for the configured provider; protected by `apiTokenMiddleware` (#63)

### Changed
- `docs/openclaw-comparison-2026-03.md` updated to v0.12.0; Signal channel and ClawMetry deferred (#64)

## [0.11.0] - 2026-03-18

### Added
- WebSocket token streaming: inbound `stream:true` flag, `chunk`/`done`/`error` frames, per-connection write mutex, gated by `StreamingEnabled`
- `AgentRouter.StreamMessage` + `streamConversational` using `llmRaw.Stream()`; ReAct and planning modes emit a single-chunk simulated stream
- OpenAI-compatible `POST /v1/chat/completions`: non-streaming JSON plus SSE streaming with a per-response write-deadline bypass for long streams
- Temporal memory decay in `HybridStore` — `exp(-age*ln2/halfLife)` replaces linear recency; configurable via `decay_enabled` / `decay_half_life_hours`
- Token-based compaction trigger in `Summarizer`: fires when the estimated token count (chars/4) exceeds `summarize_token_threshold`
- Optional Teams HMAC activity validation via `security_token`; checks `Authorization: HMAC <base64-sha256>` (opt-in, backward compatible)

### Fixed
- `Summarizer`: `GetHistory(…, 0)` returned zero rows inside the inner goroutine, so compaction never ran

### Changed
- `docs/openclaw-comparison-2026-03.md` updated to v0.11.0

## [0.10.0] - 2026-03-18

### Added
- Long-message chunking per platform: `Chunker.Split` with paragraph/sentence/word boundary preference and a 100 ms inter-chunk delay (#54)
- Message deduplication ring buffer: SHA-256 keying, 5-minute TTL, configurable window size (#55)
- LLM retry with exponential backoff and ±20% jitter; retries on 429/503/529 and deadline-exceeded (#56)
- Attachment auto-routing to the `pdf_analyze` / `image_analyze` tools (#57)
- Channel-aware response auto-formatting: Slack mrkdwn, Telegram MarkdownV2, WhatsApp, Teams (#58)
- LLM provider fallback chain — `FallbackLLM` wraps multiple providers (#59)
- Config additions: `ChunkingEnabled`, `ChunkingLimits`, `DedupEnabled`, `DedupWindowSize`, `AutoFormatEnabled`, `AutoProcessAttachments` (`GatewayConfig`); `RetryAttempts`, `RetryInitialBackoff`, `FallbackProviders` (`AgentConfig`)

### Changed
- `docs/openclaw-comparison-2026-03.md` updated to v0.10.0 (#60)

## [0.9.0] - 2026-03-18

### Added
- Per-user rate limiting: `internal/gateway/ratelimiter.go` with a lazy per-user `golang.org/x/time/rate` token bucket; `RateLimitConfig` (`enabled`, `requests_per_minute`, `burst`) plus `BUCKTOOTH_RATE_LIMIT_RPM` env override; enforced at the top of `ProcessMessage` (#47)
- BM25 hybrid memory: `internal/memory/bm25.go` (inline scorer, k1=1.5, b=0.75, no external deps) and `internal/memory/hybrid.go` (`HybridStore` blending BM25 with recency); `memory.type: hybrid` + `hybrid_weight` (#48)
- `pdf_analyze` tool via direct Anthropic HTTP (`pdfs-2024-09-25` beta); URL and sandbox-constrained local paths; 32 MB limit (#49)
- `image_analyze` tool via the Anthropic vision API; auto-detects media type; jpeg/png/gif/webp; 5 MB limit (#50)
- Cron scheduler: `internal/cron/scheduler.go` with a `time.Ticker`-based job engine; `CronConfig`/`CronJobConfig`; `GET /cron/jobs`; wired into gateway `Start`/`Stop` (#51)
- Admin API: `GET /admin/memory/{user_id}/flush`, `POST /admin/skills/reload`, `GET /admin/channels/health`; `SetMemoryStore` setter (#52)
- `/system <prompt>` command for a per-user system-prompt override, `/system reset` to clear; `UserPrefs` extended with a `systemPrompts` map; `getOrCreateAgent` honours the override (#53)

### Fixed
- WebSocket `CheckOrigin` now validates against an `AllowedWSOrigins` allowlist (nil = allow-all for dev) instead of accepting every origin (#46)
- WebSocket handler dispatches to the agent router instead of echoing; JSON `wsInboundMsg`/`wsOutboundMsg` wire format (#46)

### Changed
- `tools.FromConfig` signature accepts `AgentConfig` for API-key access

## [0.8.0] - 2026-03-18

### Added
- Agent skills system: `skills.enabled` config + `BUCKTOOTH_SKILLS_PATH` env var; skill instructions injected into every message via `<available_skills>` block; `GET /skills` API endpoint lists loaded skills (#43)
- SQLite persistent memory backend (`memory.type: sqlite`) using pure-Go `modernc.org/sqlite`; default path `~/.bucktooth/memory.db` (#41)
- Async LLM-driven conversation summarization: `memory.summarize_enabled` / `memory.summarize_threshold`; compresses histories above threshold into a single system summary message (#44)
- Prometheus latency histogram `bucktooth_message_duration_seconds`; `bucktooth_messages_total`, `bucktooth_tokens_total` counter vecs; `bucktooth_active_users` gauge (#42)
- Grafana dashboard JSON (`configs/grafana-dashboard.json`) with message rate, p50/p95/p99 latency, token usage, and active-users panels (#42)
- Prometheus alert rules (`configs/alerting-rules.yml`): `HighLatency`, `ChannelUnhealthy`, `HighErrorRate`, `GatewayDown` (#42)
- `bucktooth mcp serve` subcommand: exposes registered tools as an MCP stdio server (#45)
- `bucktooth completion [bash|zsh|fish|powershell]` subcommand (#39)
- Webhook HMAC-SHA256 verification middleware (`internal/channels/webhook`) for Teams and future webhook channels (#40)

### Changed
- Tool registration extracted from `gateway.go` into `internal/tools/from_config.go` (`tools.FromConfig`) — used by both the gateway and `mcp serve` (#45)

## [0.7.0] - 2026-03-18

### Added
- `shell` tool: execute shell commands with optional human-approval gate; two-step confirmation flow (first call returns approval prompt, second call with `approved:true` executes); optional `allowed_commands` allowlist (#33)
- Microsoft Teams channel: Bot Framework REST API webhook at `/channels/teams/messages`; OAuth2 bearer token with automatic refresh; replies via Bot Framework conversations API (#32)

## [0.6.1] - 2026-03-17

### Added
- MCP (Model Context Protocol) client support: connect to `stdio` and `http` MCP servers via the `mcp.servers` config block; tools are discovered at startup and registered into the tool registry (#30)

## [0.6.0] - 2026-03-17

### Added
- Gateway bearer token auth via `gateway.api_token` / `BUCKTOOTH_API_TOKEN` (#26)
- Multi-provider LLM: `openai`, `openai-compatible`, `gemini`, `ollama`, `litellm`, `bedrock` (#27)
- Token/cost tracking: `tokens_in`/`tokens_out` counters in `Stats`, exposed via `/dashboard/data` (#28)
- `web_fetch` tool: HTTP fetch with HTML-to-text extraction via `golang.org/x/net/html` (#29)
- Vector memory backend (`memory.type: vector`) wrapping agenkit-go VectorMemory (#31)

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
