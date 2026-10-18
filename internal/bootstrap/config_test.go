package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAPIKey(t *testing.T) {
	const (
		saved    = "sk-saved"
		tarot    = "sk-tarot"
		deepseek = "sk-deepseek"
	)
	cases := []struct {
		name     string
		saved    string
		provider string
		tarot    string
		deepseek string
		want     string
	}{
		{"tarot key wins over everything", saved, "openai", tarot, deepseek, tarot},
		{"tarot key wins with no file", "", "deepseek", tarot, deepseek, tarot},
		{"deepseek env starts with no config", "", "deepseek", "", deepseek, deepseek},
		{"deepseek env fills an empty saved key", "", "openai", "", deepseek, deepseek},
		{"deepseek env does not replace another provider", saved, "openai", "", deepseek, saved},
		{"deepseek env does not replace anthropic", saved, "anthropic", "", deepseek, saved},
		{"deepseek env replaces a saved deepseek key", saved, "deepseek", "", deepseek, deepseek},
		{"no env keeps the saved key", saved, "openai", "", "", saved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveAPIKey(tc.saved, tc.provider, tc.tarot, tc.deepseek)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadConfig_KeyPriority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearTarotEnv(t)

	t.Run("deepseek env without a file", func(t *testing.T) {
		t.Setenv("DEEPSEEK_API_KEY", "sk-from-env")
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.APIKey != "sk-from-env" {
			t.Fatalf("key %q", cfg.APIKey)
		}
		if cfg.Provider != "deepseek" {
			t.Fatalf("provider %q", cfg.Provider)
		}
	})

	t.Run("saved openai key is kept", func(t *testing.T) {
		writeTestConfig(t, home, &fileConfig{
			Provider: "openai",
			APIKey:   "sk-openai",
			Model:    "gpt-4o",
		})
		t.Setenv("DEEPSEEK_API_KEY", "sk-from-env")
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.APIKey != "sk-openai" || cfg.Provider != "openai" {
			t.Fatalf("provider=%s key=%s", cfg.Provider, cfg.APIKey)
		}
	})

	t.Run("tarot api key still overrides the file", func(t *testing.T) {
		t.Setenv("DEEPSEEK_API_KEY", "sk-from-env")
		t.Setenv("TAROT_API_KEY", "sk-tarot")
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.APIKey != "sk-tarot" {
			t.Fatalf("key %q", cfg.APIKey)
		}
	})
}

func TestLoadConfig_ModeStaysRaw(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearTarotEnv(t)
	t.Setenv("TAROT_API_KEY", "sk-test")
	t.Setenv("TAROT_MODE", "pro")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "pro" {
		t.Fatalf("mode %q", cfg.Mode)
	}
}

func TestSaveFileConfig_Permissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := saveFileConfig(&fileConfig{Provider: "openai", APIKey: "sk-test", Mode: "professional"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(home, ".tarot-agent", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode %o, want 0600", info.Mode().Perm())
	}
}

func clearTarotEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"TAROT_API_KEY",
		"DEEPSEEK_API_KEY",
		"TAROT_PROVIDER",
		"TAROT_BASE_URL",
		"TAROT_MODEL",
		"TAROT_MODE",
		"TAROT_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}
}

func writeTestConfig(t *testing.T, home string, fc *fileConfig) {
	t.Helper()
	if err := saveFileConfig(fc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".tarot-agent", "config.json")); err != nil {
		t.Fatal(err)
	}
}
