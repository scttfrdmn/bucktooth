package channels

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// BaseChannel provides common functionality for all channels
type BaseChannel struct {
	name          string
	logger        zerolog.Logger
	healthStatus  HealthStatus
	healthMu      sync.RWMutex
	messageQueue  chan *Message
	queueSize     int
	connected     bool
	connectedMu   sync.RWMutex
}

// NewBaseChannel creates a new base channel
func NewBaseChannel(name string, logger zerolog.Logger, queueSize int) *BaseChannel {
	if queueSize <= 0 {
		queueSize = 100
	}

	return &BaseChannel{
		name:         name,
		logger:       logger.With().Str("channel", name).Logger(),
		queueSize:    queueSize,
		messageQueue: make(chan *Message, queueSize),
		healthStatus: HealthStatus{
			Healthy:   false,
			Status:    "disconnected",
			LastCheck: time.Now(),
		},
	}
}

// Name returns the channel name
func (b *BaseChannel) Name() string {
	return b.name
}

// Health returns the current health status
func (b *BaseChannel) Health() HealthStatus {
	b.healthMu.RLock()
	defer b.healthMu.RUnlock()
	return b.healthStatus
}

// UpdateHealth updates the health status
func (b *BaseChannel) UpdateHealth(healthy bool, status string, err error) {
	b.healthMu.Lock()
	defer b.healthMu.Unlock()

	b.healthStatus = HealthStatus{
		Healthy:   healthy,
		Status:    status,
		LastCheck: time.Now(),
	}

	if err != nil {
		b.healthStatus.Error = err.Error()
	}
}

// IsConnected returns whether the channel is connected
func (b *BaseChannel) IsConnected() bool {
	b.connectedMu.RLock()
	defer b.connectedMu.RUnlock()
	return b.connected
}

// SetConnected sets the connection status
func (b *BaseChannel) SetConnected(connected bool) {
	b.connectedMu.Lock()
	defer b.connectedMu.Unlock()
	b.connected = connected

	if connected {
		b.UpdateHealth(true, "connected", nil)
	} else {
		b.UpdateHealth(false, "disconnected", nil)
	}
}

// QueueMessage adds a message to the queue
func (b *BaseChannel) QueueMessage(msg *Message) error {
	select {
	case b.messageQueue <- msg:
		return nil
	default:
		b.logger.Warn().Str("msg_id", msg.ID).Msg("message queue full, dropping message")
		return ErrQueueFull
	}
}

// MessageQueue returns the message queue channel
func (b *BaseChannel) MessageQueue() <-chan *Message {
	return b.messageQueue
}

// Close closes the message queue
func (b *BaseChannel) Close() {
	close(b.messageQueue)
}

// Common errors
var (
	ErrQueueFull      = &ChannelError{Code: "QUEUE_FULL", Message: "message queue is full"}
	ErrNotConnected   = &ChannelError{Code: "NOT_CONNECTED", Message: "channel is not connected"}
	ErrAlreadyConnected = &ChannelError{Code: "ALREADY_CONNECTED", Message: "channel is already connected"}
)

// ChannelError represents a channel-specific error
type ChannelError struct {
	Code    string
	Message string
	Err     error
}

func (e *ChannelError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ChannelError) Unwrap() error {
	return e.Err
}
