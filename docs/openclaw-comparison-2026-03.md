# BuckTooth vs OpenClaw — Feature Gap Analysis

**Date**: 2026-03-18 (updated 2026-03-18 for v0.13.0)
**BuckTooth version**: v0.13.0
**OpenClaw version**: v2026.3.13 (released 2026-03-14)

OpenClaw ([github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)) is a TypeScript/Node.js
personal AI gateway that went viral in late January 2026, currently at 320K+ GitHub stars. It is the
dominant open-source reference in the space. BuckTooth is a Go re-implementation targeting a different
runtime profile (single binary, low memory, Kubernetes-native).

---

## Quick Reference

| Attribute | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Language | Go | TypeScript / Node.js ≥22 |
| Version | v0.13.0 | v2026.3.13 |
| Binary | Single static binary (~25 MB) | Node.js runtime required |
| License | Apache 2.0 | MIT |
| Helm chart | ✅ | ❌ |
| Distroless image | ✅ | ❌ |

---

## 1. Channels

| Channel | BuckTooth | OpenClaw |
|---------|-----------|---------|
| Discord | ✅ | ✅ |
| Slack (Socket Mode) | ✅ | ✅ |
| Telegram (long-polling) | ✅ | ✅ |
| WhatsApp (whatsmeow) | ✅ | ✅ |
| Microsoft Teams | ✅ v0.9.0 | ✅ |
| Teams HMAC activity validation | ✅ v0.11.0 | N/A |
| Signal | ✅ v0.13.0 | ✅ |
| iMessage (BlueBubbles) | ❌ | ✅ |
| Matrix / IRC / LINE | ❌ | ✅ |
| Mattermost / Nextcloud | ❌ | ✅ |
| Twitch / Nostr / Feishu | ❌ | ✅ |
| Google Chat / Zalo | ❌ | ✅ |

OpenClaw supports 24+ channels. Highest-value remaining gap: **Signal** (private users).

---

## 2. LLM Provider Support

| Provider | agenkit-go | BuckTooth wired |
|----------|-----------|----------------|
| Anthropic Claude | ✅ `adapter/llm/anthropic.go` | ✅ |
| OpenAI / GPT | ✅ `adapter/llm/openai.go` | ✅ v0.6.0 |
| OpenAI-compatible (vLLM, llama.cpp, etc.) | ✅ `adapter/llm/openai_compatible.go` | ✅ v0.6.0 |
| Google Gemini | ✅ `adapter/llm/gemini.go` | ✅ v0.6.0 |
| AWS Bedrock | ✅ `adapter/llm/bedrock.go` | ✅ v0.6.0 |
| Ollama (local models) | ✅ `adapter/llm/ollama.go` | ✅ v0.6.0 |
| LiteLLM proxy | ✅ `adapter/llm/litellm.go` | ✅ v0.6.0 |
| LLM retry / exponential backoff | — | ✅ v0.10.0 |
| LLM provider fallback chain | — | ✅ v0.10.0 |
| Streaming responses (WS token streaming) | ✅ | ✅ v0.11.0 |
| OpenAI-compatible completions endpoint | — | ✅ v0.11.0 |
| OpenAI-compatible GET /v1/models endpoint | — | ✅ v0.12.0 |

---

## 3. Memory

| Capability | agenkit-go | BuckTooth | OpenClaw |
|-----------|-----------|-----------|---------|
| In-memory history | ✅ | ✅ | ✅ |
| Redis-backed persistence | ✅ | ✅ v0.6.0 | ✅ (plugin) |
| Per-user history isolation | — | ✅ v0.5.0 | ✅ |
| Vector store interface | ✅ `VectorStore` | ✅ v0.6.0 | ✅ |
| Semantic search (cosine) | ✅ `MemoryVectorStore` | ✅ v0.6.0 | ✅ |
| Hybrid BM25 + vector | ❌ | ✅ v0.9.0 | ✅ |
| SQLite persistence | — | ✅ v0.8.0 | ❌ |
| Memory summarization (message count) | — | ✅ v0.8.0 | ✅ |
| Token-based compaction trigger | — | ✅ v0.11.0 | ✅ |
| Temporal decay / recency boost | ❌ | ✅ v0.11.0 | ✅ |

---

## 4. Tools

| Tool | BuckTooth | OpenClaw |
|------|-----------|---------|
| Calculator | ✅ | ✅ |
| File system (sandboxed) | ✅ | ✅ |
| Web search (Brave) | ✅ | ✅ (Brave, Firecrawl, Gemini) |
| Web fetch / content extraction | ✅ v0.6.0 | ✅ |
| Message formatter | ✅ | ✅ |
| Calendar | ✅ | ❌ |
| Shell / exec (with approval gate) | ✅ v0.8.0 | ✅ |
| PDF analysis | ✅ v0.9.0 | ✅ |
| Image analysis / generation | ✅ v0.9.0 | ✅ |
| Attachment auto-routing | ✅ v0.10.0 | ✅ |
| Cron / scheduled jobs | ✅ v0.9.0 | ✅ |
| MCP tool bridge | ✅ v0.8.0 | ✅ |
| Browser automation (Chrome CDP) | ✅ v0.12.0 | ✅ |

---

## 5. Reliability & Quality

