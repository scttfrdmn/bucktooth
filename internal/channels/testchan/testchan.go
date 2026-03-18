// Package testchan provides an HTTP-driven test channel for harness/integration testing.
// POST /test/send injects a message into the pipeline.
// GET  /test/responses retrieves (and clears) captured agent responses.
package testchan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// TestChannel is a no-credential channel whose messages are injected via HTTP.
type TestChannel struct {
	*channels.BaseChannel
	responses []string
	mu        sync.Mutex
	logger    zerolog.Logger
}

// New creates a TestChannel. It is registered with the gateway via gateway.New()
// when config.Gateway.TestChannel is true.
func New(logger zerolog.Logger) *TestChannel {
	base := channels.NewBaseChannel("test", logger, 100)
	return &TestChannel{
		BaseChannel: base,
		responses:   make([]string, 0),
		logger:      logger.With().Str("channel", "test").Logger(),
	}
}

// Connect implements channels.Channel.
func (tc *TestChannel) Connect(_ context.Context) error {
	tc.SetConnected(true)
	tc.logger.Info().Msg("test channel connected")
	return nil
}

// Disconnect implements channels.Channel.
func (tc *TestChannel) Disconnect() error {
	tc.SetConnected(false)
	tc.Close()
	tc.logger.Info().Msg("test channel disconnected")
	return nil
}

// SendMessage implements channels.Channel; it captures the response for retrieval.
func (tc *TestChannel) SendMessage(_ context.Context, msg *channels.Message) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.responses = append(tc.responses, msg.Content)
	tc.logger.Debug().Str("content", msg.Content).Msg("test channel captured response")
	return nil
}

// ReceiveMessages implements channels.Channel.
func (tc *TestChannel) ReceiveMessages(_ context.Context) (<-chan *channels.Message, error) {
	return tc.MessageQueue(), nil
}

// sendRequest is the JSON body for POST /test/send.
type sendRequest struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
}

// HandleSend handles POST /test/send.
func (tc *TestChannel) HandleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = "test-user"
	}

	msg := &channels.Message{
		ID:        fmt.Sprintf("test-%d", time.Now().UnixNano()),
		ChannelID: "test",
		UserID:    req.UserID,
		Username:  req.UserID,
		Content:   req.Content,
		Timestamp: time.Now(),
	}

	if err := tc.QueueMessage(msg); err != nil {
		http.Error(w, "queue full", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "queued"}); err != nil {
		tc.logger.Error().Err(err).Msg("failed to encode send response")
	}
}

// HandleResponses handles GET /test/responses, returning and clearing captured responses.
func (tc *TestChannel) HandleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tc.mu.Lock()
	resp := make([]string, len(tc.responses))
	copy(resp, tc.responses)
	tc.responses = tc.responses[:0]
	tc.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		tc.logger.Error().Err(err).Msg("failed to encode responses")
	}
}
