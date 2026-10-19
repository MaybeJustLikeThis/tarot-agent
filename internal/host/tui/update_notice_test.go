package tui

import (
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/update"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateNoticeFitsAndDismisses(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.version = "0.5.0"
	notice := update.ShortNotice("0.5.0", "v0.6.0")
	updated, _ := m.Update(updateAvailableMsg{text: notice})
	m = updated.(*Model)
	if m.updateNotice == "" {
		t.Fatal("notice was not stored")
	}
	view := m.View()
	for _, want := range []string{"0.5.0 → v0.6.0", "tarot-agent update", "输入 q 回车退出"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q\n%s", want, view)
		}
	}
	assertFitsScreen(t, "update-notice", view, 24)
	assertLineWidths(t, "update-notice", view, 80)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(*Model)
	if m.updateNotice != "" {
		t.Fatal("key did not dismiss the notice")
	}
	if m.input.Value() != "a" {
		t.Fatalf("key was swallowed, input %q", m.input.Value())
	}
	if _, ok := m.state.(*InputState); !ok {
		t.Fatalf("state %T", m.state)
	}
}

func TestUpdateNoticeStaysOffTheReading(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.state = &ReadingState{}
	updated, _ := m.Update(updateAvailableMsg{text: update.ShortNotice("0.5.0", "v0.6.0")})
	m = updated.(*Model)
	if m.updateNotice != "" {
		t.Fatal("reading picked up the startup notice")
	}
	if _, ok := m.state.(*ReadingState); !ok {
		t.Fatalf("state %T", m.state)
	}
}
