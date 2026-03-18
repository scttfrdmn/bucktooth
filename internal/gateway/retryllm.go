package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// RetryLLM wraps an llm.LLM and retries Complete calls on transient errors
// using exponential backoff with ±20% jitter.
type RetryLLM struct {
	inner          llm.LLM
	maxAttempts    int
	initialBackoff time.Duration
	logger         zerolog.Logger
}

// NewRetryLLM creates a RetryLLM. maxAttempts=1 means no retries (one attempt).
func NewRetryLLM(inner llm.LLM, maxAttempts int, initialBackoff time.Duration, logger zerolog.Logger) *RetryLLM {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if initialBackoff <= 0 {
		initialBackoff = 500 * time.Millisecond
	}
	return &RetryLLM{
		inner:          inner,
		maxAttempts:    maxAttempts,
		initialBackoff: initialBackoff,
		logger:         logger.With().Str("component", "retry_llm").Logger(),
	}
}

// Complete implements llm.LLM with retry on transient errors.
func (r *RetryLLM) Complete(ctx context.Context, messages []*agenkit.Message, opts ...llm.CallOption) (*agenkit.Message, error) {
	var lastErr error
	for attempt := 0; attempt < r.maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := r.backoffDuration(attempt)
			r.logger.Warn().
				Int("attempt", attempt).
				Dur("backoff", backoff).
				Err(lastErr).
				Msg("retry attempt")
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		msg, err := r.inner.Complete(ctx, messages, opts...)
		if err == nil {
			return msg, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("all %d attempts failed: %w", r.maxAttempts, lastErr)
}

// Stream implements llm.LLM by delegating directly (no retry on streams).
func (r *RetryLLM) Stream(ctx context.Context, messages []*agenkit.Message, opts ...llm.CallOption) (<-chan *agenkit.Message, error) {
	return r.inner.Stream(ctx, messages, opts...)
}

// Model implements llm.LLM.
func (r *RetryLLM) Model() string { return r.inner.Model() }

// Unwrap implements llm.LLM.
func (r *RetryLLM) Unwrap() interface{} { return r.inner }

// backoffDuration returns exponential backoff for attempt n with ±20% jitter.
func (r *RetryLLM) backoffDuration(attempt int) time.Duration {
	base := r.initialBackoff * (1 << uint(attempt-1))
	// ±20% jitter: multiply by a factor in [0.8, 1.2).
	jitter := 0.8 + rand.Float64()*0.4 //nolint:gosec // jitter doesn't need crypto random
	return time.Duration(float64(base) * jitter)
}

// isRetryableError returns true for errors that warrant a retry:
// HTTP 429 (rate limit), 503 (service unavailable), 529 (overloaded),
// and context deadline exceeded.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "529") ||
		strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "rate limit")
}
