package tui

import (
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/bootstrap"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWelcomeOnceSkipAndReplay(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if bootstrap.WelcomeSeen() {
		t.Fatal("marker should start unseen")
	}
	m := newTestModelSize(80, 24)
	applyWelcome(m)
	if _, ok := m.state.(*WelcomeState); !ok {
		t.Fatalf("first launch state %T", m.state)
	}

	view := m.View()
	assertWelcomeVisible(t, "80x24", view)
	assertLineWidths(t, "welcome-80x24", view, 80)
	assertFitsScreen(t, "welcome-80x24", view, 24)

	narrow := newTestModelSize(80, 20)
	narrow.state = &WelcomeState{}
	narrowView := narrow.View()
	assertWelcomeVisible(t, "80x20", narrowView)
	assertLineWidths(t, "welcome-80x20", narrowView, 80)
	assertFitsScreen(t, "welcome-80x20", narrowView, 20)

	tooSmall := newTestModelSize(60, 20)
	tooSmall.state = &WelcomeState{}
	smallView := tooSmall.View()
	if !strings.Contains(smallView, "终端窗口太小") {
		t.Fatalf("small terminal: %q", smallView)
	}
	assertLineWidths(t, "welcome-60x20", smallView, 60)
	updated, _ := tooSmall.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if _, ok := updated.(*Model).state.(*WelcomeState); !ok {
		t.Fatal("a key on a too-small terminal should keep the guide")
	}
	if bootstrap.WelcomeSeen() {
		t.Fatal("too-small skip must not mark the guide seen")
	}

	updated, cmd := narrow.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q on the guide should skip, not quit")
	}
	mm := updated.(*Model)
	if _, ok := mm.state.(*InputState); !ok {
		t.Fatalf("after skip state %T", mm.state)
	}
	if !bootstrap.WelcomeSeen() {
		t.Fatal("skip should mark the guide seen")
	}

	again := newTestModelSize(80, 24)
	applyWelcome(again)
	if _, ok := again.state.(*InputState); !ok {
		t.Fatalf("second launch state %T", again.state)
	}

	again.input.SetValue("")
	updated, _ = again.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if _, ok := updated.(*Model).state.(*WelcomeState); !ok {
		t.Fatal("empty input ? should reopen the guide")
	}
	if !bootstrap.WelcomeSeen() {
		t.Fatal("reopening should keep the marker")
	}

	typing := newTestModelSize(80, 24)
	typing.input.SetValue("接下来")
	updated, _ = typing.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	typed := updated.(*Model)
	if _, ok := typed.state.(*InputState); !ok {
		t.Fatalf("typing ? changed state to %T", typed.state)
	}
	if !strings.Contains(typed.input.Value(), "?") {
		t.Fatalf("question mark was not typed: %q", typed.input.Value())
	}
}

func TestWelcomeCtrlCQuitsAndMarks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	m := newTestModelSize(80, 20)
	m.state = &WelcomeState{}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("missing quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("quit cmd returned %T", cmd())
	}
	if _, ok := updated.(*Model).state.(*WelcomeState); !ok {
		t.Fatal("ctrl+c should leave the guide in place while quitting")
	}
	if !bootstrap.WelcomeSeen() {
		t.Fatal("ctrl+c after the guide is visible should mark it seen")
	}
}

func assertWelcomeVisible(t *testing.T, label, view string) {
	t.Helper()
	for _, phrase := range []string{
		"接下来三个月",
		"这段关系",
		"这个选择",
		"深入版",
		"简洁版",
		"tab",
		"追问",
		"e 导出",
		"daily",
		"card",
		"export",
		"update",
		"任意键",
		"?",
	} {
		if !strings.Contains(view, phrase) {
			t.Errorf("%s: missing %q\n%s", label, phrase, view)
		}
	}
}
