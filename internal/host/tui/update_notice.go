package tui

import (
	"context"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/update"
	tea "github.com/charmbracelet/bubbletea"
)

// updateAvailableMsg is delivered once when a newer release exists.
// A nil command result means the check failed or nothing is newer.
type updateAvailableMsg struct {
	text string
}

func checkUpdateCmd(current string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), update.CheckTimeout)
		defer cancel()
		res, err := update.Check(ctx, nil, current)
		if err != nil || res == nil || !res.Newer {
			return nil
		}
		return updateAvailableMsg{text: update.ShortNotice(res.Current, res.Latest)}
	}
}
