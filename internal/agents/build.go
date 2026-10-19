package agents

import (
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/host/reminder"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

//go:embed prompts/system_pro.md
var systemPromptPro string

//go:embed prompts/system_casual.md
var systemPromptCasual string

// maxAgentTurns is large enough for a normal Celtic cross.
// 10 card meanings + spread + disclaimer + save + final text is about 14
// turns when each tool call is its own turn; 30 leaves room for guard retries.
const maxAgentTurns = 30

// AgentMode controls the depth and style of tarot readings.
type AgentMode string

const (
	ModeProfessional AgentMode = "professional"
	ModeCasual       AgentMode = "casual"
)

// NormalizeMode accepts the existing spellings and reports whether raw was valid.
// Empty, "pro", "professional", and "deep" are professional.
// "casual" and "simple" are casual.
// Any other non-empty value is invalid and falls back to professional.
func NormalizeMode(raw string) (AgentMode, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "pro", "professional", "deep":
		return ModeProfessional, true
	case "casual", "simple":
		return ModeCasual, true
	default:
		return ModeProfessional, false
	}
}

// ParseMode converts a string to AgentMode, defaulting to professional.
func ParseMode(s string) AgentMode {
	mode, _ := NormalizeMode(s)
	return mode
}

// ResolveMode applies a configured mode and an optional --mode flag.
// The flag wins when it is non-empty. Invalid values fall back to professional
// and return a warning instead of failing startup.
func ResolveMode(configured, flag string) (AgentMode, string) {
	raw := configured
	if strings.TrimSpace(flag) != "" {
		raw = flag
	}
	mode, ok := NormalizeMode(raw)
	if ok {
		return mode, ""
	}
	return mode, fmt.Sprintf("警告: 解读模式 %q 无效，已回退到深入版（可用 professional、pro、deep、casual 或 simple）", strings.TrimSpace(raw))
}

// Label returns the Chinese display name for the mode.
func (m AgentMode) Label() string {
	if m == ModeCasual {
		return "简洁版"
	}
	return "深入版"
}

// BuildResult holds the agent and its associated guard for event tracking.
type BuildResult struct {
	Agent *agentcore.Agent
	Guard *reminder.ReadingGuard
	Mode  AgentMode
}

// BuildAgent assembles the main tarot reading agent with all tools
// and a StopGuard that ensures readings are completed properly.
func BuildAgent(model *llm.LiteLLMAdapter, s *store.Store, mode AgentMode) *BuildResult {
	prompt := systemPromptPro
	if mode == ModeCasual {
		prompt = systemPromptCasual
	}

	guard, guardFunc := reminder.NewReadingGuard()

	agent := agentcore.NewAgent(
		agentcore.WithModel(model),
		agentcore.WithSystemPrompt(prompt),
		agentcore.WithTools(
			tools.GetCardMeaningTool(s),
			tools.GetSpreadLayoutTool(s),
			tools.GetDisclaimerTool(),
			tools.SaveReadingTool(s),
		),
		// Celtic cross can spend one turn per card lookup, plus spread,
		// disclaimer, save, the final reading, and a few guard retries.
		agentcore.WithMaxTurns(maxAgentTurns),
		agentcore.WithStopGuard(guardFunc),
		agentcore.WithOnMessage(func(msg agentcore.AgentMessage) {
			slog.Debug("agent message",
				"role", string(msg.GetRole()),
				"content_len", len(msg.TextContent()),
			)
		}),
	)

	return &BuildResult{
		Agent: agent,
		Guard: guard,
		Mode:  mode,
	}
}
