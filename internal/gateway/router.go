package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/patterns"
	"github.com/scttfrdmn/agenkit/agenkit-go/skills"
	"github.com/scttfrdmn/bucktooth/internal/agents"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
	"github.com/scttfrdmn/bucktooth/internal/observability"
	"github.com/scttfrdmn/bucktooth/internal/tools"
	"go.opentelemetry.io/otel"
)

const systemPrompt = "You are a helpful AI assistant. You are friendly, concise, and helpful."

// AgentRouter routes messages to appropriate agents
type AgentRouter struct {
	userAgents      map[string]*patterns.ConversationalAgent
	userAgentsMu    sync.RWMutex
	llmClient       *llmClientAdapter // stored for lazy per-user agent creation
	llmRaw          llm.LLM
	registry        *tools.Registry
	memoryStore     memory.Store
	stats           *Stats
	logger          zerolog.Logger
	config          config.AgentConfig
	skillRegistry   *skills.SkillRegistry
	skillsMaxActive int
	summarizer      *memory.Summarizer
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

// NewAgentRouter creates a new agent router. It returns the router, the
// underlying llm.LLM instance (for use by the summarizer), and any error.
func NewAgentRouter(cfg config.AgentConfig, skillsCfg config.SkillsConfig, memStore memory.Store, registry *tools.Registry, stats *Stats, logger zerolog.Logger) (*AgentRouter, llm.LLM, error) {
	var llmInstance llm.LLM
	switch cfg.LLMProvider {
	case "stub":
		llmInstance = NewStubLLM(cfg.StubResponse)
	case "anthropic":
		opts := []llm.AnthropicOption{}
		if cfg.APIBase != "" {
			opts = append(opts, llm.WithBaseURL(cfg.APIBase))
		}
		llmInstance = llm.NewAnthropicLLM(cfg.APIKey, cfg.LLMModel, opts...)
	case "openai":
		llmInstance = llm.NewOpenAILLM(cfg.APIKey, cfg.LLMModel)
	case "openai-compatible":
		llmInstance = llm.NewOpenAICompatibleLLM(cfg.APIBase, cfg.LLMModel, "openai-compatible", cfg.APIKey)
	case "gemini":
		g, err := llm.NewGeminiLLM(cfg.APIKey, cfg.LLMModel)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Gemini LLM: %w", err)
		}
		llmInstance = g
	case "ollama":
		llmInstance = llm.NewOllamaLLM(cfg.LLMModel, cfg.APIBase)
	case "litellm":
		llmInstance = llm.NewLiteLLMLLMWithAuth(cfg.APIBase, cfg.LLMModel, cfg.APIKey)
	case "bedrock":
		b, err := llm.NewBedrockLLM(context.Background(), llm.BedrockConfig{ModelID: cfg.LLMModel})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Bedrock LLM: %w", err)
		}
		llmInstance = b
	default:
		return nil, nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}

	llmClient := &llmClientAdapter{llm: llmInstance}

	ar := &AgentRouter{
		userAgents:  make(map[string]*patterns.ConversationalAgent),
		llmClient:   llmClient,
		llmRaw:      llmInstance,
		registry:    registry,
		memoryStore: memStore,
		stats:       stats,
		logger:      logger.With().Str("component", "router").Logger(),
		config:      cfg,
	}

	// Initialise skill registry if enabled.
	if skillsCfg.Enabled {
		searchPaths := make([]string, len(skillsCfg.SearchPaths))
		for i, p := range skillsCfg.SearchPaths {
			if strings.HasPrefix(p, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					p = filepath.Join(home, p[2:])
				}
			}
			searchPaths[i] = p
		}
		reg := skills.NewSkillRegistry(searchPaths)
		if err := reg.DiscoverSkills(); err != nil {
			logger.Warn().Err(err).Msg("skill discovery returned error (non-fatal)")
		}
		discovered := reg.All()
		ar.skillRegistry = reg
		ar.skillsMaxActive = skillsCfg.MaxActiveSkills
		if ar.skillsMaxActive <= 0 {
			ar.skillsMaxActive = 3
		}
		logger.Info().Int("skills", len(discovered)).Msg("skill registry initialised")
	}

	return ar, llmInstance, nil
}

// SkillRegistry returns the router's skill registry (may be nil if skills are disabled).
func (ar *AgentRouter) SkillRegistry() *skills.SkillRegistry {
	return ar.skillRegistry
}

