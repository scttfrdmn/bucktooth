package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from a file and applies environment variable overrides
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate configuration
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(cfg *Config) {
	// Gateway overrides
	if port := os.Getenv("LOBSTER_GATEWAY_PORT"); port != "" {
		if v, err := strconv.Atoi(port); err == nil {
			cfg.Gateway.HTTPPort = v
		}
	}
	if wsPort := os.Getenv("LOBSTER_WEBSOCKET_PORT"); wsPort != "" {
		if v, err := strconv.Atoi(wsPort); err == nil {
			cfg.Gateway.WebSocketPort = v
		}
	}
	if logLevel := os.Getenv("LOBSTER_LOG_LEVEL"); logLevel != "" {
		cfg.Gateway.LogLevel = logLevel
	}

	// Agent overrides
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.Agents.APIKey = apiKey
	}
	if model := os.Getenv("LOBSTER_LLM_MODEL"); model != "" {
		cfg.Agents.LLMModel = model
	}
	if v := os.Getenv("ANTHROPIC_API_BASE"); v != "" {
		cfg.Agents.APIBase = v
	}
	if v := os.Getenv("DASHBOARD_AUTH_PASSWORD"); v != "" {
		cfg.Gateway.DashboardAuthPassword = v
	}
	if v := os.Getenv("BUCKTOOTH_API_TOKEN"); v != "" {
		cfg.Gateway.APIToken = v
	}

	// Skills overrides
	if v := os.Getenv("BUCKTOOTH_SKILLS_PATH"); v != "" {
		cfg.Skills.SearchPaths = strings.Split(v, ":")
		cfg.Skills.Enabled = true
	}

	// Channel-specific overrides
	if token := os.Getenv("DISCORD_BOT_TOKEN"); token != "" {
		if cfg.Channels["discord"].Auth == nil {
			discordCfg := cfg.Channels["discord"]
			discordCfg.Auth = make(map[string]any)
			cfg.Channels["discord"] = discordCfg
		}
		cfg.Channels["discord"].Auth["token"] = token
	}
}

// validate checks if the configuration is valid
func validate(cfg *Config) error {
	if cfg.Gateway.HTTPPort <= 0 || cfg.Gateway.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", cfg.Gateway.HTTPPort)
	}
	if cfg.Gateway.WebSocketPort <= 0 || cfg.Gateway.WebSocketPort > 65535 {
		return fmt.Errorf("invalid WebSocket port: %d", cfg.Gateway.WebSocketPort)
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(cfg.Gateway.LogLevel)] {
		return fmt.Errorf("invalid log level: %s", cfg.Gateway.LogLevel)
	}

	if cfg.Agents.LLMProvider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	if cfg.Agents.LLMModel == "" {
		return fmt.Errorf("LLM model is required")
	}

	if cfg.Agents.MaxHistory <= 0 {
		return fmt.Errorf("max_history must be positive")
	}

	return nil
}
