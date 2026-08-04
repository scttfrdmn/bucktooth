# Channel Setup Guide

BuckTooth supports six messaging platforms. This guide covers the setup steps for each one.

---

## Discord

### 1. Create a bot

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications).
2. Click **New Application**, give it a name, then navigate to **Bot**.
3. Click **Add Bot** and copy the bot token.
4. Under **Privileged Gateway Intents**, enable **Message Content Intent**.

### 2. Configure BuckTooth

```yaml
channels:
  discord:
    enabled: true
    auth:
      token: ${DISCORD_BOT_TOKEN}
```

Required env var: `DISCORD_BOT_TOKEN`

### 3. Invite the bot

Generate an OAuth2 invite URL from **OAuth2 → URL Generator** with scopes `bot` and permissions:
- Send Messages
- Read Message History
- View Channels

### Troubleshooting

- **Bot not responding**: Check that **Message Content Intent** is enabled in the Developer Portal.
- **Missing permissions**: Re-invite the bot with the correct OAuth2 scopes.

---

## Telegram

### 1. Create a bot via BotFather

1. Open Telegram and message [@BotFather](https://t.me/BotFather).
2. Send `/newbot` and follow the prompts.
3. Copy the API token BotFather provides.

### 2. Configure BuckTooth

```yaml
channels:
  telegram:
    enabled: true
    auth:
      token: ${TELEGRAM_BOT_TOKEN}
```

Required env var: `TELEGRAM_BOT_TOKEN`

BuckTooth uses long-polling (no webhook server required).

### Troubleshooting

- **Bot not reachable**: Ensure no other process is polling the same bot token.
- **Group messages ignored**: By default, BuckTooth only responds when directly mentioned in groups.

---

## WhatsApp

BuckTooth uses [whatsmeow](https://github.com/tulir/whatsmeow) for WhatsApp Web protocol support.

### 1. Pair a device

On first run, BuckTooth prints a QR code in the terminal. Scan it with WhatsApp on your phone:

**WhatsApp → Settings → Linked Devices → Link a Device**

The session is saved to `session_file` so you only need to scan once.

### 2. Configure BuckTooth

```yaml
channels:
  whatsapp:
    enabled: true
    auth:
      session_file: ~/.bucktooth/whatsapp.db
    options:
      groups: false    # set true to respond in group chats
```

### Troubleshooting

- **QR code not appearing**: Delete the session file and restart to force re-pairing.
- **Session expired**: WhatsApp invalidates sessions if the phone is offline for >14 days. Re-pair.

---

## Slack

Slack's Socket Mode lets BuckTooth receive events without a public webhook URL.

### 1. Create a Slack app

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → **Create New App → From scratch**.
2. Under **Socket Mode**, enable it and create an **App-Level Token** with scope `connections:write`. Copy the `xapp-…` token.
3. Under **OAuth & Permissions**, add Bot Token Scopes:
   - `channels:history`, `channels:read`
   - `chat:write`
   - `app_mentions:read`
   - `im:history`, `im:read`, `im:write`
4. Install the app to your workspace and copy the `xoxb-…` Bot Token.
5. Under **Event Subscriptions → Subscribe to Bot Events**, add `message.channels`, `message.im`, `app_mention`.

### 2. Configure BuckTooth

```yaml
channels:
  slack:
    enabled: true
    auth:
      bot_token: ${SLACK_BOT_TOKEN}    # xoxb-...
      app_token: ${SLACK_APP_TOKEN}    # xapp-...
```

Required env vars: `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN`

### Troubleshooting

- **`invalid_auth` error**: Confirm both tokens are correct and the app is installed to the workspace.
- **No messages received**: Verify the Event Subscriptions are saved and the correct bot event types are subscribed.

---

## Microsoft Teams

BuckTooth integrates with Teams via the Bot Framework activity endpoint.

### 1. Register an Azure AD app

1. In the [Azure Portal](https://portal.azure.com), create a new **App Registration**.
2. Note the **Application (client) ID** (`app_id`).
3. Under **Certificates & secrets**, create a new client secret (`app_password`).

### 2. Register a Bot in Bot Framework

1. Go to [dev.botframework.com](https://dev.botframework.com) → **My Bots → Create a bot**.
2. Use the `app_id` from above as the Microsoft App ID.
3. Set the messaging endpoint to: `https://<your-domain>/channels/teams/activity`
4. Connect the bot to the Microsoft Teams channel.

### 3. Configure BuckTooth

```yaml
channels:
  teams:
    enabled: true
    auth:
      app_id: ${TEAMS_APP_ID}
      app_password: ${TEAMS_APP_PASSWORD}
    options:
      webhook_path: /channels/teams/activity
```

Required env vars: `TEAMS_APP_ID`, `TEAMS_APP_PASSWORD`

BuckTooth validates the Bot Framework HMAC signature on every inbound activity.

### Troubleshooting

- **401 Unauthorized**: Confirm `app_id` and `app_password` match the Azure registration.
- **No messages received**: Verify the messaging endpoint URL is publicly reachable and matches the Bot Framework registration.

---

## Signal

BuckTooth connects to Signal via [signal-cli](https://github.com/AsamK/signal-cli) running in JSON-RPC daemon mode.

### 1. Install signal-cli

Requires signal-cli **≥ 0.13.0**.

```bash
# macOS (Homebrew)
brew install signal-cli

# Linux: download from GitHub releases
# https://github.com/AsamK/signal-cli/releases
```

### 2. Register or link a phone number

```bash
# Register a new number (requires SMS or voice verification)
signal-cli -a +1234567890 register

# Or link an existing Signal account
signal-cli link -n "BuckTooth"
```

### 3. Start the signal-cli daemon

```bash
signal-cli -a +1234567890 daemon --receive-mode on-start
# Default JSON-RPC WebSocket: ws://localhost:7583
```

For production, use the Docker sidecar pattern shown in `docker-compose.yml`.

### 4. Configure BuckTooth

```yaml
channels:
  signal:
    enabled: true
    auth:
      phone_number: "+1234567890"
      signald_url: "ws://localhost:7583/v1/receive"
```

### Troubleshooting

- **Connection refused**: Ensure signal-cli daemon is running before starting BuckTooth.
- **`not_found` errors**: The phone number must be registered and the daemon started with the same number.
- **Messages not delivered**: Confirm `--receive-mode on-start` is set (not `manual`).
