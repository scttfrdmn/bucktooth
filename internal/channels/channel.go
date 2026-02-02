package channels

import (
	"context"
	"time"
)

// Message represents a message in the system
type Message struct {
	ID          string                 `json:"id"`
	ChannelID   string                 `json:"channel_id"`
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Attachments []Attachment           `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
}

// HealthStatus represents the health of a channel
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
}

// Channel represents a messaging platform channel
type Channel interface {
	// Name returns the channel name (e.g., "discord", "whatsapp")
	Name() string

	// Connect establishes connection to the messaging platform
	Connect(ctx context.Context) error

	// Disconnect closes the connection gracefully
	Disconnect() error

	// SendMessage sends a message to the channel
	SendMessage(ctx context.Context, msg *Message) error

	// ReceiveMessages returns a channel that receives incoming messages
	ReceiveMessages(ctx context.Context) (<-chan *Message, error)

	// Health returns the current health status of the channel
	Health() HealthStatus
}

// ChannelRegistry manages multiple channels
type ChannelRegistry struct {
	channels map[string]Channel
}

// NewChannelRegistry creates a new channel registry
func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{
		channels: make(map[string]Channel),
	}
}

// Register adds a channel to the registry
func (r *ChannelRegistry) Register(channel Channel) {
	r.channels[channel.Name()] = channel
}

// Get retrieves a channel by name
func (r *ChannelRegistry) Get(name string) (Channel, bool) {
	ch, ok := r.channels[name]
	return ch, ok
}

// All returns all registered channels
func (r *ChannelRegistry) All() []Channel {
	channels := make([]Channel, 0, len(r.channels))
	for _, ch := range r.channels {
		channels = append(channels, ch)
	}
	return channels
}

// HealthCheck returns the health status of all channels
func (r *ChannelRegistry) HealthCheck() map[string]HealthStatus {
	health := make(map[string]HealthStatus)
	for name, ch := range r.channels {
		health[name] = ch.Health()
	}
	return health
}
