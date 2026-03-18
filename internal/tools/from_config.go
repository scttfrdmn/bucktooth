package tools

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// FromConfig builds a tool Registry from the provided ToolsConfig, registering
// each enabled tool. agentCfg is used by tools that call the Anthropic API
// directly (pdf_analyze, image_analyze). Returns an error only if a required
// tool fails to initialise.
func FromConfig(cfg config.ToolsConfig, agentCfg config.AgentConfig, logger zerolog.Logger) (*Registry, error) {
	registry := NewRegistry()

	if cfg.Calculator.Enabled {
		registry.Register(NewCalculatorTool())
		logger.Info().Msg("calculator tool registered")
	}

	if cfg.Message.Enabled {
		registry.Register(NewMessageFormatterTool())
		logger.Info().Msg("message_formatter tool registered")
	}

	if cfg.FileSystem.Enabled {
		opts := cfg.FileSystem.Options
		sandboxDir, _ := opts["sandbox_dir"].(string)
		maxFileSize := int64(0)
		if v, ok := opts["max_file_size"].(int); ok {
			maxFileSize = int64(v)
		}
		fsTool, err := NewFilesystemTool(sandboxDir, maxFileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to create filesystem tool: %w", err)
		}
		registry.Register(fsTool)
		logger.Info().Str("sandbox", sandboxDir).Msg("filesystem tool registered")
	}

	if cfg.WebSearch.Enabled {
		apiKey, _ := cfg.WebSearch.Options["api_key"].(string)
		maxResults := 5
		if v, ok := cfg.WebSearch.Options["max_results"].(int); ok && v > 0 {
			maxResults = v
		}
		registry.Register(NewWebSearchTool(apiKey, maxResults))
		logger.Info().Msg("web_search tool registered")
	}

	if cfg.WebFetch.Enabled {
		maxBytes := 0
		if v, ok := cfg.WebFetch.Options["max_bytes"].(int); ok {
			maxBytes = v
		}
		registry.Register(NewWebFetchTool(maxBytes))
		logger.Info().Msg("web_fetch tool registered")
	}

	if cfg.Shell.Enabled {
		requireApproval := true
		if v, ok := cfg.Shell.Options["require_approval"].(bool); ok {
			requireApproval = v
		}
		var allowedCmds []string
		if v, ok := cfg.Shell.Options["allowed_commands"].([]any); ok {
			for _, item := range v {
				if s, ok := item.(string); ok {
					allowedCmds = append(allowedCmds, s)
				}
			}
		}
		registry.Register(NewShellTool(requireApproval, allowedCmds))
		logger.Info().Bool("require_approval", requireApproval).Msg("shell tool registered")
	}

	if cfg.Calendar.Enabled {
		storePath, _ := cfg.Calendar.Options["store_path"].(string)
		calTool, err := NewCalendarTool(storePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create calendar tool: %w", err)
		}
		registry.Register(calTool)
		logger.Info().Str("store", storePath).Msg("calendar tool registered")
	}

	if cfg.PDF.Enabled {
		model, _ := cfg.PDF.Options["model"].(string)
		if model == "" {
			model = agentCfg.LLMModel
		}
		sandboxDir, _ := cfg.PDF.Options["sandbox_dir"].(string)
		apiBase := agentCfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		registry.Register(NewPDFAnalysisTool(agentCfg.APIKey, apiBase, model, sandboxDir))
		logger.Info().Msg("pdf_analyze tool registered")
	}

	if cfg.Image.Enabled {
		model, _ := cfg.Image.Options["model"].(string)
		if model == "" {
			model = agentCfg.LLMModel
		}
		sandboxDir, _ := cfg.Image.Options["sandbox_dir"].(string)
		maxBytes := int64(0)
		if v, ok := cfg.Image.Options["max_bytes"].(int); ok {
			maxBytes = int64(v)
		}
		apiBase := agentCfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		registry.Register(NewImageAnalysisTool(agentCfg.APIKey, apiBase, model, maxBytes, sandboxDir))
		logger.Info().Msg("image_analyze tool registered")
	}

	return registry, nil
}
