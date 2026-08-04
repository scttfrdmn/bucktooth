# BuckTooth vs OpenClaw — Feature Gap Analysis

**Date**: 2026-03-18 (updated 2026-08-04 — §10 and §11 added for planned work)
**BuckTooth version**: v0.14.0
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
| Version | v0.14.0 | v2026.3.13 |
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
| Exec approval gates | ✅ v0.8.0 | ✅ |
| Filesystem sandboxing | ✅ (partial) | ✅ (Docker containment) |
| Security audit endpoint | ✅ v0.14.0 | ✅ |

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
| Multi-arch Docker (amd64+arm64) + GHCR | ✅ v0.14.0 | ✅ |
| Helm chart current | ✅ v0.14.0 | ❌ |
| User documentation | ✅ v0.14.0 | ✅ |
| Admin API | ✅ v0.9.0 | ✅ |
| Tailscale / public tunnel | ❌ | ✅ |
| Remote device pairing | ❌ | ✅ |
| Companion macOS/iOS/Android apps | ❌ | ✅ |
| Voice Wake / Talk Mode | ❌ | ✅ |

BuckTooth's deployment story is stronger for Kubernetes and CI environments.
Companion apps and voice are out of scope for a Go gateway.

---

## 10. Signed & Sandboxed Skills — planned, no sibling equivalent

This is the one area where BuckTooth is not tracking a gap but opening a lead, so it is
recorded separately from the parity tables above.

| Capability | BuckTooth | RustyNail | OpenClaw / ClawHub |
|-----------|-----------|-----------|--------------------|
| Skill format | `SKILL.md` + MCP; **Wasm component planned** | `SKILL.md` | `SKILL.md` + MCP server |
| Skill sandboxing | ❌ (dep *check* only) | ❌ | ❌ (host process authority) |
| Capability manifest | 🚧 planned | ❌ | ❌ |
| Skill signing / verification | 🚧 planned | 🚧 planned (twin) | ❌ |
| Identity-pinned verification | 🚧 planned | 🚧 planned (twin) | ❌ |
| OCI artifact distribution | 🚧 planned | 🚧 planned (twin) | ❌ (git/npm) |
| Policy-gated capabilities | 🚧 planned | 🚧 planned (twin) | ❌ |

ClawHub's 13,700+ skills are unsigned and unsandboxed: a `SKILL.md` plus an MCP server
running with the full authority of the host process. BuckTooth's v0.12.0 dependency
validation (§6) reached parity with that model, but a dependency *check* is not a
boundary — confirming `curl` exists does not constrain what a skill does with it. The
nearest containment row in this document is "Filesystem sandboxing | ✅ (partial) |
✅ (Docker containment)" (§8), and neither project's answer there is strong.

