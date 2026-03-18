package gateway

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeMessageReceived     EventType = "message.received"
	EventTypeMessageSent         EventType = "message.sent"
	EventTypeChannelConnected    EventType = "channel.connected"
	EventTypeChannelDisconnected EventType = "channel.disconnected"
	EventTypeAgentStarted        EventType = "agent.started"
	EventTypeAgentCompleted      EventType = "agent.completed"
	EventTypeAgentError          EventType = "agent.error"
)

// Event represents an event in the system
type Event struct {
	Type      EventType
	ChannelID string
	Message   *channels.Message
	Data      map[string]any
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event)

// EventBus manages event subscriptions and publishing
type EventBus struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
	logger   zerolog.Logger
}

// NewEventBus creates a new event bus
func NewEventBus(logger zerolog.Logger) *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
		logger:   logger.With().Str("component", "eventbus").Logger(),
	}
}

// Subscribe adds a handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.logger.Debug().Str("event_type", string(eventType)).Msg("subscribed to event")
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(ctx context.Context, event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		eb.logger.Debug().Str("event_type", string(event.Type)).Msg("no handlers for event")
		return
	}

	eb.logger.Debug().
		Str("event_type", string(event.Type)).
		Str("channel_id", event.ChannelID).
		Int("handler_count", len(handlers)).
		Msg("publishing event")

	// Call handlers concurrently
	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h EventHandler) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					eb.logger.Error().
						Interface("panic", r).
						Str("event_type", string(event.Type)).
						Msg("handler panicked")
				}
			}()
			h(ctx, event)
		}(handler)
	}

	wg.Wait()
}

// MessageReceivedEvent creates a message received event
func MessageReceivedEvent(msg *channels.Message) Event {
	return Event{
		Type:      EventTypeMessageReceived,
		ChannelID: msg.ChannelID,
		Message:   msg,
	}
}

// MessageSentEvent creates a message sent event
func MessageSentEvent(msg *channels.Message) Event {
	return Event{
		Type:      EventTypeMessageSent,
		ChannelID: msg.ChannelID,
		Message:   msg,
	}
}

// ChannelConnectedEvent creates a channel connected event
func ChannelConnectedEvent(channelName string) Event {
	return Event{
		Type:      EventTypeChannelConnected,
		ChannelID: channelName,
		Data: map[string]any{
			"channel": channelName,
		},
	}
}

// ChannelDisconnectedEvent creates a channel disconnected event
func ChannelDisconnectedEvent(channelName string) Event {
	return Event{
		Type:      EventTypeChannelDisconnected,
		ChannelID: channelName,
		Data: map[string]any{
			"channel": channelName,
		},
	}
}

// AgentStartedEvent creates an agent started event
func AgentStartedEvent(msg *channels.Message) Event {
	return Event{
		Type:      EventTypeAgentStarted,
		ChannelID: msg.ChannelID,
		Message:   msg,
	}
}

// AgentCompletedEvent creates an agent completed event
func AgentCompletedEvent(msg *channels.Message, response string) Event {
	return Event{
		Type:      EventTypeAgentCompleted,
		ChannelID: msg.ChannelID,
		Message:   msg,
		Data: map[string]any{
			"response": response,
		},
	}
}

// AgentErrorEvent creates an agent error event
func AgentErrorEvent(msg *channels.Message, err error) Event {
	return Event{
		Type:      EventTypeAgentError,
		ChannelID: msg.ChannelID,
		Message:   msg,
		Data: map[string]any{
			"error": err.Error(),
		},
	}
}
