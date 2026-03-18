package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// HTTPServer provides HTTP endpoints
type HTTPServer struct {
	port            int
	server          *http.Server
	channelRegistry *channels.ChannelRegistry
	logger          zerolog.Logger

	// Dashboard WebSocket hub
	dashClients  map[*websocket.Conn]bool
	dashMu       sync.RWMutex
	dashUpgrader websocket.Upgrader
	staticFiles  http.Handler
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port int, registry *channels.ChannelRegistry, logger zerolog.Logger) *HTTPServer {
	return &HTTPServer{
		port:            port,
		channelRegistry: registry,
		logger:          logger.With().Str("component", "http").Logger(),
		dashClients:     make(map[*websocket.Conn]bool),
		dashUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// SetStaticFiles sets the handler used to serve the embedded web dashboard.
func (h *HTTPServer) SetStaticFiles(fs http.Handler) {
	h.staticFiles = fs
}

// BroadcastEvent sends payload to all connected dashboard WebSocket clients.
func (h *HTTPServer) BroadcastEvent(payload []byte) {
	h.dashMu.RLock()
	defer h.dashMu.RUnlock()

	for conn := range h.dashClients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			h.logger.Debug().Err(err).Msg("dashboard ws write error")
		}
	}
}

// handleDashWS upgrades the HTTP connection to a dashboard WebSocket.
func (h *HTTPServer) handleDashWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.dashUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade dashboard ws connection")
		return
	}

	h.dashMu.Lock()
	h.dashClients[conn] = true
	h.dashMu.Unlock()

	h.logger.Debug().Str("remote", r.RemoteAddr).Msg("dashboard client connected")

	defer func() {
		h.dashMu.Lock()
		delete(h.dashClients, conn)
		h.dashMu.Unlock()
		if err := conn.Close(); err != nil {
			h.logger.Debug().Err(err).Msg("dashboard ws close error")
		}
		h.logger.Debug().Str("remote", r.RemoteAddr).Msg("dashboard client disconnected")
	}()

	// Read loop — we only handle close/ping frames; clients don't send data.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Dashboard static files and WebSocket
	if h.staticFiles != nil {
		mux.Handle("/", h.staticFiles)
	}
	mux.HandleFunc("/api/ws", h.handleDashWS)

	// Health check endpoint
	mux.HandleFunc("/health", h.handleHealth)

	// Status endpoint
	mux.HandleFunc("/status", h.handleStatus)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	h.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", h.port),
		Handler:      h.loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	h.logger.Info().Int("port", h.port).Msg("starting HTTP server")

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		h.logger.Info().Msg("shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Close dashboard clients
		h.dashMu.Lock()
		for conn := range h.dashClients {
			if err := conn.Close(); err != nil {
				h.logger.Debug().Err(err).Msg("dashboard ws shutdown close error")
			}
		}
		h.dashMu.Unlock()

		return h.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

// handleHealth handles health check requests
func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := h.channelRegistry.HealthCheck()

	allHealthy := true
	for _, status := range health {
		if !status.Healthy {
			allHealthy = false
			break
		}
	}

	statusCode := http.StatusOK
	if !allHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   getStatusString(allHealthy),
		"channels": health,
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode health response")
	}
}

// handleStatus handles status requests
func (h *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	channels := h.channelRegistry.All()
	channelInfo := make([]map[string]any, 0, len(channels))

	for _, ch := range channels {
		health := ch.Health()
		channelInfo = append(channelInfo, map[string]any{
			"name":    ch.Name(),
			"healthy": health.Healthy,
			"status":  health.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   "running",
		"channels": channelInfo,
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode status response")
	}
}

// loggingMiddleware logs HTTP requests
func (h *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		h.logger.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapper.statusCode).
			Dur("duration", time.Since(start)).
			Msg("HTTP request")
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func getStatusString(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "unhealthy"
}
