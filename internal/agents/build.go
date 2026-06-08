package agents

import (
	_ "embed"
	"log/slog"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/store"
	"github.com/voocel/tarot-agent/internal/tools"
)

//go:embed prompts/system_pro.md
var systemPromptPro string

//go:embed prompts/system_casual.md
var systemPromptCasual string

// AgentMode controls the depth and style of tarot readings.
type AgentMode string

const (
	ModeProfessional AgentMode = "professional"
	ModeCasual       AgentMode = "casual"
)

// ParseMode converts a string to AgentMode, defaulting to professional.
func ParseMode(s string) AgentMode {
	if s == "casual" {
		return ModeCasual
	}
	return ModeProfessional
}

// Label returns the Chinese display name for the mode.
func (m AgentMode) Label() string {
	if m == ModeCasual {
		return "轻松模式"
	}
	return "专业模式"
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
		agentcore.WithMaxTurns(15),
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
