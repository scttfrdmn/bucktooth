package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test gateway defaults
	if cfg.Gateway.HTTPPort != 8080 {
		t.Errorf("Expected HTTPPort 8080, got %d", cfg.Gateway.HTTPPort)
	}

	if cfg.Gateway.WebSocketPort != 18789 {
		t.Errorf("Expected WebSocketPort 18789, got %d", cfg.Gateway.WebSocketPort)
	}

	if cfg.Gateway.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got %s", cfg.Gateway.LogLevel)
	}

	if cfg.Gateway.ShutdownTimeout != 30*time.Second {
		t.Errorf("Expected ShutdownTimeout 30s, got %v", cfg.Gateway.ShutdownTimeout)
	}

	// Test agent defaults
	if cfg.Agents.LLMProvider != "anthropic" {
		t.Errorf("Expected LLMProvider 'anthropic', got %s", cfg.Agents.LLMProvider)
	}

	if cfg.Agents.MaxHistory != 20 {
		t.Errorf("Expected MaxHistory 20, got %d", cfg.Agents.MaxHistory)
	}

	// Test memory defaults
	if cfg.Memory.Type != "inmemory" {
		t.Errorf("Expected Memory type 'inmemory', got %s", cfg.Memory.Type)
	}

	// Test observability defaults
	if !cfg.Observability.Metrics.Enabled {
		t.Error("Expected Metrics to be enabled by default")
	}

	if cfg.Observability.Metrics.Path != "/metrics" {
		t.Errorf("Expected Metrics path '/metrics', got %s", cfg.Observability.Metrics.Path)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid HTTP port",
			cfg: &Config{
				Gateway: GatewayConfig{
					HTTPPort:      -1,
					WebSocketPort: 18789,
					LogLevel:      "info",
				},
				Agents: AgentConfig{
					LLMProvider: "anthropic",
					LLMModel:    "claude-sonnet-4-5-20250220",
					MaxHistory:  20,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Gateway: GatewayConfig{
					HTTPPort:      8080,
					WebSocketPort: 18789,
					LogLevel:      "invalid",
				},
				Agents: AgentConfig{
					LLMProvider: "anthropic",
					LLMModel:    "claude-sonnet-4-5-20250220",
					MaxHistory:  20,
				},
			},
			wantErr: true,
		},
		{
			name: "missing LLM provider",
			cfg: &Config{
				Gateway: GatewayConfig{
					HTTPPort:      8080,
					WebSocketPort: 18789,
					LogLevel:      "info",
				},
				Agents: AgentConfig{
					LLMProvider: "",
					LLMModel:    "claude-sonnet-4-5-20250220",
					MaxHistory:  20,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