The [Signed & Sandboxed Skills](https://github.com/scttfrdmn/bucktooth/issues/75)
milestone replaces the model rather than extending it: skills become Wasm components
whose reachable syscalls are exactly the host functions the runtime linked, and that
linking is derived from a signed, policy-evaluated capability manifest. Runtime is
wazero here and wasmtime in RustyNail; the artifact format, manifest schema, attestation
predicate, and verification rules are shared so artifacts published by either twin verify
under both.

Because this ships in both twins at once, it is not a parity item in either direction.
Update this section as issues #76–#84 land.

---

## 11. Bounded Decomposition — quarry as a host capability, planned

The second lead, and likewise recorded outside the parity tables because no sibling has
an equivalent.

| Capability | BuckTooth | RustyNail | OpenClaw |
|-----------|-----------|-----------|---------|
| Cost tracking (after the fact) | ✅ v0.13.0 (`GET /v1/usage`) | ✅ | ✅ (ClawMetry) |
| Per-request spend cap enforced before spend | 🚧 planned | 🚧 planned (twin) | ❌ |
| Bounded recursive decomposition | 🚧 planned | 🚧 planned (twin) | ❌ |
| Cost receipt returned with the answer | 🚧 planned | 🚧 planned (twin) | ❌ |
| Stability / provenance estimate on an answer | 🚧 planned | 🚧 planned (twin) | ❌ |
| Pre-spend plan disclosure + approval | 🚧 planned | 🚧 planned (twin) | ❌ |
| Scope-isolated sub-answer cache | 🚧 planned | 🚧 planned (twin) | ❌ |

The nearest thing either project has today is cost *tracking* — accounting after the
money is gone (§7). Neither can accept "research this, spend at most $5, done by
tonight" as a constraint, and neither returns an answer with a receipt saying what it
cost, which cap bound it, and which claims are unstable.

The [Quarry Host](https://github.com/scttfrdmn/bucktooth/issues/85) milestone integrates
[quarry](https://github.com/scttfrdmn/quarry) as a subprocess host capability. The
gateway becomes quarry's `Provider` chokepoint via its own OpenAI-compatible endpoint, so
quarry rides existing provider config, metering, and fallback chains rather than holding
credentials of its own. The boundary is subprocess-per-run in **both** twins — deliberately
not an in-process import even in the Go twin, where quarry is directly importable, so the
two share exactly one contract.

Two things worth recording as constraints rather than choices:

- quarry's `RunEvent` stream is a **post-hoc fold of a completed run**, not a live feed,
  and carries cost and trust but not tree *shape*. Live decomposition-tree rendering is
  therefore deferred pending upstream work, not merely unbuilt.
- Cache isolation between senders (quarry P6) is the **host's** responsibility: quarry can
  only isolate along the `Scope` it is handed, so minting a per-sender scope is a
  correctness requirement, not a hardening nicety.

Shared `RunEvent` fixtures (#93) hold both twins to bit-identical handling of the same
input, which makes this the cleanest Go-vs-Rust comparison surface the experiment has.
Update this section as issues #86–#93 land.

---

## Priority Matrix (post-v0.14.0)

### Shipped in v0.12.0
1. ✅ **ClawHub SKILL.md dep validation** — `dep_check_enabled: true` in `skills` config
2. ✅ **Browser automation (Chrome CDP)** — `browser_enabled: true` in `tools` config
3. ✅ **GET /v1/models endpoint** — OpenAI-compatible model listing

### Shipped in v0.13.0
4. ✅ **Signal channel** — via signal-cli JSON-RPC WebSocket daemon (`signald_url` in channel auth)
5. ✅ **LLM cost tracking** — `cost_tracking.enabled: true` in `observability` config; exposes `GET /v1/usage`
6. ✅ **GitHub Actions CI/CD** — `.github/workflows/ci.yml` (push/PR) + `release.yml` (tag → release)
7. ✅ **Harness integration test expansion** — `/v1/models`, `/admin/skills/deps`, `/v1/usage`, Signal unavailability

### Shipped in v0.14.0
8. ✅ **User documentation** — getting-started guide, configuration reference, per-channel setup, architecture update
9. ✅ **Multi-arch Docker + GHCR** — release workflow builds linux/amd64 + linux/arm64, pushes to `ghcr.io/scttfrdmn/bucktooth`
10. ✅ **Helm chart updated to v0.13.0** — Teams, Signal, cost tracking, all current tools and options
11. ✅ **Security audit endpoint** — `GET /admin/audit` returns scored security posture report
12. ✅ **Exec approval gates documented** — shell tool `require_approval=true` is the default; now documented and surfaced in audit

### In flight — not parity work

Both are leads rather than gaps, so they sit outside the numbered parity sections and
outside the priority ranking, which orders work by parity value:

- **[Signed & Sandboxed Skills](https://github.com/scttfrdmn/bucktooth/issues/75)** (§10) —
  Wasm skills, capability manifests, OCI distribution, identity-pinned verification (#76–#84)
- **[Quarry Host](https://github.com/scttfrdmn/bucktooth/issues/85)** (§11) —
  bounded decomposition under sender-supplied caps, with a cost receipt (#86–#93)

Both ship in RustyNail simultaneously via mirrored issue sets, so neither moves the
BuckTooth-vs-RustyNail comparison.

### Larger investments — later
- **Device pairing approval flow** — Signal device pairing UI
- **ClawMetry equivalent** — purpose-built AI observability dashboard
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
