package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWelcomeMarkerOldConfigAndPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearTarotEnv(t)
	t.Setenv("TAROT_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	if WelcomeSeen() {
		t.Fatal("missing marker should be unseen")
	}

	dir := filepath.Join(home, ".tarot-agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte("{\n  \"provider\": \"openai\",\n  \"api_key\": \"sk-old\",\n  \"mode\": \"casual\"\n}\n")
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TAROT_API_KEY", "sk-env")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sk-env" || cfg.Provider != "openai" || cfg.Mode != "casual" {
		t.Fatalf("old config changed: %+v", cfg)
	}
	if WelcomeSeen() {
		t.Fatal("old config without a marker is unseen")
	}

	if err := MarkWelcomeSeen(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("config rewritten:\n%s", got)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o755 {
		t.Fatalf("existing dir mode %o, want 0755", dirInfo.Mode().Perm())
	}
	fileInfo, err := os.Stat(filepath.Join(dir, welcomeFile))
	if err != nil {
		t.Fatal(err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("marker mode %o, want 0600", fileInfo.Mode().Perm())
	}
	if !WelcomeSeen() {
		t.Fatal("marker should be seen")
	}
	if err := MarkWelcomeSeen(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkWelcomeSeenCreatesPrivateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if WelcomeSeen() {
		t.Fatal("unseen")
	}
	if err := MarkWelcomeSeen(); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".tarot-agent")
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o, want 0700", dirInfo.Mode().Perm())
	}
	fileInfo, err := os.Stat(filepath.Join(dir, welcomeFile))
	if err != nil {
		t.Fatal(err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("marker mode %o, want 0600", fileInfo.Mode().Perm())
	}
	if !WelcomeSeen() {
		t.Fatal("seen")
	}
}
