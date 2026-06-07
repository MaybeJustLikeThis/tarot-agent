package bootstrap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configDir returns the tarot-agent config directory (~/.tarot-agent).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".tarot-agent"), nil
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// fileConfig is the JSON structure persisted to disk.
type fileConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
}

// loadFileConfig reads config from ~/.tarot-agent/config.json if it exists.
func loadFileConfig() (*fileConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not an error, just no file
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &fc, nil
}

// saveFileConfig writes config to ~/.tarot-agent/config.json atomically.
func saveFileConfig(fc *fileConfig) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Atomic write: tmp + rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// NeedsSetup returns true if no API key is available (neither file nor env).
func NeedsSetup() bool {
	// Check env vars first
	if os.Getenv("DEEPSEEK_API_KEY") != "" || os.Getenv("TAROT_API_KEY") != "" {
		return false
	}
	// Check file
	fc, err := loadFileConfig()
	if err != nil || fc == nil {
		return true
	}
	return fc.APIKey == ""
}

// RunSetup runs the interactive first-time setup wizard.
// Returns the populated Config.
func RunSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════╗")
	fmt.Println("  ║       星语 Tarot Agent — 首次设置      ║")
	fmt.Println("  ╚═══════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  本工具需要 DeepSeek API Key 来驱动 AI 解读。")
	fmt.Println("  获取方式：https://platform.deepseek.com/api_keys")
	fmt.Println()
	fmt.Print("  请粘贴你的 API Key（sk-xxx）：> ")

	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("API key 不能为空")
	}

	// Save to file
	fc := &fileConfig{
		APIKey:  apiKey,
		BaseURL: defaultBaseURL,
		Model:   defaultModel,
	}
	if err := saveFileConfig(fc); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	cfgPath, _ := configPath()
	fmt.Printf("\n  ✅ 配置已保存到 %s\n", cfgPath)
	fmt.Println("  下次启动无需重复输入。")
	fmt.Println()

	return &Config{
		APIKey:   fc.APIKey,
		BaseURL:  fc.BaseURL,
		Model:    fc.Model,
		LogLevel: defaultLogLevel,
	}, nil
}
