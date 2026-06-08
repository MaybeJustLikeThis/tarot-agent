package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/store"
)

// Run starts the Bubble Tea TUI.
func Run(agent *agentcore.Agent, guard *reminder.ReadingGuard, s *store.Store, mode string) error {
	m := NewModel(agent, guard, s, mode)

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)

	_, err := p.Run()
	return err
}
