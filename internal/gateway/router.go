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
	"github.com/scttfrdmn/bucktooth/internal/tools"
)

const systemPrompt = "You are a helpful AI assistant. You are friendly, concise, and helpful."

// AgentRouter routes messages to appropriate agents
type AgentRouter struct {
	conversationalAgent *patterns.ConversationalAgent
	llmRaw              *llm.AnthropicLLM
	registry            *tools.Registry
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

// llmAgent wraps AnthropicLLM as a stateless agenkit.Agent for use inside ReActAgent.
type llmAgent struct {
	llm *llm.AnthropicLLM
}

func (a *llmAgent) Name() string         { return "LLMAgent" }
func (a *llmAgent) Capabilities() []string { return []string{"llm"} }
func (a *llmAgent) Introspect() *agenkit.IntrospectionResult {
	return &agenkit.IntrospectionResult{AgentName: "LLMAgent", Capabilities: a.Capabilities()}
}
func (a *llmAgent) Process(ctx context.Context, msg *agenkit.Message) (*agenkit.Message, error) {
	return a.llm.Complete(ctx, []*agenkit.Message{msg})
}

// NewAgentRouter creates a new agent router
func NewAgentRouter(cfg config.AgentConfig, memStore memory.Store, registry *tools.Registry, logger zerolog.Logger) (*AgentRouter, error) {
	// Create LLM client based on provider
	switch cfg.LLMProvider {
	case "anthropic":
		// ok
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}

	anthropicLLM := llm.NewAnthropicLLM(cfg.APIKey, cfg.LLMModel)
	llmClient := &llmClientAdapter{llm: anthropicLLM}

	// Create conversational agent (used as fallback when no tools are registered)
	conversationalAgent, err := patterns.NewConversationalAgent(&patterns.ConversationalAgentConfig{
		LLMClient:    llmClient,
		SystemPrompt: systemPrompt,
		MaxHistory:   cfg.MaxHistory,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create conversational agent: %w", err)
	}

	return &AgentRouter{
		conversationalAgent: conversationalAgent,
		llmRaw:              anthropicLLM,
		registry:            registry,
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

	agentMessage := &agenkit.Message{
		Role:    "user",
		Content: msg.Content,
	}

	var responseText string

	if ar.registry != nil && ar.registry.Enabled() {
		// Tool-augmented path: use ReActAgent
		inner := &llmAgent{llm: ar.llmRaw}
		reactAgent, err := patterns.NewReActAgent(&patterns.ReActConfig{
			Agent:    inner,
			Tools:    ar.registry.GetAll(),
			MaxSteps: 5,
		})
		if err != nil {
			ar.logger.Error().Err(err).Msg("failed to create ReActAgent, falling back to conversational")
			goto conversational
		}

		response, err := reactAgent.Process(ctx, agentMessage)
		if err != nil {
			ar.logger.Error().Err(err).Msg("ReActAgent processing failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}

		responseText = response.ContentString()
		goto store
	}

conversational:
	{
		// Plain conversational path
		response, err := ar.conversationalAgent.Process(ctx, agentMessage)
		if err != nil {
			ar.logger.Error().Err(err).Msg("agent processing failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		responseText = response.ContentString()
	}

store:
	// Persist turn to memory
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
	return nil
}
