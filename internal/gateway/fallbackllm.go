package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// FallbackLLM tries providers in order, moving to the next on non-retryable
// provider failures (HTTP 500, 503, model-not-found). HTTP 429 is NOT a
// fallback trigger — it should be handled by RetryLLM instead.
type FallbackLLM struct {
	providers []llm.LLM
	logger    zerolog.Logger
}

// NewFallbackLLM creates a FallbackLLM wrapping the given providers.
// The first provider is the primary; subsequent providers are fallbacks.
func NewFallbackLLM(providers []llm.LLM, logger zerolog.Logger) *FallbackLLM {
	return &FallbackLLM{
		providers: providers,
		logger:    logger.With().Str("component", "fallback_llm").Logger(),
	}
}

// Complete implements llm.LLM with provider fallback.
func (f *FallbackLLM) Complete(ctx context.Context, messages []*agenkit.Message, opts ...llm.CallOption) (*agenkit.Message, error) {
	var lastErr error
	for i, p := range f.providers {
		msg, err := p.Complete(ctx, messages, opts...)
		if err == nil {
			return msg, nil
		}
		lastErr = err
		if !isFallbackableError(err) {
			// Non-fallbackable error (e.g. 429); return immediately.
			return nil, err
		}
		f.logger.Warn().
			Int("provider_index", i).
			Str("model", p.Model()).
			Err(err).
			Msg("provider failed, trying next fallback")
	}
	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// Stream implements llm.LLM by delegating to the first provider (no fallback on streams).
func (f *FallbackLLM) Stream(ctx context.Context, messages []*agenkit.Message, opts ...llm.CallOption) (<-chan *agenkit.Message, error) {
	if len(f.providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}
	return f.providers[0].Stream(ctx, messages, opts...)
}

// Model implements llm.LLM — returns the primary provider's model name.
func (f *FallbackLLM) Model() string {
	if len(f.providers) == 0 {
		return "fallback"
	}
	return f.providers[0].Model()
}

// Unwrap implements llm.LLM — returns the primary provider.
func (f *FallbackLLM) Unwrap() interface{} {
	if len(f.providers) == 0 {
		return nil
	}
	return f.providers[0]
}

// isFallbackableError returns true for errors that should trigger provider
// fallback: HTTP 500 (server error), 503 (service unavailable), and
// model-not-found errors. HTTP 429 is deliberately excluded — it means the
// provider is alive but busy, so RetryLLM should handle it.
func isFallbackableError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "500") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "model not found") ||
		strings.Contains(s, "model_not_found") ||
		strings.Contains(s, "no such model")
}
