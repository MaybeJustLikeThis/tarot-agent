package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/agents"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/bootstrap"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/daily"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/host/tui"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/voocel/agentcore"
)

// version is the release version. GoReleaser overrides it with
// -X main.version={{.Version}}. Local builds stay "dev".
var version = "dev"

var (
	errFlagHelp  = errors.New("help")
	errFlagParse = errors.New("flag")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errFlagHelp) {
			os.Exit(0)
		}
		if errors.Is(err, errFlagParse) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	modeFlag, showVersion, dailyCmd, err := parseCLI(args)
	if err != nil {
		return err
	}
	if showVersion {
		fmt.Printf("tarot-agent %s\n", version)
		return nil
	}
	if dailyCmd {
		return runDaily(modeFlag)
	}
	return runTUI(modeFlag)
}

// parseCLI accepts --mode and --version before or after the daily command.
// Any other positional argument is left alone so a plain launch stays the same.
func parseCLI(args []string) (mode string, showVersion, dailyCmd bool, err error) {
	flagArgs, dailyCmd := extractDaily(args)
	fs := flag.NewFlagSet("tarot-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	modeFlag := fs.String("mode", "", "reading mode: pro, professional, deep, casual, or simple")
	versionFlag := fs.Bool("version", false, "print version and exit")
	if perr := fs.Parse(flagArgs); perr != nil {
		if perr == flag.ErrHelp {
			return "", false, false, errFlagHelp
		}
		return "", false, false, errFlagParse
	}
	return *modeFlag, *versionFlag, dailyCmd, nil
}

func extractDaily(args []string) ([]string, bool) {
	var flagArgs []string
	dailyCmd := false
	for _, arg := range args {
		if arg == "daily" && !dailyCmd {
			dailyCmd = true
			continue
		}
		flagArgs = append(flagArgs, arg)
	}
	return flagArgs, dailyCmd
}

func runTUI(modeFlag string) error {
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

	mode, modeWarn := agents.ResolveMode(cfg.Mode, modeFlag)
	if modeWarn != "" {
		fmt.Fprintf(os.Stderr, "%s\n", modeWarn)
	}
	cfg.Mode = string(mode)

	// Initialize logger — write to file so it doesn't interfere with TUI
	logPath := bootstrap.LogFilePath()
	bootstrap.InitLogger(cfg.LogLevel, logPath)
	if modeWarn != "" {
		slog.Warn("invalid reading mode, using professional", "detail", modeWarn)
	}
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
	return tui.Run(result.Agent, result.Guard, s, result.Mode.Label(), cfg.Model)
}

func runDaily(modeFlag string) error {
	cfg, err := loadDailyConfig()
	if err != nil {
		return err
	}
	mode, modeWarn := agents.ResolveMode(cfg.Mode, modeFlag)
	if modeWarn != "" {
		fmt.Fprintf(os.Stderr, "%s\n", modeWarn)
	}

	s, err := store.New()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	var guide daily.Guide
	modelName := ""
	if cfg.APIKey != "" {
		modelName = cfg.Model
		model, merr := bootstrap.NewModelTimeout(cfg, daily.GuideTimeout)
		if merr != nil {
			guide = func(context.Context, string) (string, error) { return "", merr }
		} else {
			guide = func(ctx context.Context, prompt string) (string, error) {
				resp, gerr := model.Generate(ctx, []agentcore.Message{agentcore.UserMsg(prompt)}, nil)
				if gerr != nil {
					return "", gerr
				}
				if resp == nil {
					return "", fmt.Errorf("empty response")
				}
				return resp.Message.TextContent(), nil
			}
		}
	}
	return daily.Run(context.Background(), daily.Options{
		Store: s,
		Mode:  mode,
		Model: modelName,
		Out:   os.Stdout,
		Guide: guide,
	})
}

func loadDailyConfig() (*bootstrap.Config, error) {
	if bootstrap.NeedsSetup() {
		cfg := bootstrap.DefaultConfig()
		if v := os.Getenv("TAROT_MODE"); v != "" {
			cfg.Mode = v
		}
		cfg.APIKey = ""
		return cfg, nil
	}
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
