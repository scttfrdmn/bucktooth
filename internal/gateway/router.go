package gateway

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/patterns"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
)

// AgentRouter routes messages to appropriate agents
type AgentRouter struct {
	conversationalAgent *patterns.ConversationalAgent
	memoryStore         memory.Store
	logger              zerolog.Logger
	config              config.AgentConfig
}

// llmClientAdapter wraps an LLM to implement the patterns.LLMClient interface
type llmClientAdapter struct {
	llm *llm.AnthropicLLM
}

func (a *llmClientAdapter) Chat(ctx context.Context, messages []*agenkit.Message) (*agenkit.Message, error) {
	return a.llm.Complete(ctx, messages)
}

// NewAgentRouter creates a new agent router
func NewAgentRouter(cfg config.AgentConfig, memStore memory.Store, logger zerolog.Logger) (*AgentRouter, error) {
	// Create LLM client based on provider
	var llmClient patterns.LLMClient

	switch cfg.LLMProvider {
	case "anthropic":
		anthropicLLM := llm.NewAnthropicLLM(cfg.APIKey, cfg.LLMModel)
		llmClient = &llmClientAdapter{llm: anthropicLLM}
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}

	// Create conversational agent
	conversationalAgent, err := patterns.NewConversationalAgent(&patterns.ConversationalAgentConfig{
		LLMClient:    llmClient,
		SystemPrompt: "You are a helpful AI assistant. You are friendly, concise, and helpful.",
		MaxHistory:   cfg.MaxHistory,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create conversational agent: %w", err)
	}

	return &AgentRouter{
		conversationalAgent: conversationalAgent,
		memoryStore:         memStore,
		logger:              logger.With().Str("component", "router").Logger(),
		config:              cfg,
	}, nil
}

// ProcessMessage processes a message and returns a response
func (ar *AgentRouter) ProcessMessage(ctx context.Context, msg *channels.Message) (string, error) {
	ar.logger.Info().
		Str("user_id", msg.UserID).
		Str("channel_id", msg.ChannelID).
		Str("content", msg.Content).
		Msg("processing message")

	// Create agenkit message
	agentMessage := &agenkit.Message{
		Role:    "user",
		Content: msg.Content,
	}

	// Process with conversational agent (manages its own history)
	response, err := ar.conversationalAgent.Process(ctx, agentMessage)
	if err != nil {
		ar.logger.Error().Err(err).Msg("agent processing failed")
		return "", fmt.Errorf("agent processing failed: %w", err)
	}

	// Extract response content
	responseText := response.Content

	// Store in memory for cross-channel persistence
	if err := ar.memoryStore.AddMessage(ctx, msg.UserID, memory.Message{
		Role:      "user",
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}); err != nil {
		ar.logger.Error().Err(err).Msg("failed to store user message")
	}

	if err := ar.memoryStore.AddMessage(ctx, msg.UserID, memory.Message{
		Role:      "assistant",
		Content:   responseText,
		Timestamp: msg.Timestamp,
	}); err != nil {
		ar.logger.Error().Err(err).Msg("failed to store assistant message")
	}

	ar.logger.Info().
		Str("user_id", msg.UserID).
		Str("response", responseText).
		Msg("message processed successfully")

	return responseText, nil
}

// Close cleans up resources
func (ar *AgentRouter) Close() error {
	// Cleanup if needed
	return nil
}
