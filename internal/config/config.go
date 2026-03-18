package config

import (
	"time"
)

// Config represents the entire application configuration
type Config struct {
	Gateway       GatewayConfig            `yaml:"gateway"`
	Channels      map[string]ChannelConfig `yaml:"channels"`
	Agents        AgentConfig              `yaml:"agents"`
	Tools         ToolsConfig              `yaml:"tools"`
	MCP           MCPConfig                `yaml:"mcp"`
	Memory        MemoryConfig             `yaml:"memory"`
	Skills        SkillsConfig             `yaml:"skills"`
	Observability ObservabilityConfig      `yaml:"observability"`
	RateLimit     RateLimitConfig          `yaml:"rate_limit"`
	Cron          CronConfig               `yaml:"cron"`
}

// MCPConfig configures MCP (Model Context Protocol) server connections.
type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"`
}

// MCPServerConfig configures a single MCP server connection.
type MCPServerConfig struct {
	// Name is a friendly label used in log messages.
	Name string `yaml:"name"`
	// Type is "stdio" (spawn a subprocess) or "http" (connect over HTTP JSON-RPC).
	Type string `yaml:"type"`
	// Command is the executable to spawn (stdio only).
	Command string `yaml:"command"`
	// Args are the arguments passed to Command (stdio only).
	Args []string `yaml:"args"`
	// Env contains additional KEY=VALUE environment variables for the subprocess (stdio only).
	// If empty the subprocess inherits the parent environment.
	Env []string `yaml:"env"`
	// URL is the base URL of the MCP HTTP server (http only).
	URL string `yaml:"url"`
}

// GatewayConfig configures the gateway server
type GatewayConfig struct {
	WebSocketPort         int            `yaml:"websocket_port"`
	HTTPPort              int            `yaml:"http_port"`
	LogLevel              string         `yaml:"log_level"`
	ShutdownTimeout       time.Duration  `yaml:"shutdown_timeout"`
	TestChannel           bool           `yaml:"test_channel"`
	DashboardAuthPassword string         `yaml:"dashboard_auth_password"`
	APIToken              string         `yaml:"api_token"`          // optional; if set, all non-probe routes require Bearer auth
	AllowedWSOrigins      []string       `yaml:"allowed_ws_origins"`  // nil = allow all (dev)
	StreamingEnabled      bool           `yaml:"streaming_enabled"`   // enables WS token streaming; default false
	ChunkingEnabled       bool           `yaml:"chunking_enabled"`    // default true
	ChunkingLimits        map[string]int `yaml:"chunking_limits"`    // per-channel-type char limits (overrides defaults)
	DedupEnabled          bool           `yaml:"dedup_enabled"`      // default true
	DedupWindowSize       int            `yaml:"dedup_window_size"`  // ring buffer size, default 256
	AutoFormatEnabled     bool           `yaml:"auto_format_enabled"`     // default true
	AutoProcessAttachments bool          `yaml:"auto_process_attachments"` // default true
}

// RateLimitConfig configures per-user token-bucket rate limiting.
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"` // default 60
	Burst             int  `yaml:"burst"`               // default 10
}

// CronConfig holds the list of scheduled jobs.
type CronConfig struct {
	Jobs []CronJobConfig `yaml:"jobs"`
}

// CronJobConfig configures a single scheduled job.
type CronJobConfig struct {
	Name      string `yaml:"name"`
	Schedule  string `yaml:"schedule"`   // time.Duration string, e.g. "5m", "1h"
	Message   string `yaml:"message"`
	ChannelID string `yaml:"channel_id"`
	UserID    string `yaml:"user_id"`
	Enabled   bool   `yaml:"enabled"`
}

// ChannelConfig configures a messaging channel
type ChannelConfig struct {
	Enabled bool           `yaml:"enabled"`
	Auth    map[string]any `yaml:"auth"`
	Options map[string]any `yaml:"options"`
}

// AgentConfig configures AI agents
type AgentConfig struct {
	LLMProvider  string  `yaml:"llm_provider"`
	LLMModel     string  `yaml:"llm_model"`
	APIKey       string  `yaml:"api_key"`
	APIBase      string  `yaml:"api_base"` // optional: override Anthropic API endpoint (e.g. for proxies)
	MaxHistory   int     `yaml:"max_history"`
	Temperature  float64 `yaml:"temperature"`
	MaxTokens    int     `yaml:"max_tokens"`
	StubResponse string  `yaml:"stub_response"` // empty = echo mode
	// Mode selects the agent pattern: "conversational", "react" (default), or "planning".
	Mode string `yaml:"mode"`
	// Retry settings for transient LLM errors.
	RetryAttempts       int    `yaml:"retry_attempts"`        // default 3; 0 = disabled
	RetryInitialBackoff string `yaml:"retry_initial_backoff"` // default "500ms"
	// FallbackProviders is tried in order when the primary provider fails with a non-retryable error.
	FallbackProviders []AgentConfig `yaml:"fallback_providers"`
}

