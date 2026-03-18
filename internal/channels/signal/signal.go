package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// jsonrpcRequest is a JSON-RPC 2.0 request with an ID.
type jsonrpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	ID      int            `json:"id"`
}

// jsonrpcNotification is a JSON-RPC 2.0 notification (no ID field).
type jsonrpcNotification struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// envelope is the top-level Signal message envelope from signal-cli.
type envelope struct {
	Source      string      `json:"source"`
	DataMessage dataMessage `json:"dataMessage"`
}

// dataMessage holds the text body of a Signal message.
type dataMessage struct {
	Message string `json:"message"`
}

// SignalChannel implements the channels.Channel interface for Signal via the
// signal-cli JSON-RPC WebSocket daemon (signal-cli ≥ 0.13.0 with --receive-mode on-start).
//
// Config example (gateway.yaml):
//
//	channels:
//	  signal:
//	    enabled: true
//	    auth:
//	      phone_number: "+15551234567"
//	      signald_url: "ws://localhost:2735"
type SignalChannel struct {
	*channels.BaseChannel
	phoneNumber string
	signaldURL  string
	conn        *websocket.Conn
	connMu      sync.RWMutex
	logger      zerolog.Logger
	nextID      int
	idMu        sync.Mutex
}

// New creates a new SignalChannel. phoneNumber is the registered Signal phone number;
// signaldURL is the WebSocket URL of the running signal-cli daemon.
func New(phoneNumber, signaldURL string, logger zerolog.Logger) *SignalChannel {
	base := channels.NewBaseChannel("signal", logger, 100)
	return &SignalChannel{
		BaseChannel: base,
		phoneNumber: phoneNumber,
		signaldURL:  signaldURL,
		logger:      logger.With().Str("channel", "signal").Logger(),
	}
}

// Connect dials the signald WebSocket and marks the channel as connected.
func (c *SignalChannel) Connect(ctx context.Context) error {
	if c.IsConnected() {
		return channels.ErrAlreadyConnected
	}
	if err := c.dial(ctx); err != nil {
		c.UpdateHealth(false, "failed", err)
		return fmt.Errorf("signal: connect: %w", err)
	}
	c.SetConnected(true)
	c.logger.Info().Str("url", c.signaldURL).Msg("connected to signald")
	return nil
}

// dial establishes a new WebSocket connection to signald.
func (c *SignalChannel) dial(ctx context.Context) error {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, c.signaldURL, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.signaldURL, err)
	}
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
	return nil
}

// Disconnect closes the WebSocket connection and stops the message queue.
func (c *SignalChannel) Disconnect() error {
	if !c.IsConnected() {
		return nil
	}
	c.connMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connMu.Unlock()

	c.SetConnected(false)
	if conn != nil {
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = conn.Close()
	}
	c.Close()
	c.logger.Info().Msg("disconnected from signald")
	return nil
}

// SendMessage sends a text message to the given recipient via signal-cli JSON-RPC.
// The recipient phone number is taken from msg.UserID; if empty, msg.ChannelID is used.
func (c *SignalChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return channels.ErrNotConnected
	}

	recipient := msg.UserID
	if recipient == "" {
		recipient = msg.ChannelID
	}

	c.idMu.Lock()
	c.nextID++
	id := c.nextID
	c.idMu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  "send",
		Params: map[string]any{
			"recipient": recipient,
			"message":   msg.Content,
		},
		ID: id,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("signal: marshal send request: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.UpdateHealth(false, "send error", err)
		return fmt.Errorf("signal: send: %w", err)
	}
	return nil
}

// ReceiveMessages returns the message queue channel and starts the background
// receive loop that reads from signald and re-connects on disconnect.
func (c *SignalChannel) ReceiveMessages(ctx context.Context) (<-chan *channels.Message, error) {
	if !c.IsConnected() {
		return nil, channels.ErrNotConnected
	}
	go c.receiveLoop(ctx)
	return c.MessageQueue(), nil
}

// receiveLoop reads from signald with exponential-backoff reconnect on failure.
func (c *SignalChannel) receiveLoop(ctx context.Context) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := c.readMessages(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Warn().Err(err).Dur("retry_in", backoff).Msg("signald connection lost, reconnecting")
			c.UpdateHealth(false, "reconnecting", err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			if err := c.dial(ctx); err != nil {
				c.logger.Error().Err(err).Msg("signal: reconnect failed")
				backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
				continue
			}
			c.SetConnected(true)
			c.logger.Info().Msg("signal: reconnected to signald")
			backoff = 500 * time.Millisecond
		}
	}
}

// readMessages blocks reading from the current WebSocket connection until it closes or ctx is cancelled.
func (c *SignalChannel) readMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()
		if conn == nil {
			return fmt.Errorf("connection is nil")
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		if err := c.processMessage(data); err != nil {
			c.logger.Warn().Err(err).Msg("signal: failed to process incoming message")
		}
	}
}

// processMessage decodes a JSON-RPC notification and enqueues a Message when it's
// a "receive" notification containing a non-empty text body.
func (c *SignalChannel) processMessage(data []byte) error {
	var notif jsonrpcNotification
	if err := json.Unmarshal(data, &notif); err != nil {
		return fmt.Errorf("unmarshal notification: %w", err)
	}
	if notif.Method != "receive" {
		return nil // ignore non-receive notifications (e.g. send acknowledgements)
	}

	env, err := extractEnvelope(notif.Params)
	if err != nil {
		return err
	}
	if env.DataMessage.Message == "" {
		return nil // receipt, typing indicator, etc.
	}

	msg := &channels.Message{
		ChannelID: c.Name(),
		UserID:    env.Source,
		Username:  env.Source,
		Content:   env.DataMessage.Message,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"signal_source": env.Source,
		},
	}
	return c.QueueMessage(msg)
}

// extractEnvelope unmarshals the "envelope" field from a JSON-RPC notification's params.
func extractEnvelope(params map[string]any) (envelope, error) {
	var env envelope
	envRaw, ok := params["envelope"]
	if !ok {
		return env, fmt.Errorf("missing 'envelope' field in receive params")
	}
	envBytes, err := json.Marshal(envRaw)
	if err != nil {
		return env, fmt.Errorf("re-marshal envelope: %w", err)
	}
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return env, fmt.Errorf("unmarshal envelope struct: %w", err)
	}
	return env, nil
}
