package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/agenkit/agenkit-go/skills"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	cronsched "github.com/scttfrdmn/bucktooth/internal/cron"
	"github.com/scttfrdmn/bucktooth/internal/memory"
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

	// Stats, agent router, and user prefs for dashboard/data endpoint
	stats       *Stats
	agentRouter *AgentRouter
	userPrefs   *UserPrefs

	// Precomputed "Basic <b64>" string; empty means no auth required
	dashAuthHash string

	// Bearer token for API auth; empty means no auth required
	apiToken string

	// Version string for /dashboard/data
	version string

	// Skill registry for GET /skills (nil when skills are disabled)
	skillRegistry *skills.SkillRegistry

	// Extra routes registered before server start
	extraRoutes map[string]http.Handler

	// Admin API dependencies
	memoryStore memory.Store
	scheduler   *cronsched.Scheduler
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port int, registry *channels.ChannelRegistry, agentRouter *AgentRouter, stats *Stats, logger zerolog.Logger) *HTTPServer {
	return &HTTPServer{
		port:            port,
		channelRegistry: registry,
		agentRouter:     agentRouter,
		stats:           stats,
		logger:          logger.With().Str("component", "http").Logger(),
		dashClients:     make(map[*websocket.Conn]bool),
		dashUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		extraRoutes: make(map[string]http.Handler),
	}
}

// SetAPIToken sets the bearer token required on all non-probe routes.
// An empty string disables token auth (default).
func (h *HTTPServer) SetAPIToken(token string) {
	h.apiToken = token
}