// ToolsConfig configures available tools
type ToolsConfig struct {
	Calendar   ToolConfig `yaml:"calendar"`
	FileSystem ToolConfig `yaml:"filesystem"`
	WebSearch  ToolConfig `yaml:"websearch"`
	WebFetch   ToolConfig `yaml:"webfetch"`
	Calculator ToolConfig `yaml:"calculator"`
	Message    ToolConfig `yaml:"message"`
	Shell      ToolConfig `yaml:"shell"`
	PDF        ToolConfig `yaml:"pdf"`
	Image      ToolConfig `yaml:"image"`
}

// ToolConfig configures a specific tool
type ToolConfig struct {
	Enabled bool           `yaml:"enabled"`
	Options map[string]any `yaml:"options"`
}

// MemoryConfig configures memory storage
type MemoryConfig struct {
	Type                    string         `yaml:"type"` // "inmemory", "redis", "vector", "sqlite", or "hybrid"
	Options                 map[string]any `yaml:"options"`
	SummarizeEnabled        bool           `yaml:"summarize_enabled"`
	SummarizeThreshold      int            `yaml:"summarize_threshold"`       // default 30
	SummarizeTokenThreshold int            `yaml:"summarize_token_threshold"` // trigger on est. token count; 0=disabled
	HybridWeight            float64        `yaml:"hybrid_weight"`             // 0.0=pure semantic, 1.0=pure BM25; default 0.5
	DecayEnabled            bool           `yaml:"decay_enabled"`             // exponential time-based recency decay; default false
	DecayHalfLifeHours      float64        `yaml:"decay_half_life_hours"`     // half-life in hours; default 24.0
}

// SkillsConfig configures the agent skills system.
type SkillsConfig struct {
	Enabled         bool     `yaml:"enabled"`
	SearchPaths     []string `yaml:"search_paths"`
	MaxActiveSkills int      `yaml:"max_active_skills"`
}

// ObservabilityConfig configures metrics and tracing
type ObservabilityConfig struct {
	Metrics MetricsConfig `yaml:"metrics"`
	Tracing TracingConfig `yaml:"tracing"`
}

// MetricsConfig configures Prometheus metrics
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// TracingConfig configures OpenTelemetry tracing
type TracingConfig struct {
	Enabled     bool    `yaml:"enabled"`
	Endpoint    string  `yaml:"endpoint"`
	SampleRate  float64 `yaml:"sample_rate"`
	ServiceName string  `yaml:"service_name"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Gateway: GatewayConfig{
			WebSocketPort:          18789,
			HTTPPort:               8080,
			LogLevel:               "info",
			ShutdownTimeout:        30 * time.Second,
			ChunkingEnabled:        true,
			DedupEnabled:           true,
			DedupWindowSize:        256,
			AutoFormatEnabled:      true,
			AutoProcessAttachments: true,
		},
		Channels: map[string]ChannelConfig{},
		Agents: AgentConfig{
			LLMProvider: "anthropic",
			LLMModel:    "claude-sonnet-4-5-20250220",
			MaxHistory:  20,
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		Tools: ToolsConfig{
			Calculator: ToolConfig{Enabled: true},
			Message:    ToolConfig{Enabled: true},
			FileSystem: ToolConfig{
				Enabled: true,
				Options: map[string]any{
					"sandbox_dir":   "~/bucktooth-files",
					"max_file_size": 10485760, // 10MB
				},
			},
		},
		Memory: MemoryConfig{
			Type:               "inmemory",
			SummarizeThreshold: 30,
			HybridWeight:       0.5,
			DecayHalfLifeHours: 24.0,
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerMinute: 60,
			Burst:             10,
		},
		Skills: SkillsConfig{
			Enabled:         false,
			SearchPaths:     []string{"~/.bucktooth/skills", "./skills"},
			MaxActiveSkills: 3,
		},
		Observability: ObservabilityConfig{
			Metrics: MetricsConfig{
				Enabled: true,
				Port:    8080,
				Path:    "/metrics",
			},
			Tracing: TracingConfig{
				Enabled:     false,
				SampleRate:  0.1,
				ServiceName: "bucktooth",
			},
		},
	}
}
