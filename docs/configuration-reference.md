# Configuration Reference

All configuration lives in a single YAML file (default: `configs/gateway.yaml`).

Values can reference environment variables using `${VAR_NAME}` syntax. BuckTooth also loads
a `.env` file from the working directory on startup.

---

## `gateway`

Core server settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `http_port` | int | `8080` | HTTP API / dashboard server port |
| `websocket_port` | int | `18789` | WebSocket streaming server port |
| `log_level` | string | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `shutdown_timeout` | duration | `30s` | Graceful shutdown timeout |
| `dashboard_auth_password` | string | `""` | Enables Basic auth on `/` and `/dashboard/*`; env: `DASHBOARD_AUTH_PASSWORD` |
| `api_token` | string | `""` | Enables Bearer token auth on all non-probe routes; env: `BUCKTOOTH_API_TOKEN` |
| `allowed_ws_origins` | []string | `[]` | Allowed WebSocket origins (empty = allow all) |
| `streaming_enabled` | bool | `false` | Enable token-by-token streaming to WebSocket clients |
| `chunking_enabled` | bool | `true` | Split long responses into platform-appropriate chunks |
| `chunking_limits` | map[string]int | `{}` | Per-channel-type character limits (overrides built-in defaults) |
| `dedup_enabled` | bool | `true` | Drop duplicate inbound messages via ring buffer |
| `dedup_window_size` | int | `256` | Ring buffer size for deduplication |
| `auto_format_enabled` | bool | `true` | Apply channel-aware Markdown formatting |
| `auto_process_attachments` | bool | `true` | Auto-route file attachments to PDF/image tools |
| `test_channel` | bool | `false` | Enable in-process test channel (harness mode) |

---

## `channels`

Each channel is keyed by its platform name. All channels share the same sub-structure.

### Common fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Whether to start this channel |
| `auth` | map | `{}` | Platform-specific auth credentials (see per-channel docs) |
| `options` | map | `{}` | Platform-specific options |

### `channels.discord`

| Auth key | Description |
|----------|-------------|
| `token` | Discord bot token; env: `DISCORD_BOT_TOKEN` |

### `channels.telegram`

| Auth key | Description |
|----------|-------------|
| `token` | Telegram bot token from BotFather; env: `TELEGRAM_BOT_TOKEN` |

### `channels.whatsapp`

| Auth key | Description |
|----------|-------------|
| `session_file` | Path to the whatsmeow session database |

| Option key | Type | Default | Description |
|------------|------|---------|-------------|
| `groups` | bool | `false` | Respond to group messages |

### `channels.slack`

| Auth key | Description |
|----------|-------------|
| `bot_token` | Bot OAuth token (`xoxb-…`); env: `SLACK_BOT_TOKEN` |
| `app_token` | App-level Socket Mode token (`xapp-…`); env: `SLACK_APP_TOKEN` |

### `channels.teams`

| Auth key | Description |
|----------|-------------|
| `app_id` | Azure AD Application (client) ID; env: `TEAMS_APP_ID` |
| `app_password` | Azure AD client secret; env: `TEAMS_APP_PASSWORD` |

| Option key | Type | Default | Description |
|------------|------|---------|-------------|
| `webhook_path` | string | `/channels/teams/activity` | HTTP path for Bot Framework activity endpoint |

### `channels.signal`

| Auth key | Description |
|----------|-------------|
| `phone_number` | Registered Signal phone number (E.164 format) |
| `signald_url` | signal-cli JSON-RPC WebSocket URL |

---

## `agents`