// apiTokenMiddleware enforces Bearer token auth when a token has been configured.
// K8s probe endpoints (/live, /ready) are always exempt.
func (h *HTTPServer) apiTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.apiToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		// K8s probes must be unauthenticated
		if r.URL.Path == "/live" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if !strings.HasPrefix(auth, "Bearer ") ||
			subtle.ConstantTimeCompare([]byte(token), []byte(h.apiToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetDashboardAuth enables Basic auth on dashboard routes using the given password.
// An empty password disables auth (default).
func (h *HTTPServer) SetDashboardAuth(password string) {
	if password == "" {
		h.dashAuthHash = ""
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("bucktooth:" + password))
	h.dashAuthHash = "Basic " + encoded
}

// SetVersion sets the version string returned by /dashboard/data.
func (h *HTTPServer) SetVersion(v string) {
	h.version = v
}

// SetUserPrefs attaches the UserPrefs store for the preferences API.
func (h *HTTPServer) SetUserPrefs(up *UserPrefs) {
	h.userPrefs = up
}

// SetSkillRegistry attaches the skill registry for the GET /skills endpoint.
func (h *HTTPServer) SetSkillRegistry(r *skills.SkillRegistry) {
	h.skillRegistry = r
}

// Handle registers an additional route before the server starts.
func (h *HTTPServer) Handle(pattern string, handler http.Handler) {
	h.extraRoutes[pattern] = handler
}

// SetStaticFiles sets the handler used to serve the embedded web dashboard.
func (h *HTTPServer) SetStaticFiles(fs http.Handler) {
	h.staticFiles = fs
}

// SetMemoryStore attaches the memory store for the admin flush endpoint.
func (h *HTTPServer) SetMemoryStore(store memory.Store) {
	h.memoryStore = store
}

// SetScheduler attaches the cron scheduler for the /cron/jobs endpoint.
func (h *HTTPServer) SetScheduler(s *cronsched.Scheduler) {
	h.scheduler = s
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

// dashAuthMiddleware enforces Basic auth when a password has been configured.
func (h *HTTPServer) dashAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.dashAuthHash != "" {
			if r.Header.Get("Authorization") != h.dashAuthHash {
				w.Header().Set("WWW-Authenticate", `Basic realm="bucktooth"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// handleLive is the liveness probe — always returns 200.
func (h *HTTPServer) handleLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "alive"}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode live response")
	}
}

// handleReady is the readiness probe — returns 503 when no channel is healthy.
func (h *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	health := h.channelRegistry.HealthCheck()
	anyHealthy := len(health) == 0 // no channels = trivially ready (dev mode)
	for _, s := range health {
		if s.Healthy {
			anyHealthy = true
			break
		}
	}
	if !anyHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	statusStr := "not ready"
	if anyHealthy {
		statusStr = "ready"
	}
	if err := json.NewEncoder(w).Encode(map[string]string{"status": statusStr}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode ready response")
	}
}

// handleDashboardData returns a JSON stats snapshot for the dashboard.
func (h *HTTPServer) handleDashboardData(w http.ResponseWriter, r *http.Request) {
	snap := h.stats.Snapshot()

	chans := h.channelRegistry.All()
	channelInfo := make([]map[string]any, 0, len(chans))
	for _, ch := range chans {
		health := ch.Health()
		channelInfo = append(channelInfo, map[string]any{
			"name":    ch.Name(),
			"healthy": health.Healthy,
			"status":  health.Status,
		})
	}

	activeUsers := 0
	if h.agentRouter != nil {
		activeUsers = h.agentRouter.ActiveUsers()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"version":         h.version,
		"uptime_seconds":  snap.UptimeSeconds,
		"messages_in":     snap.MessagesIn,
		"messages_out":    snap.MessagesOut,
		"tokens_in":       snap.TokensIn,
		"tokens_out":      snap.TokensOut,
		"active_users":    activeUsers,
		"channels":        channelInfo,
		"recent_messages": snap.Recent,
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode dashboard data response")
	}
}

// handleSkills returns a JSON list of all loaded agent skills.
func (h *HTTPServer) handleSkills(w http.ResponseWriter, r *http.Request) {
	type skillInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var list []skillInfo
	if h.skillRegistry != nil {
		all := h.skillRegistry.All()
		list = make([]skillInfo, len(all))
		for i, s := range all {
			list[i] = skillInfo{Name: s.Name, Description: s.Description}
		}
	} else {
		list = []skillInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"skills": list, "count": len(list)}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode skills response")
	}
}

// handleUserPreferences handles GET and POST for /users/{user_id}/preferences.
func (h *HTTPServer) handleUserPreferences(w http.ResponseWriter, r *http.Request) {
	// Path: /users/{user_id}/preferences
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[1] != "preferences" {
		http.NotFound(w, r)
		return
	}
	userID := parts[0]
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		channelID := ""
		if h.userPrefs != nil {
			channelID = h.userPrefs.Get(userID)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"preferred_channel_id": channelID}); err != nil {
			h.logger.Error().Err(err).Msg("failed to encode user preferences response")
		}

	case http.MethodPost:
		var body struct {
			PreferredChannelID string `json:"preferred_channel_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if h.userPrefs != nil {
			if body.PreferredChannelID == "" {
				h.userPrefs.Delete(userID)
			} else {
				h.userPrefs.Set(userID, body.PreferredChannelID)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"preferred_channel_id": body.PreferredChannelID}); err != nil {
			h.logger.Error().Err(err).Msg("failed to encode user preferences response")
		}

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAdminMemoryFlush handles GET /admin/memory/{user_id}/flush.
func (h *HTTPServer) handleAdminMemoryFlush(w http.ResponseWriter, r *http.Request) {
	// Path: /admin/memory/{user_id}/flush
	path := strings.TrimPrefix(r.URL.Path, "/admin/memory/")
	userID := strings.TrimSuffix(path, "/flush")
	if userID == "" || userID == path {
		http.NotFound(w, r)
		return
	}
	if h.memoryStore != nil {
		if err := h.memoryStore.ClearHistory(r.Context(), userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user_id": userID}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode admin memory flush response")
	}
}

// handleAdminSkillsReload handles POST /admin/skills/reload.
func (h *HTTPServer) handleAdminSkillsReload(w http.ResponseWriter, r *http.Request) {
	if h.skillRegistry == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.skillRegistry.DiscoverSkills(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	count := len(h.skillRegistry.All())
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"status": "ok", "skills_loaded": count}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode admin skills reload response")
	}
}

// handleAdminChannelsHealth handles GET /admin/channels/health.
func (h *HTTPServer) handleAdminChannelsHealth(w http.ResponseWriter, r *http.Request) {
	health := h.channelRegistry.HealthCheck()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode admin channels health response")
	}
}

// handleCronJobs handles GET /cron/jobs.
func (h *HTTPServer) handleCronJobs(w http.ResponseWriter, r *http.Request) {
	var jobs []cronsched.JobInfo
	if h.scheduler != nil {
		jobs = h.scheduler.Jobs()
	}
	if jobs == nil {
		jobs = []cronsched.JobInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"jobs": jobs}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode cron jobs response")
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Dashboard static files and WebSocket — wrapped with optional auth
	if h.staticFiles != nil {
		mux.Handle("/", h.dashAuthMiddleware(h.staticFiles))
	}
	mux.Handle("/api/ws", h.dashAuthMiddleware(http.HandlerFunc(h.handleDashWS)))

	// Health/liveness/readiness probes (no auth required)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/live", h.handleLive)
	mux.HandleFunc("/ready", h.handleReady)

	// Status endpoint
	mux.HandleFunc("/status", h.handleStatus)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Dashboard stats JSON endpoint — protected by optional auth
	mux.Handle("/dashboard/data", h.dashAuthMiddleware(http.HandlerFunc(h.handleDashboardData)))

	// User preferences API
	mux.HandleFunc("/users/", h.handleUserPreferences)

	// Skills listing
	mux.HandleFunc("/skills", h.handleSkills)

	// Cron jobs listing
	mux.HandleFunc("/cron/jobs", h.handleCronJobs)

	// Admin API (Bearer-token protected via apiTokenMiddleware on outer handler)
	mux.HandleFunc("/admin/memory/", h.handleAdminMemoryFlush)
	mux.HandleFunc("/admin/skills/reload", h.handleAdminSkillsReload)
	mux.HandleFunc("/admin/channels/health", h.handleAdminChannelsHealth)

	// Programmatically registered extra routes (e.g. test channel)
	for pattern, handler := range h.extraRoutes {
		mux.Handle(pattern, handler)
	}

	h.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", h.port),
		Handler:      h.apiTokenMiddleware(h.loggingMiddleware(mux)),
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
