package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore persists conversation history in Redis.
//
// Each user's history is stored as a Redis list at key "bucktooth:history:{userID}".
// Each element is a JSON-encoded Message. The list is capped at maxHistory entries
// and given a rolling TTL on every write.
type RedisStore struct {
	client     *redis.Client
	keyTTL     time.Duration
	maxHistory int
}

// NewRedisStore creates a new Redis-backed memory store.
func NewRedisStore(addr, password string, db int, ttl time.Duration, maxHistory int) (*RedisStore, error) {
	if addr == "" {
		addr = "localhost:6379"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if maxHistory <= 0 {
		maxHistory = 50
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	return &RedisStore{
		client:     client,
		keyTTL:     ttl,
		maxHistory: maxHistory,
	}, nil
}

func (s *RedisStore) key(userID string) string {
	return "bucktooth:history:" + userID
}

// AddMessage appends a message to the user's history and trims to maxHistory.
func (s *RedisStore) AddMessage(ctx context.Context, userID string, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	k := s.key(userID)
	pipe := s.client.Pipeline()
	pipe.RPush(ctx, k, data)
	pipe.LTrim(ctx, k, int64(-s.maxHistory), -1)
	pipe.Expire(ctx, k, s.keyTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// GetHistory returns the last limit messages for a user.
func (s *RedisStore) GetHistory(ctx context.Context, userID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = s.maxHistory
	}

	data, err := s.client.LRange(ctx, s.key(userID), int64(-limit), -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	messages := make([]Message, 0, len(data))
	for _, item := range data {
		var msg Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue // skip malformed entries
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ClearHistory deletes all history for a user.
func (s *RedisStore) ClearHistory(ctx context.Context, userID string) error {
	return s.client.Del(ctx, s.key(userID)).Err()
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}
