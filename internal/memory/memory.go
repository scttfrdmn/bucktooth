package memory

import (
	"context"
	"time"
)

// Message represents a stored message
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Store is the interface for memory storage
type Store interface {
	// AddMessage adds a message to a user's conversation history
	AddMessage(ctx context.Context, userID string, message Message) error

	// GetHistory retrieves the last N messages for a user
	GetHistory(ctx context.Context, userID string, limit int) ([]Message, error)

	// ClearHistory clears all messages for a user
	ClearHistory(ctx context.Context, userID string) error

	// Close closes the store
	Close() error
}
