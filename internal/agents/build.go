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

//go:embed prompts/system.md
var systemPrompt string

// BuildResult holds the agent and its associated guard for event tracking.
type BuildResult struct {
	Agent *agentcore.Agent
	Guard *reminder.ReadingGuard
}

// BuildAgent assembles the main tarot reading agent with all tools
// and a StopGuard that ensures readings are completed properly.
func BuildAgent(model *llm.LiteLLMAdapter, s *store.Store) *BuildResult {
	guard, guardFunc := reminder.NewReadingGuard()

	agent := agentcore.NewAgent(
		agentcore.WithModel(model),
		agentcore.WithSystemPrompt(systemPrompt),
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
	}
}
