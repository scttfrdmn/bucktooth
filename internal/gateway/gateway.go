package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/protocols/mcp"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/channels/testchan"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
	"github.com/scttfrdmn/bucktooth/internal/tools"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// dashboardEvent is the wire struct sent to dashboard WebSocket clients.
type dashboardEvent struct {
	Type      string `json:"type"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Gateway is the main application gateway
type Gateway struct {
	config          *config.Config
	channelRegistry *channels.ChannelRegistry
	eventBus        *EventBus
	agentRouter     *AgentRouter
	memoryStore     memory.Store
	httpServer      *HTTPServer
	wsServer        *WebSocketServer
	stats           *Stats
	userPrefs       *UserPrefs
	mcpClients      []mcp.MCPClient
	logger          zerolog.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// New creates a new Gateway
func New(cfg *config.Config, logger zerolog.Logger) (*Gateway, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create memory store
	var memStore memory.Store
	switch cfg.Memory.Type {
	case "inmemory", "":
		memStore = memory.NewInMemoryStore()
	case "redis":
		opts := cfg.Memory.Options
		addr, _ := opts["addr"].(string)
		password, _ := opts["password"].(string)
		db := 0
		if v, ok := opts["db"].(int); ok {
			db = v
		}
		ttl := 24 * time.Hour
		if v, ok := opts["ttl"].(string); ok {
			if parsed, err := time.ParseDuration(v); err == nil {
				ttl = parsed
			}
		}
		maxHistory := 50
		if v, ok := opts["max_history"].(int); ok {
			maxHistory = v
		}
		redisStore, err := memory.NewRedisStore(addr, password, db, ttl, maxHistory)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create Redis memory store: %w", err)
		}
		memStore = redisStore
	case "vector":
		opts := cfg.Memory.Options
		embedProvider := memory.NewOpenAIEmbeddingProvider(
			optStr(opts, "embedding_base_url"),
			optStr(opts, "embedding_api_key"),
			optStr(opts, "embedding_model"),
		)
		memStore = memory.NewVectorStore(embedProvider)
	default:
		cancel()
		return nil, fmt.Errorf("unsupported memory type: %s", cfg.Memory.Type)
	}

	// Build tool registry from config
	toolRegistry := tools.NewRegistry()
	if cfg.Tools.Calculator.Enabled {
		toolRegistry.Register(tools.NewCalculatorTool())
		logger.Info().Msg("calculator tool registered")
	}
	if cfg.Tools.Message.Enabled {
		toolRegistry.Register(tools.NewMessageFormatterTool())
		logger.Info().Msg("message_formatter tool registered")
	}
	if cfg.Tools.FileSystem.Enabled {
		opts := cfg.Tools.FileSystem.Options
		sandboxDir, _ := opts["sandbox_dir"].(string)
		maxFileSize := int64(0)
		if v, ok := opts["max_file_size"].(int); ok {
			maxFileSize = int64(v)
		}
		fsTool, err := tools.NewFilesystemTool(sandboxDir, maxFileSize)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create filesystem tool: %w", err)
		}
		toolRegistry.Register(fsTool)
		logger.Info().Str("sandbox", sandboxDir).Msg("filesystem tool registered")
	}

	if cfg.Tools.WebSearch.Enabled {
		apiKey, _ := cfg.Tools.WebSearch.Options["api_key"].(string)
		maxResults := 5
		if v, ok := cfg.Tools.WebSearch.Options["max_results"].(int); ok && v > 0 {
			maxResults = v
		}
		toolRegistry.Register(tools.NewWebSearchTool(apiKey, maxResults))
		logger.Info().Msg("web_search tool registered")
	}

	if cfg.Tools.WebFetch.Enabled {
		maxBytes := 0
		if v, ok := cfg.Tools.WebFetch.Options["max_bytes"].(int); ok {
			maxBytes = v
		}
		toolRegistry.Register(tools.NewWebFetchTool(maxBytes))
		logger.Info().Msg("web_fetch tool registered")
	}

	if cfg.Tools.Shell.Enabled {
		requireApproval := true
		if v, ok := cfg.Tools.Shell.Options["require_approval"].(bool); ok {
			requireApproval = v
		}
		var allowedCmds []string
		if v, ok := cfg.Tools.Shell.Options["allowed_commands"].([]any); ok {
			for _, item := range v {
				if s, ok := item.(string); ok {
					allowedCmds = append(allowedCmds, s)
				}
			}
		}
		toolRegistry.Register(tools.NewShellTool(requireApproval, allowedCmds))
		logger.Info().Bool("require_approval", requireApproval).Msg("shell tool registered")
	}

	if cfg.Tools.Calendar.Enabled {
		storePath, _ := cfg.Tools.Calendar.Options["store_path"].(string)
		calTool, err := tools.NewCalendarTool(storePath)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create calendar tool: %w", err)
		}
		toolRegistry.Register(calTool)
		logger.Info().Str("store", storePath).Msg("calendar tool registered")
	}

	// Connect to configured MCP servers and register their tools.
	var mcpClients []mcp.MCPClient
	for _, srv := range cfg.MCP.Servers {
		client, mcpTools, err := connectMCPServer(ctx, srv, logger)
		if err != nil {
			// Non-fatal: log and skip the server so the gateway still starts.
			logger.Error().Err(err).Str("mcp_server", srv.Name).Msg("failed to connect to MCP server")
			continue
		}
		mcpClients = append(mcpClients, client)
		for _, t := range mcpTools {
			toolRegistry.Register(t)
		}
		logger.Info().
			Str("mcp_server", srv.Name).
			Int("tools", len(mcpTools)).
			Msg("MCP server connected")
	}

	// Create event bus
	eventBus := NewEventBus(logger)

	// Create stats and user preferences store
	stats := NewStats()

	// Create agent router
	agentRouter, err := NewAgentRouter(cfg.Agents, memStore, toolRegistry, stats, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create agent router: %w", err)
	}
	userPrefs := NewUserPrefs()

	// Create channel registry
	channelRegistry := channels.NewChannelRegistry()

	// Create HTTP server
	httpServer := NewHTTPServer(cfg.Gateway.HTTPPort, channelRegistry, agentRouter, stats, logger)
	httpServer.SetStaticFiles(webFileServer())
	httpServer.SetDashboardAuth(cfg.Gateway.DashboardAuthPassword)
	httpServer.SetAPIToken(cfg.Gateway.APIToken)
	httpServer.SetVersion(readVersionFile())
	httpServer.SetUserPrefs(userPrefs)

	// Register test channel routes and channel before the gateway struct is created.
	if cfg.Gateway.TestChannel {
		tc := testchan.New(logger)
		channelRegistry.Register(tc)
		httpServer.Handle("/test/send", http.HandlerFunc(tc.HandleSend))
		httpServer.Handle("/test/responses", http.HandlerFunc(tc.HandleResponses))
		logger.Info().Msg("test channel enabled (harness mode)")
	}

	// Create WebSocket server
	wsServer := NewWebSocketServer(cfg.Gateway.WebSocketPort, logger)

	g := &Gateway{
		config:          cfg,
		channelRegistry: channelRegistry,
		eventBus:        eventBus,
		agentRouter:     agentRouter,
		memoryStore:     memStore,
		httpServer:      httpServer,
		wsServer:        wsServer,
		stats:           stats,
		userPrefs:       userPrefs,
		mcpClients:      mcpClients,
		logger:          logger.With().Str("component", "gateway").Logger(),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Subscribe to message events
	eventBus.Subscribe(EventTypeMessageReceived, g.handleMessageReceived)

	// Subscribe to all events for dashboard broadcast
	for _, et := range []EventType{
		EventTypeMessageReceived,
		EventTypeMessageSent,
		EventTypeChannelConnected,
		EventTypeChannelDisconnected,
		EventTypeAgentStarted,
		EventTypeAgentCompleted,
		EventTypeAgentError,
	} {
		eventBus.Subscribe(et, g.broadcastToDashboard)
	}

	return g, nil
}

// Start starts the gateway
func (g *Gateway) Start() error {
	g.logger.Info().Msg("starting gateway")

	// Start HTTP server
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := g.httpServer.Start(g.ctx); err != nil {
			g.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Start WebSocket server
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := g.wsServer.Start(g.ctx); err != nil {
			g.logger.Error().Err(err).Msg("WebSocket server error")
		}
	}()

	// Start all enabled channels
	for name, channelCfg := range g.config.Channels {
		if !channelCfg.Enabled {
			continue
		}

		channel, ok := g.channelRegistry.Get(name)
		if !ok {
			g.logger.Warn().Str("channel", name).Msg("channel not registered")
			continue
		}

		if err := g.startChannel(channel); err != nil {
			g.logger.Error().Err(err).Str("channel", name).Msg("failed to start channel")
			continue
		}
	}

	// Start test channel if harness mode is enabled.
	if g.config.Gateway.TestChannel {
		if tc, ok := g.channelRegistry.Get("test"); ok {
			if err := g.startChannel(tc); err != nil {
				g.logger.Error().Err(err).Str("channel", "test").Msg("failed to start test channel")
			}
		}
	}

	g.logger.Info().Msg("gateway started successfully")
	return nil
}

// Stop stops the gateway gracefully
func (g *Gateway) Stop() error {
	g.logger.Info().Msg("stopping gateway")

	// Cancel context
	g.cancel()

	// Stop all channels
	for _, channel := range g.channelRegistry.All() {
		if err := channel.Disconnect(); err != nil {
			g.logger.Error().Err(err).Str("channel", channel.Name()).Msg("failed to disconnect channel")
		}
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		g.logger.Info().Msg("all goroutines stopped")
	case <-time.After(g.config.Gateway.ShutdownTimeout):
		g.logger.Warn().Msg("shutdown timeout exceeded")
	}

	// Close MCP clients
	for _, client := range g.mcpClients {
		if err := client.Close(); err != nil {
			g.logger.Error().Err(err).Msg("failed to close MCP client")
		}
	}

	// Close resources
	if err := g.agentRouter.Close(); err != nil {
		g.logger.Error().Err(err).Msg("failed to close agent router")
	}

	if err := g.memoryStore.Close(); err != nil {
		g.logger.Error().Err(err).Msg("failed to close memory store")
	}

	g.logger.Info().Msg("gateway stopped")
	return nil
}

// RegisterChannel registers a channel with the gateway
func (g *Gateway) RegisterChannel(channel channels.Channel) {
	g.channelRegistry.Register(channel)
	g.logger.Info().Str("channel", channel.Name()).Msg("channel registered")
}

// Handle registers an extra HTTP route, delegating to the HTTP server.
// Must be called before Start().
func (g *Gateway) Handle(pattern string, handler http.Handler) {
	g.httpServer.Handle(pattern, handler)
}

// startChannel starts a channel and begins processing messages
func (g *Gateway) startChannel(channel channels.Channel) error {
	g.logger.Info().Str("channel", channel.Name()).Msg("starting channel")

	// Connect to the channel
	if err := channel.Connect(g.ctx); err != nil {
		return fmt.Errorf("failed to connect channel: %w", err)
	}

	// Publish connected event
	g.eventBus.Publish(g.ctx, ChannelConnectedEvent(channel.Name()))

	// Start message receiver
	msgChan, err := channel.ReceiveMessages(g.ctx)
	if err != nil {
		return fmt.Errorf("failed to get message channel: %w", err)
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.receiveMessages(channel, msgChan)
	}()

	g.logger.Info().Str("channel", channel.Name()).Msg("channel started")
	return nil
}

// receiveMessages receives messages from a channel
func (g *Gateway) receiveMessages(channel channels.Channel, msgChan <-chan *channels.Message) {
	for {
		select {
		case <-g.ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				g.logger.Info().Str("channel", channel.Name()).Msg("message channel closed")
				return
			}

			// Publish message received event
			g.eventBus.Publish(g.ctx, MessageReceivedEvent(msg))
		}
	}
}

// connectMCPServer connects to a single MCP server and returns the client and its tools.
func connectMCPServer(ctx context.Context, srv config.MCPServerConfig, logger zerolog.Logger) (mcp.MCPClient, []agenkit.Tool, error) {
	var client mcp.MCPClient
	switch srv.Type {
	case "stdio":
		c, err := mcp.NewStdioClient(ctx, mcp.StdioConfig{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("mcp stdio connect: %w", err)
		}
		client = c
	case "http":
		c, err := mcp.NewHTTPClient(ctx, srv.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("mcp http connect: %w", err)
		}
		client = c
	default:
		return nil, nil, fmt.Errorf("unknown MCP server type %q (want \"stdio\" or \"http\")", srv.Type)
	}

	mcpTools, err := mcp.ToolsFromClient(ctx, client)
	if err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("list MCP tools: %w", err)
	}

	info := client.ServerInfo()
	logger.Debug().
		Str("mcp_server", srv.Name).
		Str("server_name", info.Name).
		Str("server_version", info.Version).
		Msg("MCP handshake complete")

	return client, mcpTools, nil
}

// optStr safely extracts a string value from a map[string]any by key.
func optStr(opts map[string]any, key string) string {
	if opts == nil {
		return ""
	}
	v, _ := opts[key].(string)
	return v
}

// readVersionFile reads the VERSION file from the working directory; returns "dev" on failure.
func readVersionFile() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}

// broadcastToDashboard marshals an event and sends it to all dashboard WS clients.
func (g *Gateway) broadcastToDashboard(ctx context.Context, event Event) {
	content := ""
	if event.Message != nil {
		content = event.Message.Content
	}
	if r, ok := event.Data["response"].(string); ok && content == "" {
		content = r
	}

	wire := dashboardEvent{
		Type:      string(event.Type),
		ChannelID: event.ChannelID,
		Content:   content,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(wire)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to marshal dashboard event")
		return
	}

	g.httpServer.BroadcastEvent(payload)
}

// handleMessageReceived processes received messages
func (g *Gateway) handleMessageReceived(ctx context.Context, event Event) {
	msg := event.Message
	if msg == nil {
		return
	}

	tracer := otel.Tracer("bucktooth/gateway")
	ctx, span := tracer.Start(ctx, "gateway.handle_message",
		trace.WithAttributes(
			attribute.String("channel", msg.ChannelID),
			attribute.String("user_id", msg.UserID),
		))
	defer span.End()

	g.logger.Debug().
		Str("channel", msg.ChannelID).
		Str("user", msg.Username).
		Str("content", msg.Content).
		Msg("handling message")

	// Record inbound message statistics
	g.stats.RecordInbound(msg)

	// Publish agent started event
	g.eventBus.Publish(ctx, AgentStartedEvent(msg))

	// Process message with agent router
	response, err := g.agentRouter.ProcessMessage(ctx, msg)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to process message")
		g.eventBus.Publish(ctx, AgentErrorEvent(msg, err))
		return
	}

	// Publish agent completed event
	g.eventBus.Publish(ctx, AgentCompletedEvent(msg, response))

	// Resolve target channel — honour user's preferred channel if set
	targetChannelID := g.userPrefs.Get(msg.UserID)
	if targetChannelID == "" {
		targetChannelID = msg.ChannelID
	}
	channel, ok := g.channelRegistry.Get(targetChannelID)
	if !ok {
		g.logger.Error().Str("channel_id", targetChannelID).Msg("channel not found for routing")
		return
	}

	responseMsg := &channels.Message{
		ChannelID: targetChannelID,
		Content:   response,
		Metadata:  msg.Metadata,
		Timestamp: time.Now(),
	}

	if err := channel.SendMessage(ctx, responseMsg); err != nil {
		g.logger.Error().Err(err).Msg("failed to send response")
		return
	}

	// Record outbound message statistics
	g.stats.RecordOutbound(responseMsg)

	// Publish message sent event
	g.eventBus.Publish(ctx, MessageSentEvent(responseMsg))
}