LLM and agent behaviour settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `llm_provider` | string | `anthropic` | LLM backend: `anthropic`, `openai`, `openai-compatible`, `gemini`, `ollama`, `litellm`, `bedrock`, `stub` |
| `llm_model` | string | `claude-sonnet-4-5-20250220` | Model name passed to the provider |
| `api_key` | string | `""` | API key for the LLM provider; env: `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` |
| `api_base` | string | `""` | Override the provider's base URL (proxies, local models); env: `ANTHROPIC_API_BASE` |
| `max_history` | int | `20` | Maximum conversation turns kept in memory |
| `temperature` | float | `0.7` | LLM sampling temperature |
| `max_tokens` | int | `4096` | Maximum tokens per LLM response |
| `mode` | string | `react` | Agent pattern: `conversational`, `react` (tool use), `planning` |
| `retry_attempts` | int | `3` | Transient LLM error retries (0 = disabled) |
| `retry_initial_backoff` | duration | `500ms` | Initial backoff for LLM retries |
| `fallback_providers` | []AgentConfig | `[]` | Ordered list of fallback providers tried when primary fails |
| `stub_response` | string | `""` | When provider is `stub`: fixed response (empty = echo mode) |

---

## `tools`

Each tool has an `enabled` flag and an optional `options` map.

### `tools.calculator`

No options. Performs arithmetic expressions.

### `tools.message`

No options. Sends messages to other channels (cross-channel messaging).

### `tools.filesystem`

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `sandbox_dir` | string | `~/bucktooth-files` | All file operations are restricted to this directory |
| `max_file_size` | int | `10485760` | Maximum file size in bytes (10 MB) |

### `tools.websearch`

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `api_key` | string | `""` | Brave Search API key; env: `BRAVE_SEARCH_API_KEY` |
| `max_results` | int | `5` | Maximum search results returned |

### `tools.webfetch`

No options. Fetches and extracts text content from URLs.

### `tools.calendar`

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `store_path` | string | `~/.bucktooth/calendar.json` | Path to the local calendar JSON store |

### `tools.shell`

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `require_approval` | bool | `true` | Prompt the user for confirmation before running any command |
| `allowed_commands` | []string | `[]` | Allowlist of command names (empty = all commands permitted) |

### `tools.pdf`

No options. Extracts text from uploaded PDF files.

### `tools.image`

No options. Analyzes uploaded images via the LLM's vision capability.

### `tools.browser_enabled`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `browser_enabled` | bool | `false` | Enable Chrome DevTools Protocol browser automation |
| `browser_timeout` | int | `30` | Per-action timeout in seconds |

---

## `memory`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | `inmemory` | Backend: `inmemory`, `redis`, `sqlite`, `vector`, `hybrid` |
| `options` | map | `{}` | Backend-specific options (see below) |
| `summarize_enabled` | bool | `false` | Summarize old messages when history exceeds threshold |
| `summarize_threshold` | int | `30` | Message count that triggers summarization |
| `summarize_token_threshold` | int | `0` | Token count that triggers summarization (0 = disabled) |
| `hybrid_weight` | float | `0.5` | BM25 vs. semantic weight for `hybrid` backend (0.0 = pure semantic, 1.0 = pure BM25) |
| `decay_enabled` | bool | `false` | Apply exponential recency decay to memory scores |
| `decay_half_life_hours` | float | `24.0` | Half-life for recency decay in hours |

### Redis options (`type: redis`)

| Key | Default | Description |
|-----|---------|-------------|
| `addr` | `localhost:6379` | Redis address |
| `password` | `""` | Redis password |
| `db` | `0` | Redis database index |
| `ttl` | `24h` | Conversation TTL |
| `max_history` | `50` | Maximum messages per conversation |

### SQLite options (`type: sqlite`)

| Key | Default | Description |
|-----|---------|-------------|
| `path` | `~/.bucktooth/memory.db` | Database file path |
| `max_history` | `50` | Maximum messages per conversation |

### Vector / hybrid options

| Key | Default | Description |
|-----|---------|-------------|
| `embedding_base_url` | `""` | OpenAI-compatible embeddings endpoint |
| `embedding_api_key` | `""` | API key for embeddings endpoint |
| `embedding_model` | `""` | Embedding model name |

---