| Capability | BuckTooth | OpenClaw | RustyNail |
|-----------|-----------|---------|-----------|
| Long-message chunking per platform | ✅ v0.10.0 | ✅ | ✅ |
| Message deduplication (ring buffer) | ✅ v0.10.0 | ✅ | ✅ |
| LLM retry / exponential backoff | ✅ v0.10.0 | ✅ | ✅ |
| Channel-aware markdown formatting | ✅ v0.10.0 | ✅ | ✅ |
| LLM provider fallback chain | ✅ v0.10.0 | ✅ | ✅ |
| Attachment auto-routing to tools | ✅ v0.10.0 | ✅ | ✅ |
| Streaming responses (WS) | ✅ v0.11.0 | ✅ | ✅ |
| OpenAI-compatible completions API | ✅ v0.11.0 | ✅ | ✅ |

---

## 6. Plugin / Skill Ecosystem — ClawHub

ClawHub ([github.com/openclaw/clawhub](https://github.com/openclaw/clawhub)) hosts
13,700+ community skills. Each skill is:

- A `SKILL.md` file (YAML frontmatter + Markdown instructions) declaring metadata and
  dependency requirements (`bins`, `env` vars)
- An **MCP server** — the actual tool mechanism is the
  [Model Context Protocol](https://modelcontextprotocol.io/) (JSON-RPC 2.0 over stdio
  or HTTP/SSE)

BuckTooth has MCP client support (v0.8.0) and a file-based skill system (v0.8.0).
SKILL.md dependency validation was added in v0.12.0 (`dep_check_enabled: true` in `skills` config),
completing the core ClawHub compatibility layer.

---

## 7. Observability

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Prometheus metrics | ✅ | ✅ |
| OpenTelemetry tracing (OTLP) | ✅ | ✅ |
| `/live` + `/ready` probes | ✅ v0.5.0 | ✅ |
| Message statistics + ring buffer | ✅ v0.5.0 | ✅ |
| `/dashboard/data` JSON endpoint | ✅ v0.5.0 | ✅ |
| Token usage / cost tracking | ✅ v0.6.0 | ✅ |
| LLM USD cost tracking + GET /v1/usage | ✅ v0.13.0 | ✅ (ClawMetry) |
| Purpose-built AI metrics dashboard | ❌ | ✅ (ClawMetry) |

---

## 8. Security

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Dashboard Basic auth | ✅ v0.5.0 | ✅ |
| Gateway API bearer token | ✅ v0.6.0 | ✅ (required by default) |
| Webhook HMAC verification | ✅ v0.8.0 | ✅ |
| Teams HMAC activity validation | ✅ v0.11.0 | N/A |
| Rate limiting (per-user token bucket) | ✅ v0.9.0 | ✅ |
| Message deduplication | ✅ v0.10.0 | ✅ |
| Device pairing with approval | ❌ | ✅ |
| Exec approval gates | ❌ | ✅ |
| Filesystem sandboxing | ✅ (partial) | ✅ (Docker containment) |
| Security audit command | ❌ | ✅ |

---

## 9. Deployment

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Single static binary | ✅ (~25 MB) | ❌ |
| Docker / docker-compose | ✅ | ✅ |
| Helm chart | ✅ | ❌ |
| Distroless/minimal image | ✅ | ❌ |
| Test harness (zero-credential CI) | ✅ v0.4.5 | partial |
| GitHub Actions CI/CD | ✅ v0.13.0 | ✅ |
| Admin API | ✅ v0.9.0 | ✅ |
| Tailscale / public tunnel | ❌ | ✅ |
| Remote device pairing | ❌ | ✅ |
| Companion macOS/iOS/Android apps | ❌ | ✅ |
| Voice Wake / Talk Mode | ❌ | ✅ |

BuckTooth's deployment story is stronger for Kubernetes and CI environments.
Companion apps and voice are out of scope for a Go gateway.

---

## Priority Matrix (post-v0.12.0)

### Shipped in v0.12.0
1. ✅ **ClawHub SKILL.md dep validation** — `dep_check_enabled: true` in `skills` config
2. ✅ **Browser automation (Chrome CDP)** — `browser_enabled: true` in `tools` config
3. ✅ **GET /v1/models endpoint** — OpenAI-compatible model listing

### Shipped in v0.13.0
4. ✅ **Signal channel** — via signal-cli JSON-RPC WebSocket daemon (`signald_url` in channel auth)
5. ✅ **LLM cost tracking** — `cost_tracking.enabled: true` in `observability` config; exposes `GET /v1/usage`
6. ✅ **GitHub Actions CI/CD** — `.github/workflows/ci.yml` (push/PR) + `release.yml` (tag → release)
7. ✅ **Harness integration test expansion** — `/v1/models`, `/admin/skills/deps`, `/v1/usage`, Signal unavailability

### Larger investments — v0.14.x+
- **Device pairing approval flow** — Signal device pairing, exec approval gates
- **ClawMetry equivalent** — purpose-built observability dashboard
- iMessage/Matrix channels

### Out of scope for BuckTooth core
- Companion macOS/iOS/Android native apps
- Voice Wake / Talk Mode

---

## agenkit-go Issues Opened

| Issue | Title | Impact |
|-------|-------|--------|
| [scttfrdmn/agenkit#546](https://github.com/scttfrdmn/agenkit/issues/546) | MCP protocol client + server support | ClawHub compatibility, ecosystem |
| [scttfrdmn/agenkit#547](https://github.com/scttfrdmn/agenkit/issues/547) | `NewAnthropicLLM` base URL option | `ANTHROPIC_API_BASE` / proxy support |
