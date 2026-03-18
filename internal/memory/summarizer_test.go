package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// stubLLM satisfies llm.LLM for testing.
type stubLLM struct{ resp string }

func (s *stubLLM) Complete(_ context.Context, _ []*agenkit.Message, _ ...llm.CallOption) (*agenkit.Message, error) {
	return agenkit.NewMessage("assistant", s.resp), nil
}
func (s *stubLLM) Stream(_ context.Context, msgs []*agenkit.Message, opts ...llm.CallOption) (<-chan *agenkit.Message, error) {
	msg, err := s.Complete(context.Background(), msgs, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan *agenkit.Message, 1)
	ch <- msg
	close(ch)
	return ch, nil
}
func (s *stubLLM) Model() string       { return "stub" }
func (s *stubLLM) Unwrap() interface{} { return nil }

func addMessages(t *testing.T, store Store, userID string, n int, contentLen int) {
	t.Helper()
	for i := 0; i < n; i++ {
		err := store.AddMessage(context.Background(), userID, Message{
			Role:      "user",
			Content:   strings.Repeat("x", contentLen),
			Timestamp: time.Now(),
		})
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}
}

func TestSummarizer_TokenThreshold_Triggers(t *testing.T) {
	store := NewInMemoryStore()
	llmInst := &stubLLM{resp: "summary"}
	logger := zerolog.Nop()

	// threshold=10 (will not be reached), tokenThreshold=200 tokens (~800 chars)
	s := NewSummarizer(store, llmInst, 10, 200, logger)

	// 3 messages × 400 chars = ~300 tokens → should trigger token threshold
	addMessages(t, store, "u1", 3, 400)

	// MaybeSummarize fires async goroutine; give it a moment.
	s.MaybeSummarize(context.Background(), "u1")

	// Wait briefly for the goroutine to finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		history, _ := store.GetHistory(context.Background(), "u1", 100)
		if len(history) == 1 && history[0].Role == "system" {
			return // summarization happened
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("summarization did not happen after token threshold exceeded")
}

func TestSummarizer_MessageThreshold_Triggers(t *testing.T) {
	store := NewInMemoryStore()
	llmInst := &stubLLM{resp: "summary"}
	logger := zerolog.Nop()

	// threshold=3, tokenThreshold=0 (disabled)
	s := NewSummarizer(store, llmInst, 3, 0, logger)

	addMessages(t, store, "u2", 3, 10) // exactly 3 messages — threshold met
	s.MaybeSummarize(context.Background(), "u2")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		history, _ := store.GetHistory(context.Background(), "u2", 100)
		if len(history) == 1 && history[0].Role == "system" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("summarization did not happen after message threshold exceeded")
}

func TestSummarizer_BelowBothThresholds_NoTrigger(t *testing.T) {
	store := NewInMemoryStore()
	llmInst := &stubLLM{resp: "should not be called"}
	logger := zerolog.Nop()

	// threshold=10, tokenThreshold=5000 — both comfortably above what we add
	s := NewSummarizer(store, llmInst, 10, 5000, logger)

	// 2 messages × 10 chars = ~5 tokens — well below both thresholds
	addMessages(t, store, "u3", 2, 10)
	s.MaybeSummarize(context.Background(), "u3")

	// Give a brief window; history should still be 2 messages.
	time.Sleep(50 * time.Millisecond)
	history, _ := store.GetHistory(context.Background(), "u3", 100)
	if len(history) != 2 {
		t.Errorf("expected 2 messages (no summarization), got %d", len(history))
	}
}
