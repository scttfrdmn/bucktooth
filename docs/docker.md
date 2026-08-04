# Docker Deployment

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2

---

## Quick Start with Docker Compose

```bash
# 1. Copy and edit environment variables
cp .env.example .env
# Fill in ANTHROPIC_API_KEY and at least one channel token

# 2. Start BuckTooth + Redis
docker compose up -d

# 3. View logs
docker compose logs -f bucktooth

# 4. Verify
curl -s localhost:8080/health | jq .

# 5. Stop
docker compose down
```

The web dashboard is available at **http://localhost:8080** once the container reports healthy.

---

## Environment Variables

Copy `.env.example` to `.env` and fill in your values. The most common variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | Yes | Anthropic API key |
| `DISCORD_BOT_TOKEN` | If Discord enabled | Discord bot token |
| `TELEGRAM_BOT_TOKEN` | If Telegram enabled | Telegram bot token |
| `SLACK_BOT_TOKEN` | If Slack enabled | Slack bot OAuth token (`xoxb-…`) |
| `SLACK_APP_TOKEN` | If Slack enabled | Slack app-level token (`xapp-…`) |
| `TEAMS_APP_ID` | If Teams enabled | Azure AD Application ID |
| `TEAMS_APP_PASSWORD` | If Teams enabled | Azure AD client secret |
| `BRAVE_SEARCH_API_KEY` | If web search enabled | Brave Search API key |
| `DASHBOARD_AUTH_PASSWORD` | Optional | Enables Basic auth on the dashboard |
| `BUCKTOOTH_API_TOKEN` | Optional | Enables Bearer token auth on all API routes |

See [docs/configuration-reference.md](configuration-reference.md) for the full list.

---

## Redis Memory

When running with `docker compose`, BuckTooth connects to the bundled Redis container. To use Redis for persistent conversation memory, set in `configs/gateway.yaml`:

```yaml
memory:
  type: redis
  options:
    addr: redis:6379
    ttl: 24h
    max_history: 50
```

Redis data is persisted in the `redis_data` Docker volume and survives container restarts.

---

## Adding More Channels

Enable channels by editing `configs/gateway.yaml` and adding the required tokens to `.env`:

```yaml
# gateway.yaml — enable Telegram alongside Discord
channels:
  discord:
    enabled: true
    auth:
      token: ${DISCORD_BOT_TOKEN}
  telegram:
    enabled: true
    auth:
      token: ${TELEGRAM_BOT_TOKEN}
```

---

## LLM Cost Tracking

To track USD costs per model and expose `GET /v1/usage`:

```yaml
observability:
  cost_tracking:
    enabled: true
```

---

## Signal Channel (signal-cli Sidecar)

The Signal channel requires [signal-cli](https://github.com/AsamK/signal-cli) running as a separate service. A commented-out sidecar block is included in `docker-compose.yml`:

```yaml
# signal-cli:
#   image: bbernhard/signal-cli-rest-api:latest
#   environment:
#     - MODE=json-rpc
#   volumes:
#     - signal_data:/home/.local/share/signal-cli
```

Uncomment this block and follow [docs/channels.md#signal](channels.md#signal) to register your phone number before starting.

---

## Building a Custom Image

```bash
docker build --build-arg VERSION=0.13.0 -t bucktooth:0.13.0 .
```

The Dockerfile uses a multi-stage build with a distroless final image (~25 MB).

---

## Pre-built Images (GHCR)

Multi-arch images (linux/amd64 and linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/scttfrdmn/bucktooth:v0.13.0
docker pull ghcr.io/scttfrdmn/bucktooth:latest
```

---

## Health Check

```bash
# Via CLI
docker exec <container> /usr/local/bin/bucktooth status

# Via HTTP
curl -s localhost:8080/health | jq .
```

---

## Kubernetes (Helm)

For production Kubernetes deployments, use the Helm chart:

```bash
helm install bucktooth ./charts/bucktooth \
  --set secrets.anthropicApiKey=sk-ant-... \
  --set secrets.discordBotToken=...
```

See `charts/bucktooth/values.yaml` for all available configuration options.
