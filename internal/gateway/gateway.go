package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
)

// Gateway is the main application gateway
type Gateway struct {
	config          *config.Config
	channelRegistry *channels.ChannelRegistry
	eventBus        *EventBus
	agentRouter     *AgentRouter
	memoryStore     memory.Store
	httpServer      *HTTPServer
	wsServer        *WebSocketServer
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
	case "inmemory":
		memStore = memory.NewInMemoryStore()
	default:
		cancel()
		return nil, fmt.Errorf("unsupported memory type: %s", cfg.Memory.Type)
	}

	// Create event bus
	eventBus := NewEventBus(logger)

	// Create agent router
	agentRouter, err := NewAgentRouter(cfg.Agents, memStore, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create agent router: %w", err)
	}

	// Create channel registry
	channelRegistry := channels.NewChannelRegistry()

	// Create HTTP server
	httpServer := NewHTTPServer(cfg.Gateway.HTTPPort, channelRegistry, logger)

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
		logger:          logger.With().Str("component", "gateway").Logger(),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Subscribe to message events
	eventBus.Subscribe(EventTypeMessageReceived, g.handleMessageReceived)

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

// handleMessageReceived processes received messages
func (g *Gateway) handleMessageReceived(ctx context.Context, event Event) {
	msg := event.Message
	if msg == nil {
		return
	}

	g.logger.Debug().
		Str("channel", msg.ChannelID).
		Str("user", msg.Username).
		Str("content", msg.Content).
		Msg("handling message")

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

	// Send response back to the channel
	channel, ok := g.channelRegistry.Get("discord") // TODO: Get correct channel
	if !ok {
		g.logger.Error().Msg("channel not found")
		return
	}

	responseMsg := &channels.Message{
		ChannelID: msg.ChannelID,
		Content:   response,
		Metadata:  msg.Metadata,
		Timestamp: time.Now(),
	}

	if err := channel.SendMessage(ctx, responseMsg); err != nil {
		g.logger.Error().Err(err).Msg("failed to send response")
		return
	}

	// Publish message sent event
	g.eventBus.Publish(ctx, MessageSentEvent(responseMsg))
}
