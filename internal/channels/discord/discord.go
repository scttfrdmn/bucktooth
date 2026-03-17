package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// DiscordChannel implements the Channel interface for Discord
type DiscordChannel struct {
	*channels.BaseChannel
	token   string
	session *discordgo.Session
	logger  zerolog.Logger
}

// NewDiscordChannel creates a new Discord channel
func NewDiscordChannel(token string, logger zerolog.Logger) (*DiscordChannel, error) {
	if token == "" {
		return nil, fmt.Errorf("discord token is required")
	}

	base := channels.NewBaseChannel("discord", logger, 100)

	return &DiscordChannel{
		BaseChannel: base,
		token:       token,
		logger:      logger.With().Str("channel", "discord").Logger(),
	}, nil
}

// Connect establishes connection to Discord
func (d *DiscordChannel) Connect(ctx context.Context) error {
	if d.IsConnected() {
		return channels.ErrAlreadyConnected
	}

	d.logger.Info().Msg("connecting to Discord")

	// Create Discord session
	session, err := discordgo.New("Bot " + d.token)
	if err != nil {
		d.UpdateHealth(false, "failed", err)
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Register message handler
	session.AddHandler(d.handleMessage)

	// Set intents
	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent

	// Open connection
	if err := session.Open(); err != nil {
		d.UpdateHealth(false, "failed", err)
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	d.session = session
	d.SetConnected(true)

	d.logger.Info().Msg("connected to Discord")
	return nil
}

// Disconnect closes the Discord connection
func (d *DiscordChannel) Disconnect() error {
	if !d.IsConnected() {
		return nil
	}

	d.logger.Info().Msg("disconnecting from Discord")

	if d.session != nil {
		if err := d.session.Close(); err != nil {
			d.logger.Error().Err(err).Msg("error closing Discord session")
			return err
		}
		d.session = nil
	}

	d.SetConnected(false)
	d.Close()

	d.logger.Info().Msg("disconnected from Discord")
	return nil
}

// SendMessage sends a message to Discord
func (d *DiscordChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	if !d.IsConnected() {
		return channels.ErrNotConnected
	}

	// Extract channel ID from metadata or use the message ChannelID
	channelID := msg.ChannelID
	if id, ok := msg.Metadata["discord_channel_id"].(string); ok {
		channelID = id
	}

	if channelID == "" {
		return fmt.Errorf("discord channel ID is required")
	}

	_, err := d.session.ChannelMessageSend(channelID, msg.Content)
	if err != nil {
		d.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to send Discord message")
		return fmt.Errorf("failed to send Discord message: %w", err)
	}

	d.logger.Debug().Str("channel_id", channelID).Msg("sent Discord message")
	return nil
}

// ReceiveMessages returns a channel for receiving messages
func (d *DiscordChannel) ReceiveMessages(ctx context.Context) (<-chan *channels.Message, error) {
	if !d.IsConnected() {
		return nil, channels.ErrNotConnected
	}

	return d.MessageQueue(), nil
}

// handleMessage processes incoming Discord messages
func (d *DiscordChannel) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	msg := &channels.Message{
		ID:        m.ID,
		ChannelID: d.Name(), // channel adapter name ("discord") for routing
		UserID:    m.Author.ID,
		Username:  m.Author.Username,
		Content:   m.Content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"discord_channel_id": m.ChannelID,
			"discord_guild_id":   m.GuildID,
			"discord_author_discriminator": m.Author.Discriminator,
		},
		Attachments: make([]channels.Attachment, 0, len(m.Attachments)),
	}

	// Convert Discord attachments
	for _, att := range m.Attachments {
		msg.Attachments = append(msg.Attachments, channels.Attachment{
			ID:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        int64(att.Size),
			URL:         att.URL,
		})
	}

	// Queue the message
	if err := d.QueueMessage(msg); err != nil {
		d.logger.Error().Err(err).Str("msg_id", msg.ID).Msg("failed to queue message")
	}
}
