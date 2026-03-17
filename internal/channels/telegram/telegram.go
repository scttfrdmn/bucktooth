package telegram

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// TelegramChannel implements the Channel interface for Telegram via long-polling.
type TelegramChannel struct {
	*channels.BaseChannel
	token  string
	bot    *tgbotapi.BotAPI
	cancel context.CancelFunc
	logger zerolog.Logger
}

// NewTelegramChannel creates a new Telegram channel.
func NewTelegramChannel(cfg config.ChannelConfig, logger zerolog.Logger) (*TelegramChannel, error) {
	token, _ := cfg.Auth["token"].(string)
	if token == "" {
		return nil, fmt.Errorf("telegram: token is required")
	}

	base := channels.NewBaseChannel("telegram", logger, 100)

	return &TelegramChannel{
		BaseChannel: base,
		token:       token,
		logger:      logger.With().Str("channel", "telegram").Logger(),
	}, nil
}

// Connect starts the Telegram bot and launches the long-polling goroutine.
func (t *TelegramChannel) Connect(ctx context.Context) error {
	if t.IsConnected() {
		return channels.ErrAlreadyConnected
	}

	t.logger.Info().Msg("connecting to Telegram")

	bot, err := tgbotapi.NewBotAPI(t.token)
	if err != nil {
		t.UpdateHealth(false, "failed", err)
		return fmt.Errorf("telegram: failed to create bot: %w", err)
	}

	t.bot = bot

	pollCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	t.SetConnected(true)
	t.logger.Info().Str("username", bot.Self.UserName).Msg("connected to Telegram")

	go t.poll(pollCtx)

	return nil
}

// poll reads updates from the Telegram long-poll API and queues messages.
func (t *TelegramChannel) poll(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			t.bot.StopReceivingUpdates()
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			t.handleUpdate(update)
		}
	}
}

// handleUpdate converts a Telegram Update into an internal Message and queues it.
func (t *TelegramChannel) handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	m := update.Message

	// Skip messages from bots (including ourselves).
	if m.From != nil && m.From.IsBot {
		return
	}

	username := ""
	if m.From != nil {
		username = m.From.UserName
		if username == "" {
			username = m.From.FirstName
		}
	}

	userID := ""
	if m.From != nil {
		userID = strconv.FormatInt(m.From.ID, 10)
	}

	chatID := strconv.FormatInt(m.Chat.ID, 10)

	msg := &channels.Message{
		ID:        strconv.Itoa(m.MessageID),
		ChannelID: "telegram",
		UserID:    userID,
		Username:  username,
		Content:   m.Text,
		Timestamp: time.Unix(int64(m.Date), 0),
		Metadata: map[string]interface{}{
			"telegram_chat_id": chatID,
		},
	}

	if err := t.QueueMessage(msg); err != nil {
		t.logger.Error().Err(err).Str("msg_id", msg.ID).Msg("failed to queue Telegram message")
	}
}

// Disconnect stops long-polling and closes the channel.
func (t *TelegramChannel) Disconnect() error {
	if !t.IsConnected() {
		return nil
	}

	t.logger.Info().Msg("disconnecting from Telegram")

	if t.cancel != nil {
		t.cancel()
	}

	t.SetConnected(false)
	t.Close()

	t.logger.Info().Msg("disconnected from Telegram")
	return nil
}

// SendMessage sends a text message to the Telegram chat identified in metadata.
func (t *TelegramChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	if !t.IsConnected() {
		return channels.ErrNotConnected
	}

	chatIDStr, _ := msg.Metadata["telegram_chat_id"].(string)
	if chatIDStr == "" {
		return fmt.Errorf("telegram: missing telegram_chat_id in message metadata")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: invalid chat_id %q: %w", chatIDStr, err)
	}

	outMsg := tgbotapi.NewMessage(chatID, msg.Content)
	if _, err := t.bot.Send(outMsg); err != nil {
		return fmt.Errorf("telegram: failed to send message: %w", err)
	}

	return nil
}

// ReceiveMessages returns the inbound message channel.
func (t *TelegramChannel) ReceiveMessages(ctx context.Context) (<-chan *channels.Message, error) {
	if !t.IsConnected() {
		return nil, channels.ErrNotConnected
	}
	return t.MessageQueue(), nil
}
