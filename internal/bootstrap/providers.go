package bootstrap

import "regexp"

// ProviderDefaults defines the default configuration for a supported LLM provider.
type ProviderDefaults struct {
	Name         string // litellm provider name: "openai", "deepseek", "anthropic"
	DefaultURL   string
	DefaultModel string
	KeyPattern   string // regex to validate API key format
	KeyExample   string // hint shown during setup
	Description  string // shown in setup wizard
}

// knownProviders maps provider IDs to their defaults.
// Only providers with first-class support are listed here.
var knownProviders = map[string]ProviderDefaults{
	"deepseek": {
		Name:         "deepseek",
		DefaultURL:   "https://api.deepseek.com",
		DefaultModel: "deepseek-chat",
		KeyPattern:   `^sk-.+`,
		KeyExample:   "sk-xxx",
		Description:  "DeepSeek（推荐，性价比高）",
	},
	"openai": {
		Name:         "openai",
		DefaultURL:   "https://api.openai.com/v1",
		DefaultModel: "gpt-4o",
		KeyPattern:   `^sk-.+`,
		KeyExample:   "sk-xxx",
		Description:  "OpenAI（GPT-4o 等）",
	},
	"anthropic": {
		Name:         "anthropic",
		DefaultURL:   "https://api.anthropic.com",
		DefaultModel: "claude-sonnet-4-20250514",
		KeyPattern:   `^sk-ant-.+`,
		KeyExample:   "sk-ant-xxx",
		Description:  "Anthropic（Claude 系列）",
	},
}

// defaultProvider is the provider used when none is specified.
const defaultProvider = "deepseek"

// providerOrder defines the display order in the setup wizard.
var providerOrder = []string{"deepseek", "openai", "anthropic"}

// GetProviderDefaults returns the defaults for a provider, or nil if unknown.
func GetProviderDefaults(provider string) *ProviderDefaults {
	if pd, ok := knownProviders[provider]; ok {
		return &pd
	}
	return nil
}

// ValidateKeyFormat checks if the API key matches the expected format for the provider.
// Returns nil if the provider is unknown (no format check).
func ValidateKeyFormat(provider, apiKey string) error {
	pd := GetProviderDefaults(provider)
	if pd == nil {
		return nil
	}
	matched, err := regexp.MatchString(pd.KeyPattern, apiKey)
	if err != nil {
		return nil // regex error shouldn't block the user
	}
	if !matched {
		return &KeyFormatError{Provider: provider, Example: pd.KeyExample}
	}
	return nil
}

// KeyFormatError indicates the API key doesn't match the expected format.
type KeyFormatError struct {
	Provider string
	Example  string
}

func (e *KeyFormatError) Error() string {
	return "API Key 格式不正确，通常以 " + e.Example + " 开头"
}

// inferProvider guesses the provider from a BaseURL for migration purposes.
func inferProvider(baseURL string) string {
	switch {
	case contains(baseURL, "deepseek"):
		return "deepseek"
	case contains(baseURL, "anthropic"):
		return "anthropic"
	default:
		return "openai"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
