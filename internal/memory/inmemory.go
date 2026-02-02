package memory

import (
	"context"
	"sync"
)

// InMemoryStore implements Store using in-memory storage
type InMemoryStore struct {
	histories map[string][]Message
	mu        sync.RWMutex
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		histories: make(map[string][]Message),
	}
}

// AddMessage adds a message to a user's conversation history
func (s *InMemoryStore) AddMessage(ctx context.Context, userID string, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.histories[userID] = append(s.histories[userID], message)
	return nil
}

// GetHistory retrieves the last N messages for a user
func (s *InMemoryStore) GetHistory(ctx context.Context, userID string, limit int) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.histories[userID]
	if !ok {
		return []Message{}, nil
	}

	// Return last N messages
	if len(history) <= limit {
		result := make([]Message, len(history))
		copy(result, history)
		return result, nil
	}

	result := make([]Message, limit)
	copy(result, history[len(history)-limit:])
	return result, nil
}

// ClearHistory clears all messages for a user
func (s *InMemoryStore) ClearHistory(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.histories, userID)
	return nil
}

// Close closes the store
func (s *InMemoryStore) Close() error {
	return nil
}
