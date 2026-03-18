# Migration Guide: OpenClaw → BuckTooth v0.4.0

## Feature Comparison

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

---

## Configuration Mapping

### Environment Variables

| OpenClaw | BuckTooth | Notes |
|----------|-----------|-------|
| `OPENAI_API_KEY` | `ANTHROPIC_API_KEY` | Anthropic replaces OpenAI |
| `DISCORD_TOKEN` | `DISCORD_BOT_TOKEN` | Renamed |
| `PORT` | `LOBSTER_GATEWAY_PORT` | HTTP port override |
| _(none)_ | `TELEGRAM_BOT_TOKEN` | New |
| _(none)_ | `SLACK_BOT_TOKEN` | New |
| _(none)_ | `SLACK_APP_TOKEN` | New (Socket Mode) |

### Configuration File (`configs/gateway.yaml`)

BuckTooth uses a YAML config file instead of environment-only configuration.

```yaml
# Minimal equivalent of a basic OpenClaw setup
gateway:
  http_port: 8080
  websocket_port: 18789
  log_level: info

channels:
  discord:
    enabled: true
    auth:
      token: ${DISCORD_BOT_TOKEN}

agents:
  llm_provider: anthropic
  llm_model: claude-sonnet-4-5-20250220
  api_key: ${ANTHROPIC_API_KEY}
  max_history: 20
  mode: react        # conversational | react | planning

memory:
  type: inmemory     # upgrade to "redis" for persistence
```

### Memory Store Upgrade

To enable persistent conversation memory, switch from `inmemory` to `redis`:

```yaml
memory:
  type: redis
  options:
    addr: localhost:6379   # or "redis:6379" in docker-compose
    password: ""
    db: 0
    ttl: 24h
    max_history: 50
```

---

## Step-by-Step Migration

1. **Install BuckTooth**

   ```bash
   # From source
   git clone https://github.com/scttfrdmn/bucktooth
   cd bucktooth
   make build          # produces bin/bucktooth
   ```

2. **Create configuration file**

   ```bash
   cp configs/gateway.yaml my-gateway.yaml
   # Edit my-gateway.yaml with your settings
   ```

3. **Set environment variables**

   ```bash
   export ANTHROPIC_API_KEY=sk-ant-...
   export DISCORD_BOT_TOKEN=...
   ```

4. **Validate configuration**

   ```bash
   ./bin/bucktooth config validate --config my-gateway.yaml
   # Expected: "configuration is valid"
   ```

5. **Start BuckTooth**

   ```bash
   ./bin/bucktooth start --config my-gateway.yaml
   ```

6. **Verify health**

   ```bash
   ./bin/bucktooth status
   # Returns JSON health response; exit 0 if healthy
   ```

7. **Check the dashboard**

   Open http://localhost:8080 in your browser. The live feed shows
   messages and channel status in real time.

8. **Decommission OpenClaw**

   Once BuckTooth is confirmed healthy:
   - Stop the OpenClaw process.
   - Update any reverse-proxy or DNS entries to point to BuckTooth's port 8080.
   - Optionally run both side-by-side during a transition window before cutting over.
