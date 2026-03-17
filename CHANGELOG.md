# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/scttfrdmn/bucktooth/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/bucktooth/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/bucktooth/releases/tag/v0.1.0
