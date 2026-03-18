package gateway

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/patterns"
	"github.com/scttfrdmn/bucktooth/internal/agents"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
	"github.com/scttfrdmn/bucktooth/internal/tools"
	"go.opentelemetry.io/otel"
)

const systemPrompt = "You are a helpful AI assistant. You are friendly, concise, and helpful."

// AgentRouter routes messages to appropriate agents
type AgentRouter struct {
	userAgents   map[string]*patterns.ConversationalAgent
	userAgentsMu sync.RWMutex
	llmClient    *llmClientAdapter // stored for lazy per-user agent creation
	llmRaw       llm.LLM
	registry     *tools.Registry
	memoryStore  memory.Store
	logger       zerolog.Logger
	config       config.AgentConfig
}

// llmClientAdapter wraps an LLM to implement the patterns.LLMClient interface
type llmClientAdapter struct {
	llm llm.LLM
}

func (a *llmClientAdapter) Chat(ctx context.Context, messages []*agenkit.Message) (*agenkit.Message, error) {
	return a.llm.Complete(ctx, messages)
}

// llmAgent wraps an llm.LLM as a stateless agenkit.Agent for use inside ReActAgent.
type llmAgent struct {
	llm llm.LLM
}

func (a *llmAgent) Name() string           { return "LLMAgent" }
func (a *llmAgent) Capabilities() []string { return []string{"llm"} }
func (a *llmAgent) Introspect() *agenkit.IntrospectionResult {
	return &agenkit.IntrospectionResult{AgentName: "LLMAgent", Capabilities: a.Capabilities()}
}
func (a *llmAgent) Process(ctx context.Context, msg *agenkit.Message) (*agenkit.Message, error) {
	return a.llm.Complete(ctx, []*agenkit.Message{msg})
}

// NewAgentRouter creates a new agent router
func NewAgentRouter(cfg config.AgentConfig, memStore memory.Store, registry *tools.Registry, logger zerolog.Logger) (*AgentRouter, error) {
	var llmInstance llm.LLM
	switch cfg.LLMProvider {
	case "stub":
		llmInstance = NewStubLLM(cfg.StubResponse)
	case "anthropic":
		// TODO: pass cfg.APIBase to agenkit when it exposes a base URL option
		if cfg.APIBase != "" {
			logger.Info().Str("api_base", cfg.APIBase).Msg("ANTHROPIC_API_BASE configured; pending agenkit-go support to activate")
		}
		llmInstance = llm.NewAnthropicLLM(cfg.APIKey, cfg.LLMModel)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}

	llmClient := &llmClientAdapter{llm: llmInstance}

	return &AgentRouter{
		userAgents:  make(map[string]*patterns.ConversationalAgent),
		llmClient:   llmClient,
		llmRaw:      llmInstance,
		registry:    registry,
		memoryStore: memStore,
		logger:      logger.With().Str("component", "router").Logger(),
		config:      cfg,
	}, nil
}

// getOrCreateAgent returns the per-user ConversationalAgent, creating it on first use.
func (ar *AgentRouter) getOrCreateAgent(userID string) *patterns.ConversationalAgent {
	ar.userAgentsMu.RLock()
	agent, ok := ar.userAgents[userID]
	ar.userAgentsMu.RUnlock()
	if ok {
		return agent
	}
	ar.userAgentsMu.Lock()
	defer ar.userAgentsMu.Unlock()
	// Double-check after acquiring write lock.
	if agent, ok = ar.userAgents[userID]; ok {
		return agent
	}
	agent, _ = patterns.NewConversationalAgent(&patterns.ConversationalAgentConfig{
		LLMClient:    ar.llmClient,
		SystemPrompt: systemPrompt,
		MaxHistory:   ar.config.MaxHistory,
	})
	ar.userAgents[userID] = agent
	return agent
}

// ActiveUsers returns the number of distinct users with active agent instances.
func (ar *AgentRouter) ActiveUsers() int {
	ar.userAgentsMu.RLock()
	defer ar.userAgentsMu.RUnlock()
	return len(ar.userAgents)
}

// ProcessMessage processes a message and returns a response
func (ar *AgentRouter) ProcessMessage(ctx context.Context, msg *channels.Message) (string, error) {
	ctx, span := otel.Tracer("bucktooth/router").Start(ctx, "router.process_message")
	defer span.End()

	ar.logger.Info().
		Str("user_id", msg.UserID).
		Str("channel_id", msg.ChannelID).
		Str("content", msg.Content).
		Msg("processing message")

	content := msg.Content
	agentMessage := &agenkit.Message{
		Role:    "user",
		Content: content,
	}

	var responseText string

	// /plan prefix: activate planning agent for this message regardless of global mode.
	if strings.HasPrefix(content, "/plan ") && ar.registry != nil {
		planContent := strings.TrimPrefix(content, "/plan ")
		agentMessage = &agenkit.Message{Role: "user", Content: planContent}
		executor := agents.NewToolStepExecutor(ar.registry, ar.llmRaw)
		llmClient := &llmClientAdapter{llm: ar.llmRaw}
		planAgent := agents.NewBuckToothPlanningAgent(llmClient, executor, 10)
		response, err := planAgent.Process(ctx, agentMessage)
		if err != nil {
			ar.logger.Error().Err(err).Msg("planning agent failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		responseText = response.ContentString()
		goto store
	}

	// Planning mode: decompose task into steps and execute via ToolStepExecutor.
	if ar.config.Mode == "planning" && ar.registry != nil {
		executor := agents.NewToolStepExecutor(ar.registry, ar.llmRaw)
		llmClient := &llmClientAdapter{llm: ar.llmRaw}
		planAgent := agents.NewBuckToothPlanningAgent(llmClient, executor, 10)
		response, err := planAgent.Process(ctx, agentMessage)
		if err != nil {
			ar.logger.Error().Err(err).Msg("planning agent failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		responseText = response.ContentString()
		goto store
	}

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
		// Plain conversational path — each user gets their own isolated agent instance.
		agent := ar.getOrCreateAgent(msg.UserID)
		response, err := agent.Process(ctx, agentMessage)
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
