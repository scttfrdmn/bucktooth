package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
)

// Summarizer compresses long conversation histories using the LLM.
//
// When the number of stored messages for a user reaches or exceeds Threshold,
// or when the estimated token count reaches TokenThreshold (if > 0),
// MaybeSummarize fires an async goroutine that replaces the full history with
// a single system message containing an LLM-generated summary. A per-user guard
// prevents concurrent summarization jobs for the same user.
type Summarizer struct {
	store          Store
	llm            llm.LLM
	threshold      int
	tokenThreshold int // trigger on estimated token count; 0=disabled
	logger         zerolog.Logger
	active         sync.Map // map[userID string]bool — in-progress guard
}

// NewSummarizer creates a Summarizer. threshold is the minimum history length
// that triggers summarization (values ≤ 0 default to 30). tokenThreshold, when
// > 0, additionally triggers on estimated token count (≈ chars / 4).
func NewSummarizer(store Store, llmInstance llm.LLM, threshold int, tokenThreshold int, logger zerolog.Logger) *Summarizer {
	if threshold <= 0 {
		threshold = 30
	}
	return &Summarizer{
		store:          store,
		llm:            llmInstance,
		threshold:      threshold,
		tokenThreshold: tokenThreshold,
		logger:         logger.With().Str("component", "summarizer").Logger(),
	}
}

// MaybeSummarize fires an async summarization job for userID if the history
// length meets or exceeds the threshold (or the estimated token count meets
// tokenThreshold, when configured), and no job is already running for that user.
func (s *Summarizer) MaybeSummarize(ctx context.Context, userID string) {
	history, err := s.store.GetHistory(ctx, userID, s.threshold+1)
	if err != nil {
		return
	}

	triggered := len(history) >= s.threshold
	if !triggered && s.tokenThreshold > 0 {
		est := 0
		for _, m := range history {
			est += len(m.Content)
		}
		triggered = est/4 >= s.tokenThreshold // rough 4-chars-per-token estimate
	}
	if !triggered {
		return
	}

	// Guard: only one job per user at a time.
	if _, loaded := s.active.LoadOrStore(userID, true); loaded {
		return
	}

	go s.summarize(context.Background(), userID)
}

func (s *Summarizer) summarize(ctx context.Context, userID string) {
	defer s.active.Delete(userID)

	// Use a large limit to retrieve the full history (limit=0 means "no rows" in some backends).
	history, err := s.store.GetHistory(ctx, userID, 1<<20)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("summarizer: failed to get history")
		return
	}
	// Re-check both triggers to guard against races (e.g. another goroutine
	// may have already summarized between MaybeSummarize and this goroutine running).
	triggered := len(history) >= s.threshold
	if !triggered && s.tokenThreshold > 0 {
		est := 0
		for _, m := range history {
			est += len(m.Content)
		}
		triggered = est/4 >= s.tokenThreshold
	}
	if !triggered {
		return
	}

	originalCount := len(history)

	// Build prompt for the LLM.
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation concisely, preserving all important context, decisions, and facts:\n\n")
	for _, m := range history {
		sb.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(m.Role), m.Content))
	}

	promptMsg := &agenkit.Message{
		Role:      "user",
		Content:   sb.String(),
		Timestamp: time.Now().UTC(),
	}

	response, err := s.llm.Complete(ctx, []*agenkit.Message{promptMsg})
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("summarizer: LLM call failed")
		return
	}

	summary := response.ContentString()

	// Replace history with summary.
	if err := s.store.ClearHistory(ctx, userID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("summarizer: failed to clear history")
		return
	}

	summaryMsg := Message{
		Role:      "system",
		Content:   "Previous conversation summary: " + summary,
		Timestamp: time.Now().UTC(),
	}
	if err := s.store.AddMessage(ctx, userID, summaryMsg); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("summarizer: failed to store summary")
		return
	}

	s.logger.Info().
		Str("user_id", userID).
		Int("original_messages", originalCount).
		Float64("compression_ratio", float64(1)/float64(originalCount)).
		Msg("conversation summarized")
}
