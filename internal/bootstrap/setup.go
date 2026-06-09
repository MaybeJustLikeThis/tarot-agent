package bootstrap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	Provider string `json:"provider,omitempty"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
	Model    string `json:"model,omitempty"`
	Mode     string `json:"mode,omitempty"`
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

	// Step 1: Provider selection
	fmt.Println("  选择 AI 服务商：")
	for i, key := range providerOrder {
		pd := knownProviders[key]
		fmt.Printf("    %d. %s\n", i+1, pd.Description)
	}
	fmt.Print("  请选择 [1/2/3]（默认 1）：> ")

	providerInput, _ := reader.ReadString('\n')
	providerInput = strings.TrimSpace(providerInput)
	providerIdx := 0
	if providerInput == "2" {
		providerIdx = 1
	} else if providerInput == "3" {
		providerIdx = 2
	}
	providerKey := providerOrder[providerIdx]
	pd := knownProviders[providerKey]

	// Step 2: Base URL (with default from provider)
	fmt.Printf("\n  API 地址 [%s]：> ", pd.DefaultURL)
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = pd.DefaultURL
	}

	// Step 3: Model (with default from provider)
	fmt.Printf("  模型名 [%s]：> ", pd.DefaultModel)
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = pd.DefaultModel
	}

	// Step 4: API Key with format validation
	fmt.Printf("  请粘贴你的 API Key（%s）：> ", pd.KeyExample)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("API key 不能为空")
	}
	if err := ValidateKeyFormat(providerKey, apiKey); err != nil {
		fmt.Printf("  ⚠ %s\n", err)
		fmt.Print("  是否仍要使用此 Key？[y/N]：> ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			return nil, fmt.Errorf("已取消，请重新运行设置")
		}
	}

	// Step 5: Reading mode
	fmt.Println()
	fmt.Println("  选择解读模式：")
	fmt.Println("    1. 专业模式 — 包含元素、占星、数字等深度分析")
	fmt.Println("    2. 轻松模式 — 温暖对话式解读，不使用专业术语")
	fmt.Print("  请选择 [1/2]（默认 1）：> ")

	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)
	mode := "professional"
	if modeInput == "2" {
		mode = "casual"
	}

	// Save to file
	fc := &fileConfig{
		Provider: providerKey,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
		Mode:     mode,
	}
	if err := saveFileConfig(fc); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	// Validate API key connectivity (OpenAI-compatible endpoints only)
	if providerKey != "anthropic" {
		fmt.Print("  验证 API Key...")
		if err := validateAPIEndpoint(baseURL, apiKey); err != nil {
			fmt.Printf(" ❌ 验证失败: %v\n", err)
			fmt.Println("  配置已保存，但 API Key 可能无效。你可以稍后修改 ~/.tarot-agent/config.json")
		} else {
			fmt.Println(" ✅")
		}
	}

	cfgPath, _ := configPath()
	fmt.Printf("\n  ✅ 配置已保存到 %s\n", cfgPath)
	fmt.Println("  下次启动无需重复输入。")
	fmt.Println()

	return &Config{
		Provider: fc.Provider,
		APIKey:   fc.APIKey,
		BaseURL:  fc.BaseURL,
		Model:    fc.Model,
		Mode:     fc.Mode,
		LogLevel: defaultLogLevel,
	}, nil
}

// validateAPIEndpoint checks connectivity by hitting the /v1/models endpoint.
// Only works for OpenAI-compatible APIs (OpenAI, DeepSeek, etc.).
func validateAPIEndpoint(baseURL, apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("API Key 无效 (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("服务端错误 (HTTP %d)，稍后重试", resp.StatusCode)
	}
	// 200/404/etc are acceptable — the key itself is valid.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
