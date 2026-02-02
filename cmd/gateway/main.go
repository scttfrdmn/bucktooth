package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/scttfrdmn/bucktooth/internal/channels/discord"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/gateway"
)

var (
	configPath = flag.String("config", "", "path to configuration file")
	logLevel   = flag.String("log-level", "", "log level (debug, info, warn, error)")
	port       = flag.Int("port", 0, "HTTP port (overrides config)")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override with CLI flags
	if *logLevel != "" {
		cfg.Gateway.LogLevel = *logLevel
	}
	if *port > 0 {
		cfg.Gateway.HTTPPort = *port
	}

	// Setup logging
	logger := setupLogging(cfg.Gateway.LogLevel)
	log.Logger = logger

	logger.Info().Msg("starting Lobster gateway")
	logger.Info().
		Str("llm_provider", cfg.Agents.LLMProvider).
		Str("llm_model", cfg.Agents.LLMModel).
		Int("max_history", cfg.Agents.MaxHistory).
		Msg("agent configuration")

	// Create gateway
	gw, err := gateway.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create gateway")
	}

	// Register enabled channels
	if cfg.Channels["discord"].Enabled {
		discordToken, ok := cfg.Channels["discord"].Auth["token"].(string)
		if !ok || discordToken == "" {
			logger.Fatal().Msg("Discord token is required when Discord channel is enabled")
		}

		discordChannel, err := discord.NewDiscordChannel(discordToken, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create Discord channel")
		}

		gw.RegisterChannel(discordChannel)
		logger.Info().Msg("Discord channel registered")
	}

	// Start gateway
	if err := gw.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start gateway")
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("received shutdown signal")

	// Stop gateway
	if err := gw.Stop(); err != nil {
		logger.Error().Err(err).Msg("error during shutdown")
		os.Exit(1)
	}

	logger.Info().Msg("shutdown complete")
}

func setupLogging(level string) zerolog.Logger {
	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	// Configure logger
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	// Use pretty logging in development
	if logLevel == zerolog.DebugLevel {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	return logger
}
