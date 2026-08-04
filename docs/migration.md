# Migration Guide

---

## OpenClaw → BuckTooth v0.4.0

### Feature Comparison

| Feature | OpenClaw | BuckTooth v0.4.0 |
|---------|----------|-----------------|
| Discord | Yes | Yes |
| WhatsApp | No | Yes (whatsmeow) |
| Telegram | No | Yes (long-polling) |
| Slack | No | Yes (Socket Mode) |
| LLM Provider | OpenAI | Anthropic Claude |
| Conversation Memory | Session-only | In-memory or Redis |
| Tool Use | No | Yes (ReAct + Planning) |
| Web Dashboard | No | Yes (port 8080, `/`) |
| OpenTelemetry Tracing | No | Yes (OTLP HTTP) |
| Prometheus Metrics | No | Yes (`/metrics`) |
| Docker | Manual | Multi-stage, distroless |
| Helm Chart | No | Yes (`charts/bucktooth/`) |
| CLI | Single command | cobra (`start`, `status`, `config validate`, `version`) |
| Single Binary | Yes | Yes |

### Environment Variable Mapping

| OpenClaw | BuckTooth | Notes |
|----------|-----------|-------|
| `OPENAI_API_KEY` | `ANTHROPIC_API_KEY` | Anthropic replaces OpenAI by default |
| `DISCORD_TOKEN` | `DISCORD_BOT_TOKEN` | Renamed |
| `PORT` | `LOBSTER_GATEWAY_PORT` | HTTP port override |
| _(none)_ | `TELEGRAM_BOT_TOKEN` | New |
| _(none)_ | `SLACK_BOT_TOKEN` | New |
| _(none)_ | `SLACK_APP_TOKEN` | New (Socket Mode) |

### Step-by-Step Migration

1. **Install BuckTooth**

   ```bash
   git clone https://github.com/scttfrdmn/bucktooth
   cd bucktooth
   make build          # produces bin/bucktooth
   ```

2. **Create configuration file**

   ```bash
   cp configs/gateway.yaml my-gateway.yaml
   ```

3. **Set environment variables**

   ```bash
   export ANTHROPIC_API_KEY=sk-ant-...
   export DISCORD_BOT_TOKEN=...
   ```

4. **Validate configuration**

   ```bash
   ./bin/bucktooth config validate --config my-gateway.yaml
   ```

5. **Start BuckTooth**

   ```bash
   ./bin/bucktooth start --config my-gateway.yaml
   ```

6. **Verify health**

   ```bash
   ./bin/bucktooth status
   ```

7. **Check the dashboard** — open http://localhost:8080

8. **Decommission OpenClaw** once BuckTooth is confirmed healthy.

---

## BuckTooth v0.4.0 → v0.13.0

v0.4.0 → v0.13.0 is a **backward-compatible** upgrade. All existing config keys continue to work. The sections below describe new features and config blocks you can opt into.

### New Channels

| Channel | Version Added | Config key |
|---------|--------------|-----------|
| Microsoft Teams | v0.9.0 | `channels.teams` |
| Signal | v0.13.0 | `channels.signal` |

Add to `gateway.yaml` as needed:

```yaml
channels:
  teams:
    enabled: false
    auth:
      app_id: ${TEAMS_APP_ID}
      app_password: ${TEAMS_APP_PASSWORD}
  signal:
    enabled: false
    auth:
      phone_number: "+1234567890"
      signald_url: "ws://localhost:7583/v1/receive"
```

### Memory Type Upgrades

New persistent backends are available. To switch:

```yaml
# SQLite (no external service required)
memory:
  type: sqlite
  options:
    path: ~/.bucktooth/memory.db

# Hybrid BM25 + semantic search
memory:
  type: hybrid
  options:
    embedding_base_url: https://api.openai.com/v1
    embedding_api_key: ${OPENAI_API_KEY}
    embedding_model: text-embedding-3-small
  hybrid_weight: 0.5        # 0.0 = pure semantic, 1.0 = pure BM25
  decay_enabled: true
  decay_half_life_hours: 24
```

Optional memory summarization (automatically compacts old conversations):

```yaml
memory:
  summarize_enabled: true
  summarize_threshold: 30         # message count trigger
  summarize_token_threshold: 8000 # token count trigger (0 = disabled)
```

### New Config Sections

The following top-level sections were added after v0.4.0. Add them to your config to opt in:

```yaml
# Rate limiting (v0.9.0)
rate_limit:
  enabled: true
  requests_per_minute: 60
  burst: 10

# Cron scheduled messages (v0.9.0)
cron:
  jobs:
    - name: daily-summary
      schedule: 24h
      message: "Summarize today's activity."
      channel_id: discord
      user_id: bot
      enabled: false

# Agent skills (v0.8.0)
skills:
  enabled: false
  search_paths: ["~/.bucktooth/skills", "./skills"]
  dep_check_enabled: false

# MCP tool servers (v0.8.0)
mcp:
  servers: []

# LLM cost tracking (v0.13.0)
observability:
  cost_tracking:
    enabled: false

# LLM retry + fallback (v0.10.0)
agents:
  retry_attempts: 3
  retry_initial_backoff: 500ms
  fallback_providers: []
```

### New API Endpoints

| Endpoint | Version | Description |
|----------|---------|-------------|
| `POST /v1/chat/completions` | v0.11.0 | OpenAI-compatible completions |
| `GET /v1/models` | v0.12.0 | OpenAI-compatible model list |
| `GET /v1/usage` | v0.13.0 | LLM cost and token totals |
| `GET /admin/audit` | v0.14.0 | Security posture report |
| `GET /admin/channels/health` | v0.9.0 | Per-channel health detail |
| `POST /admin/skills/reload` | v0.8.0 | Hot-reload agent skills |
| `GET /cron/jobs` | v0.9.0 | List scheduled jobs |
| `/live`, `/ready` | v0.5.0 | Kubernetes probes |

### Breaking Changes

None. All additions are backward-compatible with v0.4.0 configurations.
