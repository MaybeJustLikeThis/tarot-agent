package bootstrap

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/voocel/agentcore/llm"
)

// Config holds application configuration.
type Config struct {
	// LLM settings
	APIKey  string
	BaseURL string
	Model   string

	// App settings
	LogLevel string
}

const (
	defaultModel    = "deepseek-chat"
	defaultBaseURL  = "https://api.deepseek.com"
	defaultLogLevel = "info"
)

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model:    defaultModel,
		BaseURL:  defaultBaseURL,
		LogLevel: defaultLogLevel,
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
		if fc.APIKey != "" {
			cfg.APIKey = fc.APIKey
		}
		if fc.BaseURL != "" {
			cfg.BaseURL = fc.BaseURL
		}
		if fc.Model != "" {
			cfg.Model = fc.Model
		}
	}

	// Layer 2: env vars (override file)
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

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required: set DEEPSEEK_API_KEY or TAROT_API_KEY environment variable")
	}

	return cfg, nil
}

// InitLogger sets up structured logging based on config.
func InitLogger(level string) {
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
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
}

// NewModel creates an LLM model from the configuration.
func NewModel(cfg *Config) (*llm.LiteLLMAdapter, error) {
	model, err := llm.NewModel("openai", cfg.Model,
		llm.WithAPIKey(cfg.APIKey),
		llm.WithBaseURL(cfg.BaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("create model %s: %w", cfg.Model, err)
	}
	return model, nil
}
