# BuckTooth vs OpenClaw — Feature Gap Analysis

**Date**: 2026-03-17
**BuckTooth version**: v0.5.0
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
| Version | v0.5.0 | v2026.3.13 |
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
| Microsoft Teams | ❌ | ✅ |
| Signal | ❌ | ✅ |
| iMessage (BlueBubbles) | ❌ | ✅ |
| Matrix / IRC / LINE | ❌ | ✅ |
| Mattermost / Nextcloud | ❌ | ✅ |
| Twitch / Nostr / Feishu | ❌ | ✅ |
| Google Chat / Zalo | ❌ | ✅ |

OpenClaw supports 24+ channels. The highest-value gaps are **Teams** (enterprise) and
**Signal** (private users).

---

## 2. LLM Provider Support

agenkit-go already implements all major providers. BuckTooth's `router.go` currently
wires only `anthropic` and `stub`. The remaining providers are available and need to
be connected.

| Provider | agenkit-go | BuckTooth wired |
|----------|-----------|----------------|
| Anthropic Claude | ✅ `adapter/llm/anthropic.go` | ✅ |
| OpenAI / GPT | ✅ `adapter/llm/openai.go` | ❌ |
| OpenAI-compatible (vLLM, llama.cpp, etc.) | ✅ `adapter/llm/openai_compatible.go` | ❌ |
| Google Gemini | ✅ `adapter/llm/gemini.go` | ❌ |
| AWS Bedrock | ✅ `adapter/llm/bedrock.go` | ❌ |
| Ollama (local models) | ✅ `adapter/llm/ollama.go` | ❌ |
| LiteLLM proxy | ✅ `adapter/llm/litellm.go` | ❌ |

**Action (BuckTooth)**: Expand `NewAgentRouter` switch to cover additional
`llm_provider` values matching agenkit-go adapter names. No agenkit-go work needed.

