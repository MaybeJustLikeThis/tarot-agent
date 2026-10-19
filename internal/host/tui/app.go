package tui

import (
	"os"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/host/reminder"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/agentcore"
)

// Run starts the Bubble Tea TUI.
// TAROT_AGENT_TIMEOUT, when set, overrides the per-spread wall-clock limit.
func Run(agent *agentcore.Agent, guard *reminder.ReadingGuard, s *store.Store, mode, llmModel, version string, checkUpdate bool) error {
	override, set, err := parseAgentTimeoutEnv(os.Getenv("TAROT_AGENT_TIMEOUT"))
	if err != nil {
		return err
	}

	m := NewModel(agent, guard, s, mode, llmModel)
	m.version = version
	m.checkUpdates = checkUpdate
	m.timeoutOverride = override
	m.timeoutOverrideSet = set
	applyWelcome(m)

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)

	_, err = p.Run()
	return err
}
