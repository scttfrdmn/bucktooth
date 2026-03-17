package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// WhatsAppChannel implements the Channel interface for WhatsApp via whatsmeow.
type WhatsAppChannel struct {
	*channels.BaseChannel
	client      *whatsmeow.Client
	container   *sqlstore.Container
	sessionFile string
	allowGroups bool
	logger      zerolog.Logger
}

// NewWhatsAppChannel creates a new WhatsApp channel adapter.
func NewWhatsAppChannel(cfg config.ChannelConfig, logger zerolog.Logger) (*WhatsAppChannel, error) {
	sessionFile, _ := cfg.Auth["session_file"].(string)
	if sessionFile == "" {
		sessionFile = "~/.bucktooth/whatsapp.db"
	}

	// Expand ~ in path
	if strings.HasPrefix(sessionFile, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		sessionFile = filepath.Join(home, sessionFile[2:])
	}

	allowGroups := false
	if v, ok := cfg.Options["groups"].(bool); ok {
		allowGroups = v
	}

	base := channels.NewBaseChannel("whatsapp", logger, 100)

	return &WhatsAppChannel{
		BaseChannel: base,
		sessionFile: sessionFile,
		allowGroups: allowGroups,
		logger:      logger.With().Str("channel", "whatsapp").Logger(),
	}, nil
}

// Connect establishes connection to WhatsApp, prompting for QR scan if needed.
func (w *WhatsAppChannel) Connect(ctx context.Context) error {
	if w.IsConnected() {
		return channels.ErrAlreadyConnected
	}

	w.logger.Info().Msg("connecting to WhatsApp")

	// Ensure session directory exists
	if err := os.MkdirAll(filepath.Dir(w.sessionFile), 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Open SQLite store
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", w.sessionFile))
	if err != nil {
		return fmt.Errorf("failed to open session database: %w", err)
	}

	container := sqlstore.NewWithDB(db, "sqlite3", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		return fmt.Errorf("failed to upgrade session database: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	w.container = container

	// Create WhatsApp client
	client := whatsmeow.NewClient(device, waLog.Noop)
	client.AddEventHandler(w.handleEvent)
	w.client = client

	if device.ID == nil {
		// No existing session — need QR code pairing
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("failed to get QR channel: %w", err)
		}

		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to WhatsApp: %w", err)
		}

		w.logger.Info().Msg("waiting for QR code scan — open WhatsApp on your phone, go to Settings → Linked Devices → Link a Device")

		for item := range qrChan {
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				fmt.Fprintf(os.Stderr, "\n=== WhatsApp QR Code ===\n%s\n=======================\n\n", item.Code)
				w.logger.Info().Str("qr_code", item.Code).Msg("scan this QR code in WhatsApp")
			case whatsmeow.QRChannelSuccess.Event:
				w.logger.Info().Msg("WhatsApp QR code scanned successfully")
			case whatsmeow.QRChannelTimeout.Event:
				return fmt.Errorf("WhatsApp QR code timed out — restart to try again")
			case whatsmeow.QRChannelErrUnexpectedEvent.Event:
				return fmt.Errorf("unexpected WhatsApp event during QR pairing")
			}
		}
	} else {
		// Existing session — connect directly
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to WhatsApp: %w", err)
		}
	}

	// Brief wait for connection to stabilize
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	w.SetConnected(true)
	w.logger.Info().Msg("connected to WhatsApp")
	return nil
}

// Disconnect closes the WhatsApp connection.
func (w *WhatsAppChannel) Disconnect() error {
	if !w.IsConnected() {
		return nil
	}

	w.logger.Info().Msg("disconnecting from WhatsApp")

	if w.client != nil {
		w.client.Disconnect()
		w.client = nil
	}

	w.SetConnected(false)
	w.Close()

	w.logger.Info().Msg("disconnected from WhatsApp")
	return nil
}

// SendMessage sends a text message to a WhatsApp JID.
func (w *WhatsAppChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	if !w.IsConnected() {
		return channels.ErrNotConnected
	}

	jidStr := msg.ChannelID
	if id, ok := msg.Metadata["whatsapp_jid"].(string); ok {
		jidStr = id
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid WhatsApp JID %q: %w", jidStr, err)
	}

	_, err = w.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(msg.Content),
	})
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	w.logger.Debug().Str("jid", jidStr).Msg("sent WhatsApp message")
	return nil
}

// ReceiveMessages returns the incoming message queue.
func (w *WhatsAppChannel) ReceiveMessages(ctx context.Context) (<-chan *channels.Message, error) {
	if !w.IsConnected() {
		return nil, channels.ErrNotConnected
	}
	return w.MessageQueue(), nil
}

// handleEvent processes incoming WhatsApp events.
func (w *WhatsAppChannel) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		w.handleMessage(v)
	}
}

func (w *WhatsAppChannel) handleMessage(evt *events.Message) {
	// Ignore our own messages
	if evt.Info.IsFromMe {
		return
	}

	// Skip group messages unless configured to allow them
	if !w.allowGroups && evt.Info.Chat.Server == types.GroupServer {
		return
	}

	// Extract text content
	text := ""
	if conv := evt.Message.GetConversation(); conv != "" {
		text = conv
	} else if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}

	if text == "" {
		return // skip non-text messages
	}

	jid := evt.Info.Chat.String()

	msg := &channels.Message{
		ID:        evt.Info.ID,
		ChannelID: w.Name(), // "whatsapp" — for gateway routing
		UserID:    evt.Info.Sender.String(),
		Username:  evt.Info.PushName,
		Content:   text,
		Timestamp: evt.Info.Timestamp,
		Metadata: map[string]interface{}{
			"whatsapp_jid":    jid,
			"whatsapp_sender": evt.Info.Sender.String(),
		},
	}

	if err := w.QueueMessage(msg); err != nil {
		w.logger.Error().Err(err).Str("msg_id", msg.ID).Msg("failed to queue WhatsApp message")
	}
}
