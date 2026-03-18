package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// WebSocketServer provides WebSocket connectivity
type WebSocketServer struct {
	port     int
	server   *http.Server
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	clientMu sync.RWMutex
	logger   zerolog.Logger
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(port int, logger zerolog.Logger) *WebSocketServer {
	return &WebSocketServer{
		port: port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(map[*websocket.Conn]bool),
		logger:  logger.With().Str("component", "websocket").Logger(),
	}
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

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		ws.logger.Info().Msg("shutting down WebSocket server")

		// Close all client connections
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

// handleWebSocket handles WebSocket connections
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

	// Read messages from client
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.Error().Err(err).Msg("websocket error")
			}
			break
		}

		ws.logger.Debug().
			Int("type", messageType).
			Str("message", string(message)).
			Msg("received message")

		// Echo back for now (TODO: implement proper message handling)
		if err := conn.WriteMessage(messageType, message); err != nil {
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
