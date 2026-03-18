package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// wsInboundMsg is the JSON structure expected from WebSocket clients.
type wsInboundMsg struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
}

// wsOutboundMsg is the JSON structure sent back to WebSocket clients.
type wsOutboundMsg struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// WebSocketServer provides WebSocket connectivity
type WebSocketServer struct {
	port           int
	server         *http.Server
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	clientMu       sync.RWMutex
	logger         zerolog.Logger
	allowedOrigins map[string]bool // nil = allow all (dev)
	router         *AgentRouter
}

// NewWebSocketServer creates a new WebSocket server.
// allowedOrigins is the set of permitted Origin header values; nil/empty allows all (dev mode).
func NewWebSocketServer(port int, allowedOrigins []string, router *AgentRouter, logger zerolog.Logger) *WebSocketServer {
	ws := &WebSocketServer{
		port:    port,
		clients: make(map[*websocket.Conn]bool),
		logger:  logger.With().Str("component", "websocket").Logger(),
		router:  router,
	}
	if len(allowedOrigins) > 0 {
		ws.allowedOrigins = make(map[string]bool, len(allowedOrigins))
		for _, o := range allowedOrigins {
			ws.allowedOrigins[o] = true
		}
	}
	ws.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if len(ws.allowedOrigins) == 0 {
				return true
			}
			return ws.allowedOrigins[r.Header.Get("Origin")]
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	return ws
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", ws.port),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	ws.logger.Info().Int("port", ws.port).Msg("starting WebSocket server")

	errChan := make(chan error, 1)
	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		ws.logger.Info().Msg("shutting down WebSocket server")

		ws.clientMu.Lock()
		for conn := range ws.clients {
			if err := conn.Close(); err != nil {
				ws.logger.Debug().Err(err).Msg("ws client close error")
			}
		}
		ws.clientMu.Unlock()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return ws.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

// handleWebSocket handles incoming WebSocket connections, routing messages through
// the agent router instead of echoing them back.
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Error().Err(err).Msg("failed to upgrade connection")
		return
	}

	ws.clientMu.Lock()
	ws.clients[conn] = true
	ws.clientMu.Unlock()

	ws.logger.Info().Str("remote_addr", r.RemoteAddr).Msg("client connected")

	defer func() {
		ws.clientMu.Lock()
		delete(ws.clients, conn)
		ws.clientMu.Unlock()
		if err := conn.Close(); err != nil {
			ws.logger.Debug().Err(err).Msg("ws client close error")
		}
		ws.logger.Info().Str("remote_addr", r.RemoteAddr).Msg("client disconnected")
	}()

	ctx := r.Context()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.Error().Err(err).Msg("websocket error")
			}
			break
		}

		ws.logger.Debug().Str("message", string(raw)).Msg("received message")

		var in wsInboundMsg
		if err := json.Unmarshal(raw, &in); err != nil || in.UserID == "" || in.Content == "" {
			out, _ := json.Marshal(wsOutboundMsg{Error: "invalid message: must be JSON with user_id and content"})
			if writeErr := conn.WriteMessage(websocket.TextMessage, out); writeErr != nil {
				break
			}
			continue
		}

		chanMsg := &channels.Message{
			UserID:    in.UserID,
			ChannelID: "websocket",
			Content:   in.Content,
			Timestamp: time.Now(),
		}

		response, err := ws.router.ProcessMessage(ctx, chanMsg)
		if err != nil {
			ws.logger.Error().Err(err).Str("user_id", in.UserID).Msg("router error")
			out, _ := json.Marshal(wsOutboundMsg{UserID: in.UserID, Error: err.Error()})
			if writeErr := conn.WriteMessage(websocket.TextMessage, out); writeErr != nil {
				break
			}
			continue
		}

		out, _ := json.Marshal(wsOutboundMsg{UserID: in.UserID, Content: response})
		if err := conn.WriteMessage(websocket.TextMessage, out); err != nil {
			ws.logger.Error().Err(err).Msg("failed to write message")
			break
		}
	}
}

// Broadcast sends a message to all connected clients
func (ws *WebSocketServer) Broadcast(message []byte) {
	ws.clientMu.RLock()
	defer ws.clientMu.RUnlock()

	for conn := range ws.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			ws.logger.Error().Err(err).Msg("failed to broadcast message")
		}
	}
}
