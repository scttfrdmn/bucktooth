package config

import (
	"time"
)

// Config represents the entire application configuration
type Config struct {
	Gateway  GatewayConfig            `yaml:"gateway"`
	Channels map[string]ChannelConfig `yaml:"channels"`
	Agents   AgentConfig              `yaml:"agents"`
	Tools    ToolsConfig              `yaml:"tools"`
	Memory   MemoryConfig             `yaml:"memory"`
	Observability ObservabilityConfig  `yaml:"observability"`
}

// GatewayConfig configures the gateway server
type GatewayConfig struct {
	WebSocketPort int           `yaml:"websocket_port"`
	HTTPPort      int           `yaml:"http_port"`
	LogLevel      string        `yaml:"log_level"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// ChannelConfig configures a messaging channel
type ChannelConfig struct {
	Enabled bool                   `yaml:"enabled"`
	Auth    map[string]interface{} `yaml:"auth"`
	Options map[string]interface{} `yaml:"options"`
}

// AgentConfig configures AI agents
type AgentConfig struct {
	LLMProvider string `yaml:"llm_provider"`
	LLMModel    string `yaml:"llm_model"`
	APIKey      string `yaml:"api_key"`
	MaxHistory  int    `yaml:"max_history"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int    `yaml:"max_tokens"`
}

// ToolsConfig configures available tools
type ToolsConfig struct {
	Calendar   ToolConfig `yaml:"calendar"`
	FileSystem ToolConfig `yaml:"filesystem"`
	WebSearch  ToolConfig `yaml:"websearch"`
	Calculator ToolConfig `yaml:"calculator"`
	Message    ToolConfig `yaml:"message"`
}

// ToolConfig configures a specific tool
type ToolConfig struct {
	Enabled bool                   `yaml:"enabled"`
	Options map[string]interface{} `yaml:"options"`
}

// MemoryConfig configures memory storage
type MemoryConfig struct {
	Type    string                 `yaml:"type"` // "inmemory" or "redis"
	Options map[string]interface{} `yaml:"options"`
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
	Enabled      bool    `yaml:"enabled"`
	Endpoint     string  `yaml:"endpoint"`
	SampleRate   float64 `yaml:"sample_rate"`
	ServiceName  string  `yaml:"service_name"`
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
				Options: map[string]interface{}{
					"sandbox_dir": "~/bucktooth-files",
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
				ServiceName: "lobster",
			},
		},
	}
}
