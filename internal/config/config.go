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
	Observability ObservabilityConfig      `yaml:"observability"`
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
	WebSocketPort         int           `yaml:"websocket_port"`
	HTTPPort              int           `yaml:"http_port"`
	LogLevel              string        `yaml:"log_level"`
	ShutdownTimeout       time.Duration `yaml:"shutdown_timeout"`
	TestChannel           bool          `yaml:"test_channel"`
	DashboardAuthPassword string        `yaml:"dashboard_auth_password"`
	APIToken              string        `yaml:"api_token"` // optional; if set, all non-probe routes require Bearer auth
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
}

// ToolConfig configures a specific tool
type ToolConfig struct {
	Enabled bool           `yaml:"enabled"`
	Options map[string]any `yaml:"options"`
}

// MemoryConfig configures memory storage
type MemoryConfig struct {
	Type    string         `yaml:"type"` // "inmemory", "redis", or "vector"
	Options map[string]any `yaml:"options"`
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
			WebSocketPort:   18789,
			HTTPPort:        8080,
			LogLevel:        "info",
			ShutdownTimeout: 30 * time.Second,
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
			Type: "inmemory",
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
