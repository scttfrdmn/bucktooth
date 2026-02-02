# BuckTooth Architecture

## Overview

BuckTooth is a multi-channel AI assistant gateway built with Go and Agenkit-Go. It connects to various messaging platforms (Discord, WhatsApp, Telegram, Slack) and provides AI-powered responses using Anthropic's Claude.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│  BuckTooth Gateway (Single Go Binary)                     │
│  ┌─────────────────────────────────────────────────┐   │
│  │ HTTP Server (:8080)                             │   │
│  │ - /health  - Health check endpoint              │   │
│  │ - /status  - Status endpoint                    │   │
│  │ - /metrics - Prometheus metrics                 │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ WebSocket Server (:18789)                       │   │
│  │ - Real-time client connections                  │   │
│  │ - Event streaming                               │   │
│  └─────────────────────────────────────────────────┘   │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Event Bus                                       │  │
│  │ - Message routing                               │  │
│  │ - Event publishing/subscription                 │  │
│  └────────┬────────────────────────────────────────┘  │
│           │                                             │
│  ┌────────┴────────┬──────────┬──────────┐            │
│  │ Discord Channel │ WhatsApp │ Telegram │ Slack...   │
│  │ - Receive msgs  │ (Phase 2)│ (Phase 3)│            │
│  │ - Send msgs     │          │          │            │
│  └────────┬────────┴──────────┴──────────┘            │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Agent Router                                    │  │
│  │ - Route to appropriate agent                    │  │
│  │ - Manage conversation context                   │  │
│  └────────┬────────────────────────────────────────┘  │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Agenkit ConversationalAgent                     │  │
│  │ - LLM interaction (Anthropic Claude)            │  │
│  │ - Conversation history management               │  │
│  └────────┬────────────────────────────────────────┘  │
│           │                                             │
│  ┌────────▼────────────────────────────────────────┐  │
│  │ Memory Store (In-Memory)                        │  │
│  │ - User conversation history                     │  │
│  │ - Per-user message storage                      │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Components

### Gateway

The main application component that orchestrates all other components.

**Responsibilities:**
- Initialize and manage component lifecycle
- Handle graceful shutdown
- Register and manage channels
- Coordinate message flow

**Files:**
- `internal/gateway/gateway.go`

### Event Bus

Publish-subscribe event system for internal communication.

**Event Types:**
- `message.received` - New message from a channel
- `message.sent` - Message sent to a channel
- `channel.connected` - Channel successfully connected
- `channel.disconnected` - Channel disconnected
- `agent.started` - Agent started processing
- `agent.completed` - Agent completed processing
- `agent.error` - Agent encountered an error

**Files:**
- `internal/gateway/eventbus.go`

### Channels

Adapters for different messaging platforms.

**Channel Interface:**
```go
type Channel interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect() error
    SendMessage(ctx context.Context, msg *Message) error
    ReceiveMessages(ctx context.Context) (<-chan *Message, error)
    Health() HealthStatus
}
```

**Implemented Channels:**
- Discord (Phase 1) - `internal/channels/discord/`
- WhatsApp (Phase 2) - `internal/channels/whatsapp/`
- Telegram (Phase 3) - `internal/channels/telegram/`
- Slack (Phase 3) - `internal/channels/slack/`

**Files:**
- `internal/channels/channel.go` - Channel interface
- `internal/channels/base.go` - Base channel implementation
- `internal/channels/discord/discord.go` - Discord implementation

### Agent Router

Routes messages to appropriate agents and manages conversation context.

**Responsibilities:**
- Select appropriate agent for each message
- Manage conversation history
- Interface with memory store
- Handle agent errors

**Current Agents:**
- ConversationalAgent (Agenkit pattern)

**Future Agents (Phase 2+):**
- ReActAgent - For tool-augmented reasoning
- PlanningAgent - For multi-step workflows
- RouterAgent - For intelligent agent selection

**Files:**
- `internal/gateway/router.go`

### Memory Store

Stores conversation history for each user.

**Implementations:**
- In-Memory (Phase 1) - Simple map-based storage
- Redis (Future) - Persistent distributed storage

**Files:**
- `internal/memory/memory.go` - Store interface
- `internal/memory/inmemory.go` - In-memory implementation

### HTTP Server

Provides REST API endpoints for monitoring and control.

**Endpoints:**
- `GET /health` - Health check with channel status
- `GET /status` - Detailed system status
- `GET /metrics` - Prometheus metrics

**Files:**
- `internal/gateway/http.go`

### WebSocket Server

Provides real-time WebSocket connections for clients.

**Features:**
- Real-time message streaming
- Event notifications
- Client management

**Files:**
- `internal/gateway/websocket.go`

## Message Flow

1. User sends message via Discord
2. Discord channel receives message via webhook
3. Channel queues message internally
4. Gateway reads message from channel queue
5. Gateway publishes `message.received` event
6. Agent router receives event and processes message
7. Router retrieves conversation history from memory store
8. Router sends message + history to ConversationalAgent
9. ConversationalAgent calls Anthropic API
10. Agent returns response
11. Router stores message and response in memory
12. Gateway publishes `agent.completed` event
13. Gateway sends response back through Discord channel
14. Discord channel delivers message to user

## Configuration

Configuration is loaded from:
1. Default values (embedded)
2. YAML config file (`configs/gateway.yaml`)
3. Environment variables (`LOBSTER_*`)
4. CLI flags (`--flag`)

**Configuration Structure:**
```yaml
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

memory:
  type: inmemory
```

## Observability

### Logging

Structured JSON logging using zerolog:
- Log levels: debug, info, warn, error
- Contextual logging with trace IDs
- Component-specific loggers

### Metrics

Prometheus metrics exported at `/metrics`:
- Message counters (received, sent)
- Agent processing duration
- Channel health status
- Error rates

### Health Checks

Health endpoint at `/health`:
- Overall system health
- Per-channel health status
- Connection status

## Error Handling

- Channel errors don't affect other channels
- Graceful degradation
- Automatic reconnection for channels
- Error events published via event bus

## Scalability

Current architecture supports:
- Multiple channels simultaneously
- Concurrent message processing
- Horizontal scaling (future with Redis memory)

**Bottlenecks:**
- In-memory storage (Phase 1 only)
- Single-instance deployment (Phase 1)

**Future Improvements:**
- Redis for distributed memory
- Message queue for async processing
- Load balancing across instances

## Security

- Input validation on all messages
- API key management via environment variables
- No credentials in config files
- Per-channel authentication

## Testing Strategy

- Unit tests for all components
- Integration tests for message flow
- E2E tests for complete workflows
- Channel adapter tests with mocks
