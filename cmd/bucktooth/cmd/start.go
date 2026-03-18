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
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/scttfrdmn/bucktooth/internal/channels/discord"
	signalchan "github.com/scttfrdmn/bucktooth/internal/channels/signal"
	slackchan "github.com/scttfrdmn/bucktooth/internal/channels/slack"
	teamschan "github.com/scttfrdmn/bucktooth/internal/channels/teams"
	telegramchan "github.com/scttfrdmn/bucktooth/internal/channels/telegram"
	"github.com/scttfrdmn/bucktooth/internal/channels/whatsapp"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/gateway"
	"github.com/scttfrdmn/bucktooth/internal/observability"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the BuckTooth gateway",
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with CLI flags
	if logLevel != "" {
		cfg.Gateway.LogLevel = logLevel
	}
	if port > 0 {
		cfg.Gateway.HTTPPort = port
	}

	// Setup logging
	logger := setupLogging(cfg.Gateway.LogLevel)
	log.Logger = logger

	logger.Info().Msg("starting BuckTooth gateway")
	logger.Info().
		Str("llm_provider", cfg.Agents.LLMProvider).
		Str("llm_model", cfg.Agents.LLMModel).
		Int("max_history", cfg.Agents.MaxHistory).
		Msg("agent configuration")

	if cfg.Gateway.TestChannel {
		logger.Info().Msg("harness mode: test_channel enabled, stub LLM active")
	}

	// Initialise OpenTelemetry tracing
	tracerShutdown, err := observability.InitTracer(cfg.Observability.Tracing)
	if err != nil {
		return fmt.Errorf("failed to initialise tracer: %w", err)
	}
	defer func() {
		if err := tracerShutdown(context.Background()); err != nil {
			logger.Error().Err(err).Msg("tracer shutdown error")
		}
	}()

	// Create gateway
	gw, err := gateway.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Register enabled channels
	if cfg.Channels["discord"].Enabled {
		discordToken, ok := cfg.Channels["discord"].Auth["token"].(string)
		if !ok || discordToken == "" {
			return fmt.Errorf("discord token is required when discord channel is enabled")
		}

		discordChannel, err := discord.NewDiscordChannel(discordToken, logger)
		if err != nil {
			return fmt.Errorf("failed to create Discord channel: %w", err)
		}

		gw.RegisterChannel(discordChannel)
		logger.Info().Msg("Discord channel registered")
	}

	if cfg.Channels["whatsapp"].Enabled {
		whatsappChannel, err := whatsapp.NewWhatsAppChannel(cfg.Channels["whatsapp"], logger)
		if err != nil {
			return fmt.Errorf("failed to create WhatsApp channel: %w", err)
		}

		gw.RegisterChannel(whatsappChannel)
		logger.Info().Msg("WhatsApp channel registered")
	}

	if cfg.Channels["telegram"].Enabled {
		telegramChannel, err := telegramchan.NewTelegramChannel(cfg.Channels["telegram"], logger)
		if err != nil {
			return fmt.Errorf("failed to create Telegram channel: %w", err)
		}

		gw.RegisterChannel(telegramChannel)
		logger.Info().Msg("Telegram channel registered")
	}

	if cfg.Channels["slack"].Enabled {
		slackChannel, err := slackchan.NewSlackChannel(cfg.Channels["slack"], logger)
		if err != nil {
			return fmt.Errorf("failed to create Slack channel: %w", err)
		}

		gw.RegisterChannel(slackChannel)
		logger.Info().Msg("Slack channel registered")
	}

	if cfg.Channels["teams"].Enabled {
		teamsChannel, err := teamschan.NewTeamsChannel(cfg.Channels["teams"], logger)
		if err != nil {
			return fmt.Errorf("failed to create Teams channel: %w", err)
		}

		gw.RegisterChannel(teamsChannel)
		gw.Handle("/channels/teams/messages", http.HandlerFunc(teamsChannel.HandleMessage))
		logger.Info().Msg("Teams channel registered")
	}

	if cfg.Channels["signal"].Enabled {
		opts := cfg.Channels["signal"].Auth
		phone, _ := opts["phone_number"].(string)
		signaldURL, _ := opts["signald_url"].(string)
		if phone == "" || signaldURL == "" {
			logger.Warn().Msg("signal channel enabled but phone_number or signald_url not set — skipping")
		} else {
			signalChannel := signalchan.New(phone, signaldURL, logger)
			gw.RegisterChannel(signalChannel)
			logger.Info().Str("phone", phone).Msg("Signal channel registered")
		}
	}

	// Start gateway
	if err := gw.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("received shutdown signal")

	// Stop gateway
	if err := gw.Stop(); err != nil {
		logger.Error().Err(err).Msg("error during shutdown")
		return err
	}

	logger.Info().Msg("shutdown complete")
	return nil
}

func setupLogging(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(lvl)

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	if lvl == zerolog.DebugLevel {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	return logger
}
