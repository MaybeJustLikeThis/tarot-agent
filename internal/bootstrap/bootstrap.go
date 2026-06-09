package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/voocel/agentcore/llm"
)

// Config holds application configuration.
type Config struct {
	// LLM settings
	Provider string // "openai", "deepseek", "anthropic"
	APIKey   string
	BaseURL  string
	Model    string

	// App settings
	LogLevel string
	Mode     string // "professional" 或 "casual"
}

const (
	defaultLogLevel = "info"
)

// DefaultConfig returns a Config with sensible defaults (DeepSeek).
func DefaultConfig() *Config {
	pd := knownProviders[defaultProvider]
	return &Config{
		Provider: pd.Name,
		Model:    pd.DefaultModel,
		BaseURL:  pd.DefaultURL,
		LogLevel: defaultLogLevel,
		Mode:     "professional",
	}
}

// LoadConfig loads configuration by merging file + env vars over defaults.
// Priority: defaults < config file < env vars.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Layer 1: config file (~/.tarot-agent/config.json)
	if fc, err := loadFileConfig(); err != nil {
		slog.Warn("failed to load config file, using env vars only", "error", err)
	} else if fc != nil {
		if fc.Provider != "" {
			cfg.Provider = fc.Provider
		}
		if fc.APIKey != "" {
			cfg.APIKey = fc.APIKey
		}
		if fc.BaseURL != "" {
			cfg.BaseURL = fc.BaseURL
		}
		if fc.Model != "" {
			cfg.Model = fc.Model
		}
		if fc.Mode != "" {
			cfg.Mode = fc.Mode
		}
	}

	// Migration: existing config without provider field — infer from BaseURL.
	if cfg.Provider == "" && cfg.BaseURL != "" {
		cfg.Provider = inferProvider(cfg.BaseURL)
		slog.Info("inferred provider from base_url", "provider", cfg.Provider, "base_url", cfg.BaseURL)
	}

	// Layer 2: env vars (override file)
	if v := os.Getenv("TAROT_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("TAROT_API_KEY"); v != "" {
		cfg.APIKey = v
	} else if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.APIKey = v
	}

	if v := os.Getenv("TAROT_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("TAROT_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("TAROT_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("TAROT_MODE"); v != "" {
		cfg.Mode = v
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required: set TAROT_API_KEY environment variable or run setup")
	}

	return cfg, nil
}

// LogFilePath returns the path to the log file (~/.tarot-agent/tarot.log).
func LogFilePath() string {
	dir, err := configDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tarot.log")
}

// InitLogger sets up structured logging based on config.
// When logFile is non-empty, logs go to the file; otherwise to stderr.
func InitLogger(level string, logFile string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	var handler slog.Handler
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			// Fallback to stderr if file can't be opened
			handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
		} else {
			handler = slog.NewTextHandler(f, &slog.HandlerOptions{Level: l})
		}
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	}
	slog.SetDefault(slog.New(handler))
}

// NewModel creates an LLM model from the configuration.
func NewModel(cfg *Config) (*llm.LiteLLMAdapter, error) {
	model, err := llm.NewModel(cfg.Provider, cfg.Model,
		llm.WithAPIKey(cfg.APIKey),
		llm.WithBaseURL(cfg.BaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("create model %s (%s): %w", cfg.Model, cfg.Provider, err)
	}
	return model, nil
}
