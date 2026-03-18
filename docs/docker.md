# Docker Quick Start

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2

## Running with Docker Compose

```bash
# Copy and edit environment variables
cp .env.example .env
# Edit .env with your tokens

# Start BuckTooth + Redis
docker-compose up -d

# View logs
docker-compose logs -f bucktooth

# Stop
docker-compose down
```

The dashboard is available at http://localhost:8080 once the container is healthy.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DISCORD_BOT_TOKEN` | If Discord enabled | Discord bot token |
| `ANTHROPIC_API_KEY` | Yes | Anthropic API key |
| `TELEGRAM_BOT_TOKEN` | If Telegram enabled | Telegram bot token |
| `SLACK_BOT_TOKEN` | If Slack enabled | Slack bot token |
| `SLACK_APP_TOKEN` | If Slack enabled | Slack app-level token |
| `BRAVE_SEARCH_API_KEY` | If web search enabled | Brave Search API key |

## Redis Memory Configuration

When running with `docker-compose`, BuckTooth automatically connects to the bundled Redis
container. To use Redis for conversation memory, set in `configs/gateway.yaml`:

```yaml
memory:
  type: redis
  options:
    addr: redis:6379
    ttl: 24h
    max_history: 50
```

Redis data is persisted in the `redis_data` Docker volume.

## Building a Custom Image

```bash
docker build --build-arg VERSION=0.4.0 -t bucktooth:0.4.0 .
```

## Health Check

The container uses `bucktooth status` as its health check. You can query it directly:

```bash
docker exec <container> /usr/local/bin/bucktooth status
```
