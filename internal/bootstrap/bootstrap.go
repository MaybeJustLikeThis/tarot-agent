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
	cfg.APIKey = resolveAPIKey(cfg.APIKey, cfg.Provider, os.Getenv("TAROT_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))

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

// resolveAPIKey chooses the API key after the config file and TAROT_PROVIDER are applied.
// TAROT_API_KEY always wins. DEEPSEEK_API_KEY starts a session when no key is saved,
// and replaces a saved key only when the effective provider is DeepSeek.
// It does not overwrite another provider's saved key.
func resolveAPIKey(savedKey, provider, tarotKey, deepseekKey string) string {
	if tarotKey != "" {
		return tarotKey
	}
	if deepseekKey != "" && (savedKey == "" || provider == "" || provider == "deepseek") {
		return deepseekKey
	}
	return savedKey
}

// LogFilePath returns the path to the log file (~/.tarot-agent/tarot.log).
func LogFilePath() string {
	dir, err := configDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tarot.log")
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
