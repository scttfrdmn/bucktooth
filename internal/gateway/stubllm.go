package gateway

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// StubLLM is a test double implementing llm.LLM.
// In echo mode (fixedResponse == "") it returns "echo: <last user message>".
// In fixed-response mode it always returns the configured string.
type StubLLM struct {
	fixedResponse string
	modelName     string
}

// NewStubLLM creates a StubLLM. Pass an empty string for echo mode.
func NewStubLLM(fixedResponse string) *StubLLM {
	return &StubLLM{
		fixedResponse: fixedResponse,
		modelName:     "stub",
	}
}

// Complete implements llm.LLM.
func (s *StubLLM) Complete(_ context.Context, messages []*agenkit.Message, _ ...llm.CallOption) (*agenkit.Message, error) {
	if s.fixedResponse != "" {
		return agenkit.NewMessage("assistant", s.fixedResponse), nil
	}

	// Echo mode: echo the last user message.
	var lastUserContent string
	for _, msg := range messages {
		if msg.Role == "user" {
			lastUserContent = msg.ContentString()
		}
	}
	return agenkit.NewMessage("assistant", fmt.Sprintf("echo: %s", lastUserContent)), nil
}

// Stream implements llm.LLM by wrapping Complete in a buffered channel.
func (s *StubLLM) Stream(ctx context.Context, messages []*agenkit.Message, opts ...llm.CallOption) (<-chan *agenkit.Message, error) {
	msg, err := s.Complete(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan *agenkit.Message, 1)
	ch <- msg
	close(ch)
	return ch, nil
}

// Model implements llm.LLM.
func (s *StubLLM) Model() string { return s.modelName }

// Unwrap implements llm.LLM.
func (s *StubLLM) Unwrap() interface{} { return nil }