**Note on `api_base`**: `AnthropicLLM` has an internal `baseURL` field but
`NewAnthropicLLM` does not expose it as a parameter. An agenkit-go issue has been
opened to add `NewAnthropicLLMWithOptions` (see agenkit #TBD).

---

## 3. Memory

agenkit-go already has `memory/vector_memory.go` with a full `VectorMemory` type,
`EmbeddingProvider` interface, `VectorStore` interface, and `MemoryVectorStore`
(in-memory cosine similarity). BuckTooth needs a new `memory/vector` backend that
wraps this.

| Capability | agenkit-go | BuckTooth | OpenClaw |
|-----------|-----------|-----------|---------|
| In-memory history | ✅ | ✅ | ✅ |
| Redis-backed persistence | ✅ | ✅ | ✅ (plugin) |
| Per-user history isolation | — | ✅ v0.5.0 | ✅ |
| Vector store interface | ✅ `VectorStore` | ❌ not wired | ✅ |
| Semantic search (cosine) | ✅ `MemoryVectorStore` | ❌ not wired | ✅ |
| Hybrid BM25 + vector | ❌ | ❌ | ✅ |
| Temporal decay / recency boost | ❌ | ❌ | ✅ |
| Automatic context compaction | ❌ | ❌ | ✅ |

**Action (BuckTooth)**: Add `memory/vector` type to config; wire `VectorMemory` from
agenkit-go with an `EmbeddingProvider` (Anthropic embeddings API or OpenAI-compatible).

---

## 4. Tools

| Tool | BuckTooth | OpenClaw |
|------|-----------|---------|
| Calculator | ✅ | ✅ |
| File system (sandboxed) | ✅ | ✅ |
| Web search (Brave) | ✅ | ✅ (Brave, Firecrawl, Gemini) |
| Message formatter | ✅ | ✅ |
| Calendar | ✅ | ❌ |
| Web fetch / content extraction | ❌ | ✅ |
| Shell / exec (with approval gate) | ❌ | ✅ |
| Browser automation (Chrome CDP) | ❌ | ✅ |
| PDF analysis | ❌ | ✅ |
| Image analysis / generation | ❌ | ✅ |
| Cron / scheduled jobs | ❌ | ✅ |
| MCP tool bridge | ❌ | ✅ |

**Quickest wins**: web fetch (low deps, complements existing web search), cron/scheduling.

---

## 5. Plugin / Skill Ecosystem — ClawHub

ClawHub ([github.com/openclaw/clawhub](https://github.com/openclaw/clawhub)) hosts
13,700+ community skills. Each skill is:

- A `SKILL.md` file (YAML frontmatter + Markdown instructions) declaring metadata and
  dependency requirements (`bins`, `env` vars)
- An **MCP server** — the actual tool mechanism is the
  [Model Context Protocol](https://modelcontextprotocol.io/) (JSON-RPC 2.0 over stdio
  or HTTP/SSE)

**ClawHub compatibility = MCP client support.** If BuckTooth (via agenkit-go) can
act as an MCP client, it can connect to any ClawHub skill's MCP server and expose its
tools through the existing ReActAgent pipeline. The SKILL.md metadata layer
(dependency checking, env-var requirements) is lightweight to implement separately
in BuckTooth.

agenkit-go currently has no MCP support (`protocols/agui` is the only protocol
implementation). An issue has been filed on agenkit-go to add MCP client + server
support (see below).

---

## 6. Observability

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Prometheus metrics | ✅ | ✅ |
| OpenTelemetry tracing (OTLP) | ✅ | ✅ |
| `/live` + `/ready` probes | ✅ v0.5.0 | ✅ |
| Message statistics + ring buffer | ✅ v0.5.0 | ✅ |
| `/dashboard/data` JSON endpoint | ✅ v0.5.0 | ✅ |
| Token usage / cost tracking | ❌ | ✅ |
| Purpose-built AI metrics dashboard | ❌ | ✅ (ClawMetry) |

Token/cost tracking requires surfacing `InputTokens`/`OutputTokens` from `anthropicUsage`
(already in the `AnthropicLLM` response metadata) through to the gateway stats layer.

---

## 7. Security

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Dashboard Basic auth | ✅ v0.5.0 | ✅ |
| Gateway API bearer token | ❌ | ✅ (required by default) |
| Device pairing with approval | ❌ | ✅ |
| Exec approval gates | ❌ | ✅ |
| Filesystem sandboxing | ✅ (partial) | ✅ (Docker containment) |
| Security audit command | ❌ | ✅ |

**Gateway bearer token auth is a production blocker.** The HTTP API and WebSocket
server currently have no authentication. This is the highest-priority security gap.

---

## 8. Deployment

| Capability | BuckTooth | OpenClaw |
|-----------|-----------|---------|
| Single static binary | ✅ (~25 MB) | ❌ |
| Docker / docker-compose | ✅ | ✅ |
| Helm chart | ✅ | ❌ |
| Distroless/minimal image | ✅ | ❌ |
| Test harness (zero-credential CI) | ✅ v0.4.5 | partial |
| Tailscale / public tunnel | ❌ | ✅ |
| Remote device pairing | ❌ | ✅ |
| Companion macOS/iOS/Android apps | ❌ | ✅ |
| Voice Wake / Talk Mode | ❌ | ✅ |

**BuckTooth's deployment story is stronger** for Kubernetes and CI environments.
Companion apps and voice are out of scope for a Go gateway.

---

## Priority Matrix

### Immediate (production-blocker or one-day effort)
1. **Gateway bearer token auth** — no auth on HTTP/WS is a production blocker
2. **Multi-provider LLM in router.go** — agenkit-go already has the adapters; just wiring
3. **Token/cost tracking** — metadata is already in Anthropic responses; surface it
4. **Web fetch tool** — single-dep addition, complements existing web search

### Medium effort, high ROI
5. **MCP client support in agenkit-go** — unlocks ClawHub ecosystem; tracked in agenkit issue
6. **Vector memory backend** — agenkit-go has `VectorMemory`; needs wiring + embedding provider config
7. **`AnthropicLLM` base URL option** — tracked in agenkit issue; trivial once merged

### Larger investments
8. **Microsoft Teams channel** — highest-value missing channel for enterprise
9. **Shell/exec tool with approval gate** — requires HITL pattern (already in agenkit-go `patterns/human_in_loop.go`)
10. **Signal channel** — libsignal complexity

### Out of scope for BuckTooth core
- Companion macOS/iOS/Android native apps
- Voice Wake / Talk Mode
- ClawMetry equivalent dashboard product

---

## agenkit-go Issues Opened

| Issue | Title | Impact |
|-------|-------|--------|
| [scttfrdmn/agenkit#546](https://github.com/scttfrdmn/agenkit/issues/546) | MCP protocol client + server support | ClawHub compatibility, ecosystem |
| [scttfrdmn/agenkit#547](https://github.com/scttfrdmn/agenkit/issues/547) | `NewAnthropicLLM` base URL option | `ANTHROPIC_API_BASE` / proxy support |
