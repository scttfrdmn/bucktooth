# BuckTooth - OpenClaw Clone in Go with Agenkit

**"No Claws All Teeth!"** 🦫

BuckTooth is a high-performance personal AI assistant that connects to messaging platforms (Discord, WhatsApp, Telegram, Slack) where users interact with it naturally through chat. Built with Go and Agenkit-Go for 18x better performance than the original OpenClaw.

![BuckTooth Logo](logo.png)

## Features

- **Multi-Channel Support**: Discord (Phase 1), WhatsApp, Telegram, Slack (coming soon)
- **Conversational AI**: Powered by Claude via Agenkit-Go
- **Memory**: Maintains conversation context across channels
- **Production-Ready**: Built-in observability, metrics, and health checks
- **Single Binary**: Easy deployment with minimal dependencies

## Quick Start

### Prerequisites

- Go 1.23 or higher
- Discord bot token (get from [Discord Developer Portal](https://discord.com/developers/applications))
- Anthropic API key (get from [Anthropic Console](https://console.anthropic.com))

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd BuckTooth

# Build
go build -o BuckTooth ./cmd/gateway

# Or use go run
go run ./cmd/gateway
```

### Configuration

Create a `.env` file or export environment variables:

```bash
export DISCORD_BOT_TOKEN=your_discord_bot_token
export ANTHROPIC_API_KEY=your_anthropic_api_key
```

Or create a configuration file:

```yaml
# config.yaml
gateway:
  websocket_port: 18789
  http_port: 8080
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
  temperature: 0.7
```

### Running

```bash
# Using environment variables
./BuckTooth

# Using config file
./BuckTooth --config config.yaml

# With custom log level
./BuckTooth --log-level debug
```

## Usage

1. Invite your Discord bot to a server
2. Start BuckTooth
3. Send a message to the bot in Discord
4. The bot will respond using Claude AI

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  BuckTooth Gateway (Single Go Binary)                     │
│  ┌─────────────────────────────────────────────────┐   │
│  │ WebSocket Server (:18789) + HTTP API (:8080)    │   │
│  │ - Channel Registry & Lifecycle                  │   │
│  │ - Message Router & Event Bus                    │   │
│  │ - Metrics & Health Checks                       │   │
│  └─────────────────────────────────────────────────┘   │
│           │                                             │
│  ┌────────┴────────┬──────────┬──────────┐            │
│  │ Discord Channel │ WhatsApp │ Telegram │ Slack...   │
│  └────────┬────────┴──────────┴──────────┘            │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Agent Router (Agenkit ConversationalAgent)      │  │
│  └────────┬────────────────────────────────────────┘  │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Memory Store (In-Memory)                         │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## API Endpoints

- `GET /health` - Health check (returns channel status)
- `GET /status` - Detailed status information
- `GET /metrics` - Prometheus metrics
- `WS /ws` - WebSocket connection (port 18789)

## Development

### Project Structure

```
BuckTooth/
├── cmd/
│   └── gateway/          # Main gateway binary
├── internal/
│   ├── gateway/          # Gateway implementation
│   ├── channels/         # Channel adapters (Discord, etc.)
│   ├── agents/           # Agent configurations
│   ├── tools/            # Tool implementations
│   ├── memory/           # Memory management
│   └── config/           # Configuration
├── configs/              # Configuration files
└── docs/                 # Documentation
```

### Building

```bash
# Build for current platform
go build -o BuckTooth ./cmd/gateway

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o BuckTooth-linux ./cmd/gateway

# Build with optimizations
go build -ldflags="-s -w" -o BuckTooth ./cmd/gateway
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

## Configuration Reference

See [configs/gateway.yaml](configs/gateway.yaml) for a complete configuration example.

### Environment Variables

- `DISCORD_BOT_TOKEN` - Discord bot token
- `ANTHROPIC_API_KEY` - Anthropic API key
- `LOBSTER_GATEWAY_PORT` - HTTP port (default: 8080)
- `LOBSTER_WEBSOCKET_PORT` - WebSocket port (default: 18789)
- `LOBSTER_LOG_LEVEL` - Log level (debug, info, warn, error)

## Roadmap

### Phase 1: Foundation ✅
- [x] Gateway core
- [x] Discord channel
- [x] Conversational agent
- [x] In-memory storage
- [x] Basic observability

### Phase 2: Tools & Multi-Channel (In Progress)
- [ ] Tool registry
- [ ] Calculator, Message, FileSystem tools
- [ ] WhatsApp channel
- [ ] Cross-channel messaging

### Phase 3: Expansion
- [ ] Telegram channel
- [ ] Slack channel
- [ ] Planning agent
- [ ] Calendar & Web Search tools
- [ ] OpenTelemetry tracing

### Phase 4: Polish
- [ ] Web dashboard
- [ ] Enhanced CLI
- [ ] Docker image
- [ ] Kubernetes manifests
- [ ] Migration tool from OpenClaw

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

[License TBD]

## Acknowledgments

- Built with [Agenkit-Go](https://github.com/scttfrdmn/agenkit-go)
- Inspired by [OpenClaw](https://github.com/openclaw/openclaw)
- Powered by [Anthropic Claude](https://www.anthropic.com)