## `skills`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable the agent skills system |
| `search_paths` | []string | `["~/.bucktooth/skills", "./skills"]` | Directories scanned for `SKILL.md` files |
| `max_active_skills` | int | `3` | Maximum skills injected into a single agent prompt |
| `dep_check_enabled` | bool | `false` | Validate `SKILL.md` dependency declarations before injecting |

---

## `observability`

### `observability.metrics`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Expose Prometheus metrics |
| `port` | int | `8080` | Port for metrics (shares the HTTP server) |
| `path` | string | `/metrics` | URL path for the metrics endpoint |

### `observability.tracing`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable OpenTelemetry tracing |
| `endpoint` | string | `""` | OTLP HTTP endpoint (e.g. `localhost:4318`) — no `http://` prefix |
| `sample_rate` | float | `0.1` | Fraction of traces to sample (0.0–1.0) |
| `service_name` | string | `bucktooth` | Service name tag in traces |

### `observability.cost_tracking`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Track per-model USD cost; exposes `GET /v1/usage` |

---

## `rate_limit`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable per-user token-bucket rate limiting |
| `requests_per_minute` | int | `60` | Sustained request rate per user |
| `burst` | int | `10` | Burst capacity per user |

---

## `cron`

```yaml
cron:
  jobs:
    - name: daily-summary
      schedule: 24h
      message: "Please summarize yesterday's activity."
      channel_id: discord
      user_id: bot
      enabled: true
```

| Key | Type | Description |
|-----|------|-------------|
| `name` | string | Job label (shown in `GET /cron/jobs`) |
| `schedule` | duration | How often to trigger (e.g. `5m`, `1h`, `24h`) |
| `message` | string | Message text to inject |
| `channel_id` | string | Target channel ID |
| `user_id` | string | User ID for the injected message |
| `enabled` | bool | Whether the job is active |

---

## `mcp`

Connects to external [Model Context Protocol](https://modelcontextprotocol.io/) servers and registers their tools.

```yaml
mcp:
  servers:
    - name: my-server
      type: stdio          # or "http"
      command: /usr/local/bin/my-mcp-server
      args: ["--flag"]
      env: []
```

| Key | Type | Description |
|-----|------|-------------|
| `name` | string | Friendly label (used in logs) |
| `type` | string | `stdio` (spawn subprocess) or `http` |
| `command` | string | Executable path (stdio only) |
| `args` | []string | Arguments to pass (stdio only) |
| `env` | []string | Extra `KEY=VALUE` environment variables (stdio only) |
| `url` | string | Base URL of the MCP HTTP server (http only) |

---

## Environment variable overrides

The following environment variables override their corresponding config keys:

| Env var | Config key |
|---------|-----------|
| `LOBSTER_GATEWAY_PORT` | `gateway.http_port` |
| `LOBSTER_WEBSOCKET_PORT` | `gateway.websocket_port` |
| `LOBSTER_LOG_LEVEL` | `gateway.log_level` |
| `BUCKTOOTH_API_TOKEN` | `gateway.api_token` |
| `DASHBOARD_AUTH_PASSWORD` | `gateway.dashboard_auth_password` |
| `ANTHROPIC_API_KEY` | `agents.api_key` (when provider = anthropic) |
| `ANTHROPIC_API_BASE` | `agents.api_base` |
| `OPENAI_API_KEY` | `agents.api_key` (when provider = openai) |
| `DISCORD_BOT_TOKEN` | `channels.discord.auth.token` |
| `TELEGRAM_BOT_TOKEN` | `channels.telegram.auth.token` |
| `SLACK_BOT_TOKEN` | `channels.slack.auth.bot_token` |
| `SLACK_APP_TOKEN` | `channels.slack.auth.app_token` |
| `TEAMS_APP_ID` | `channels.teams.auth.app_id` |
| `TEAMS_APP_PASSWORD` | `channels.teams.auth.app_password` |
| `BRAVE_SEARCH_API_KEY` | `tools.websearch.options.api_key` |
