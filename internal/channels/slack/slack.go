package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// SlackChannel implements the Channel interface for Slack via Socket Mode.
type SlackChannel struct {
	*channels.BaseChannel
	botToken     string
	appToken     string
	client       *slack.Client
	socketClient *socketmode.Client
	cancel       context.CancelFunc
	logger       zerolog.Logger
}

// NewSlackChannel creates a new Slack channel from config.
func NewSlackChannel(cfg config.ChannelConfig, logger zerolog.Logger) (*SlackChannel, error) {
	botToken, _ := cfg.Auth["bot_token"].(string)
	if botToken == "" {
		return nil, fmt.Errorf("slack: bot_token is required")
	}

	appToken, _ := cfg.Auth["app_token"].(string)
	if appToken == "" {
		return nil, fmt.Errorf("slack: app_token is required (Socket Mode xapp-... token)")
	}

	base := channels.NewBaseChannel("slack", logger, 100)

	return &SlackChannel{
		BaseChannel: base,
		botToken:    botToken,
		appToken:    appToken,
		logger:      logger.With().Str("channel", "slack").Logger(),
	}, nil
}

// Connect establishes the Slack Socket Mode connection.
func (s *SlackChannel) Connect(ctx context.Context) error {
	if s.IsConnected() {
		return channels.ErrAlreadyConnected
	}

	s.logger.Info().Msg("connecting to Slack")

	client := slack.New(
		s.botToken,
		slack.OptionAppLevelToken(s.appToken),
	)

	socketClient := socketmode.New(client)

	s.client = client
	s.socketClient = socketClient

	connCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.SetConnected(true)
	s.logger.Info().Msg("connected to Slack")

	// Run Socket Mode receiver and event handler concurrently.
	go func() {
		if err := socketClient.RunContext(connCtx); err != nil && connCtx.Err() == nil {
			s.logger.Error().Err(err).Msg("Slack socket client error")
		}
	}()

	go s.handleEvents(connCtx)

	return nil
}

// handleEvents processes events from the Slack Socket Mode event channel.
func (s *SlackChannel) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-s.socketClient.Events:
			if !ok {
				return
			}

			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}

				// Acknowledge immediately.
				s.socketClient.Ack(*evt.Request)

				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					s.handleCallbackEvent(eventsAPIEvent)
				}

			default:
				// Acknowledge all other event types we don't handle.
				if evt.Request != nil {
					s.socketClient.Ack(*evt.Request)
				}
			}
		}
	}
}

// handleCallbackEvent processes a Slack callback event and queues inbound messages.
func (s *SlackChannel) handleCallbackEvent(event slackevents.EventsAPIEvent) {
	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		// Ignore bot messages.
		if ev.BotID != "" {
			return
		}

		// Ignore message edits, deletions, etc.
		if ev.SubType != "" {
			return
		}

		msg := &channels.Message{
			ID:        ev.TimeStamp,
			ChannelID: "slack",
			UserID:    ev.User,
			Username:  ev.User,
			Content:   ev.Text,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"slack_channel_id": ev.Channel,
			},
		}

		if err := s.QueueMessage(msg); err != nil {
			s.logger.Error().Err(err).Str("ts", ev.TimeStamp).Msg("failed to queue Slack message")
		}
	}
}

// Disconnect stops the Socket Mode connection.
func (s *SlackChannel) Disconnect() error {
	if !s.IsConnected() {
		return nil
	}

	s.logger.Info().Msg("disconnecting from Slack")

	if s.cancel != nil {
		s.cancel()
	}

	s.SetConnected(false)
	s.Close()

	s.logger.Info().Msg("disconnected from Slack")
	return nil
}

// SendMessage posts a message to the Slack channel identified in metadata.
func (s *SlackChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	if !s.IsConnected() {
		return channels.ErrNotConnected
	}

	channelID, _ := msg.Metadata["slack_channel_id"].(string)
	if channelID == "" {
		return fmt.Errorf("slack: missing slack_channel_id in message metadata")
	}

	_, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(msg.Content, false))
	if err != nil {
		return fmt.Errorf("slack: failed to post message: %w", err)
	}

	return nil
}

// ReceiveMessages returns the inbound message channel.
func (s *SlackChannel) ReceiveMessages(ctx context.Context) (<-chan *channels.Message, error) {
	if !s.IsConnected() {
		return nil, channels.ErrNotConnected
	}
	return s.MessageQueue(), nil
}
