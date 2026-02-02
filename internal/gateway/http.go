package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port int, registry *channels.ChannelRegistry, logger zerolog.Logger) *HTTPServer {
	return &HTTPServer{
		port:            port,
		channelRegistry: registry,
		logger:          logger.With().Str("component", "http").Logger(),
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   getStatusString(allHealthy),
		"channels": health,
	})
}

// handleStatus handles status requests
func (h *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	channels := h.channelRegistry.All()
	channelInfo := make([]map[string]interface{}, 0, len(channels))

	for _, ch := range channels {
		health := ch.Health()
		channelInfo = append(channelInfo, map[string]interface{}{
			"name":    ch.Name(),
			"healthy": health.Healthy,
			"status":  health.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "running",
		"channels": channelInfo,
	})
}

// loggingMiddleware logs HTTP requests
func (h *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
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
