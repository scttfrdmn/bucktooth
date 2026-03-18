// Copyright 2026 Scott Friedman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/scttfrdmn/agenkit/agenkit-go/protocols/mcp"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/tools"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP commands",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Expose registered tools as an MCP server over stdio",
	RunE:  runMCPServe,
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger := setupLogging(cfg.Gateway.LogLevel)

	toolRegistry, err := tools.FromConfig(cfg.Tools, cfg.Agents, logger)
	if err != nil {
		return fmt.Errorf("failed to create tool registry: %w", err)
	}

	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "bucktooth",
		Version: mcpVersionFile(),
		Tools:   toolRegistry.GetAll(),
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info().Int("tools", len(toolRegistry.GetAll())).Msg("MCP server starting on stdio")
	return server.ServeStdio(ctx)
}

// mcpVersionFile reads the VERSION file; returns "dev" on error.
func mcpVersionFile() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}