// SetSummarizer attaches a memory Summarizer that is called after each message turn.
func (ar *AgentRouter) SetSummarizer(s *memory.Summarizer) {
	ar.summarizer = s
}

// maybeInjectSkills prepends relevant skill instructions to the message content
// when skills are enabled. Returns the original message unchanged if no skills
// are loaded or no relevant skills are found for the query.
func (ar *AgentRouter) maybeInjectSkills(msg *agenkit.Message) *agenkit.Message {
	if ar.skillRegistry == nil {
		return msg
	}
	query := msg.ContentString()
	relevant := ar.skillRegistry.FindRelevantSkills(query, ar.skillsMaxActive)
	if len(relevant) == 0 {
		return msg
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, s := range relevant {
		sb.WriteString(s.ToPrompt())
		sb.WriteString("\n")
	}
	sb.WriteString("</available_skills>\n\n")
	sb.WriteString(query)

	enhanced := &agenkit.Message{
		Role:      msg.Role,
		Content:   sb.String(),
		Timestamp: msg.Timestamp,
		Metadata:  make(map[string]interface{}),
	}
	for k, v := range msg.Metadata {
		enhanced.Metadata[k] = v
	}
	return enhanced
}

// extractTokenUsage reads input/output token counts from an agenkit Message's metadata.
// Returns (0, 0) for providers that don't populate usage metadata.
func extractTokenUsage(msg *agenkit.Message) (in, out uint64) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	usage, ok := msg.Metadata["usage"].(map[string]interface{})
	if !ok {
		return
	}
	if v, ok := usage["input_tokens"].(int); ok {
		in = uint64(v)
	}
	if v, ok := usage["output_tokens"].(int); ok {
		out = uint64(v)
	}
	return
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
		agentMessage = ar.maybeInjectSkills(&agenkit.Message{Role: "user", Content: planContent})
		executor := agents.NewToolStepExecutor(ar.registry, ar.llmRaw)
		llmClient := &llmClientAdapter{llm: ar.llmRaw}
		planAgent := agents.NewBuckToothPlanningAgent(llmClient, executor, 10)
		response, err := planAgent.Process(ctx, agentMessage)
		if err != nil {
			ar.logger.Error().Err(err).Msg("planning agent failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		if ar.stats != nil {
			if in, out := extractTokenUsage(response); in > 0 || out > 0 {
				ar.stats.RecordTokens(in, out)
			}
		}
		responseText = response.ContentString()
		goto store
	}

	// Planning mode: decompose task into steps and execute via ToolStepExecutor.
	if ar.config.Mode == "planning" && ar.registry != nil {
		executor := agents.NewToolStepExecutor(ar.registry, ar.llmRaw)
		llmClient := &llmClientAdapter{llm: ar.llmRaw}
		planAgent := agents.NewBuckToothPlanningAgent(llmClient, executor, 10)
		response, err := planAgent.Process(ctx, ar.maybeInjectSkills(agentMessage))
		if err != nil {
			ar.logger.Error().Err(err).Msg("planning agent failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		if ar.stats != nil {
			if in, out := extractTokenUsage(response); in > 0 || out > 0 {
				ar.stats.RecordTokens(in, out)
			}
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

		response, err := reactAgent.Process(ctx, ar.maybeInjectSkills(agentMessage))
		if err != nil {
			ar.logger.Error().Err(err).Msg("ReActAgent processing failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		if ar.stats != nil {
			if in, out := extractTokenUsage(response); in > 0 || out > 0 {
				ar.stats.RecordTokens(in, out)
			}
		}
		responseText = response.ContentString()
		goto store
	}

conversational:
	{
		// Plain conversational path — each user gets their own isolated agent instance.
		agent := ar.getOrCreateAgent(msg.UserID)
		response, err := agent.Process(ctx, ar.maybeInjectSkills(agentMessage))
		if err != nil {
			ar.logger.Error().Err(err).Msg("agent processing failed")
			return "", fmt.Errorf("agent processing failed: %w", err)
		}
		if ar.stats != nil {
			if in, out := extractTokenUsage(response); in > 0 || out > 0 {
				ar.stats.RecordTokens(in, out)
			}
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

	// Trigger async summarization if enabled.
	if ar.summarizer != nil {
		ar.summarizer.MaybeSummarize(ctx, msg.UserID)
	}

	// Update active users Prometheus gauge.
	observability.ActiveUsers.Set(float64(ar.ActiveUsers()))

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
