package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/voocel/tarot-agent/internal/agents"
	"github.com/voocel/tarot-agent/internal/bootstrap"
	"github.com/voocel/tarot-agent/internal/host/tui"
	"github.com/voocel/tarot-agent/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	modeFlag := flag.String("mode", "", "reading mode: 'pro' or 'casual'")
	flag.Parse()

	var cfg *bootstrap.Config

	// First-run setup
	if bootstrap.NeedsSetup() {
		var err error
		cfg, err = bootstrap.RunSetup()
		if err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	} else {
		var err error
		cfg, err = bootstrap.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	}

	// Command-line mode overrides config
	if *modeFlag != "" {
		switch *modeFlag {
		case "pro":
			cfg.Mode = "professional"
		case "casual":
			cfg.Mode = "casual"
		default:
			return fmt.Errorf("invalid mode %q: use 'pro' or 'casual'", *modeFlag)
		}
	}

	// Initialize logger
	bootstrap.InitLogger(cfg.LogLevel)
	slog.Info("starting tarot-agent",
		"provider", cfg.Provider,
		"model", cfg.Model,
		"base_url", cfg.BaseURL,
	)

	// Initialize store
	s, err := store.New()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	slog.Info("store initialized",
		"cards", s.Cards.Count(),
		"spreads", len(s.Spreads.GetAll()),
	)

	// Initialize LLM model
	model, err := bootstrap.NewModel(cfg)
	if err != nil {
		return fmt.Errorf("init model: %w", err)
	}
	slog.Info("model initialized", "provider", cfg.Provider, "model", cfg.Model)

	// Build agent
	mode := agents.ParseMode(cfg.Mode)
	result := agents.BuildAgent(model, s, mode)
	slog.Info("agent built", "mode", mode.Label())

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		result.Agent.Abort()
	}()

	// Launch TUI
	return tui.Run(result.Agent, result.Guard, s, result.Mode.Label())
}
