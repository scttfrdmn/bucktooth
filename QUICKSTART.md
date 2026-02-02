# BuckTooth Quick Start Guide

Get BuckTooth up and running in 5 minutes!

## Prerequisites

- Go 1.23 or higher
- Discord bot token ([Get one here](https://discord.com/developers/applications))
- Anthropic API key ([Get one here](https://console.anthropic.com))

## Step 1: Set Up Discord Bot

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name
3. Go to "Bot" section
4. Click "Add Bot"
5. Under "Privileged Gateway Intents", enable:
   - Message Content Intent
   - Server Members Intent (optional)
6. Copy the bot token (you'll need this)
7. Go to "OAuth2" → "URL Generator"
8. Select scopes: `bot`
9. Select permissions: `Send Messages`, `Read Messages/View Channels`, `Read Message History`
10. Copy the generated URL and open it to invite bot to your server

## Step 2: Configure BuckTooth

Create a `.env` file in the project root:

```bash
cp .env.example .env
```

Edit `.env` and add your credentials:

```bash
DISCORD_BOT_TOKEN=your_discord_bot_token_here
ANTHROPIC_API_KEY=your_anthropic_api_key_here
```

## Step 3: Install Dependencies

```bash
make deps
```

Or manually:

```bash
go mod download
go mod tidy
```

## Step 4: Build

```bash
make build
```

This creates `bin/BuckTooth` binary.

## Step 5: Run

```bash
./bin/BuckTooth
```

Or run directly without building:

```bash
make run
```

For debug logging:

```bash
make run-debug
```

## Step 6: Test

1. Open Discord
2. Go to a channel where your bot has access
3. Send a message: `Hello!`
4. The bot should respond with an AI-generated message

## Verification

Check that everything is working:

1. **Health Check**:
   ```bash
   curl http://localhost:8080/health
   ```

   Expected output:
   ```json
   {
     "status": "healthy",
     "channels": {
       "discord": {
         "healthy": true,
         "status": "connected"
       }
     }
   }
   ```

2. **Status Check**:
   ```bash
   curl http://localhost:8080/status
   ```

3. **Metrics**:
   ```bash
   curl http://localhost:8080/metrics
   ```

## Logs

BuckTooth logs to stdout with structured JSON:

```bash
# View logs with pretty formatting
./bin/BuckTooth | jq
```

Example log output:
```json
{
  "level": "info",
  "component": "gateway",
  "message": "starting gateway",
  "time": "2026-02-01T13:00:00Z"
}
```

## Configuration

### Environment Variables

Override defaults with environment variables:

```bash
export LOBSTER_GATEWAY_PORT=9090
export LOBSTER_WEBSOCKET_PORT=18790
export LOBSTER_LOG_LEVEL=debug
./bin/BuckTooth
```

### Config File

Or use a custom config file:

```bash
./bin/BuckTooth --config my-config.yaml
```

Example config:

```yaml
gateway:
  http_port: 9090
  websocket_port: 18790
  log_level: debug

channels:
  discord:
    enabled: true
    auth:
      token: ${DISCORD_BOT_TOKEN}

agents:
  llm_model: claude-sonnet-4-5-20250220
  max_history: 20
  temperature: 0.7
```

## Troubleshooting

### Bot doesn't respond

1. **Check logs**: Look for errors in the output
2. **Verify bot permissions**: Ensure bot has "Send Messages" and "View Channels"
3. **Check intents**: Make sure "Message Content Intent" is enabled in Discord Developer Portal
4. **Test health endpoint**: `curl http://localhost:8080/health`

### Connection errors

1. **Verify Discord token**: Check that `DISCORD_BOT_TOKEN` is correct
2. **Verify Anthropic API key**: Check that `ANTHROPIC_API_KEY` is valid
3. **Check network**: Ensure you can reach Discord and Anthropic APIs

### Build errors

1. **Check Go version**: `go version` (need 1.23+)
2. **Clean and rebuild**:
   ```bash
   make clean
   make deps
   make build
   ```

### Import errors

If you see import path errors:
```bash
go mod tidy
go mod download
```

## Development

### Running with auto-reload

Install air:
```bash
go install github.com/cosmtrek/air@latest
```

Run:
```bash
make dev
```

### Running tests

```bash
make test
```

With coverage:
```bash
make test-coverage
open coverage.html
```

### Code formatting

```bash
make fmt
```

### Linting

```bash
# Install golangci-lint first
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
make lint
```

## Docker (Coming Soon)

Build Docker image:
```bash
make docker-build
```

Run with Docker:
```bash
make docker-run
```

## Next Steps

Once you have the basic setup working:

1. **Explore features**: Try different prompts with the bot
2. **Check metrics**: Monitor `/metrics` endpoint
3. **Read architecture**: See `docs/architecture.md`
4. **Add more channels**: WhatsApp, Telegram (Phase 2)
5. **Enable tools**: Calculator, file operations (Phase 2)

## Getting Help

- **Documentation**: See `docs/` directory
- **Issues**: Check existing issues or create a new one
- **Architecture**: Read `docs/architecture.md`
- **Status**: Check `STATUS.md` for implementation progress

## Example Interactions

Try these prompts with your bot:

```
Hello!
→ Bot responds with greeting

What's 2 + 2?
→ Bot answers (currently without calculator tool)

Tell me a joke
→ Bot tells a joke

Explain quantum computing
→ Bot provides explanation
```

## Success Checklist

- [ ] Discord bot created and invited to server
- [ ] `.env` file configured with tokens
- [ ] Dependencies installed
- [ ] Binary built successfully
- [ ] Gateway running without errors
- [ ] Bot responds to messages in Discord
- [ ] `/health` endpoint returns healthy status
- [ ] Logs show successful message processing

Congratulations! 🎉 BuckTooth is now running!
