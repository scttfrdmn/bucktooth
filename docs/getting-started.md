# Getting Started with BuckTooth

This guide walks you through installing BuckTooth, connecting your first messaging channel, and sending your first AI-powered message.

---

## 1. Prerequisites

| Requirement | Version / Notes |
|-------------|-----------------|
| Go | 1.25 or higher |
| An LLM API key | Anthropic API key (default) or OpenAI |
| A messaging platform bot token | At least one channel (Discord is the easiest to start with) |

---

## 2. Install

### From source

```bash
git clone https://github.com/scttfrdmn/bucktooth
cd bucktooth
make build          # produces bin/bucktooth
```

### Using `go install`

```bash
go install github.com/scttfrdmn/bucktooth/cmd/bucktooth@latest
```

---

## 3. Create a minimal configuration

Copy the example config and edit it:

```bash
cp configs/gateway.yaml my-gateway.yaml
```

Edit `my-gateway.yaml` to enable at least one channel. The minimal working config for Discord looks like:

```yaml
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
  mode: react

memory:
  type: inmemory
```

Set required environment variables:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export DISCORD_BOT_TOKEN=...
```

Or copy `.env.example` to `.env` and fill in your values — BuckTooth loads `.env` automatically on startup.

---

## 4. Validate the configuration

```bash
./bin/bucktooth config validate --config my-gateway.yaml
# Expected output: "configuration is valid"
```

---

## 5. Start BuckTooth

```bash
./bin/bucktooth start --config my-gateway.yaml
```

You should see log output like:

```
{"level":"info","component":"gateway","message":"starting gateway"}
{"level":"info","component":"http","port":8080,"message":"starting HTTP server"}
{"level":"info","component":"channel","channel":"discord","message":"channel started"}
{"level":"info","component":"gateway","message":"gateway started successfully"}
```

---

## 6. Verify

In a new terminal:

```bash
# Check health
curl -s localhost:8080/health | jq .
```

Expected response:

```json
{
  "status": "healthy",
  "channels": [{"name": "discord", "healthy": true, "status": "connected"}]
}
```

Open the web dashboard at **http://localhost:8080** to see real-time event feed, channel status, and message statistics.

---

## 7. Send your first message

1. Invite your Discord bot to a server (see [docs/channels.md](channels.md#discord) for the invite URL).
2. In any channel the bot has access to, send a message mentioning the bot or DM it directly.
3. The bot will reply using Claude.

---

## 8. Next steps

- **Add more channels**: See [docs/channels.md](channels.md) for Discord, Telegram, WhatsApp, Slack, Teams, and Signal setup.
- **Configure tools**: Enable the calculator, web search, file system, and more in `gateway.yaml`.
- **Persistent memory**: Switch from `type: inmemory` to `type: redis` or `type: sqlite` in the `memory` block.
- **Full config reference**: See [docs/configuration-reference.md](configuration-reference.md) for every available option.
- **Docker deployment**: See [docs/docker.md](docker.md) to run BuckTooth with Docker Compose.
