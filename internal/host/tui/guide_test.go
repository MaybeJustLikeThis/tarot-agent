package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuestionGuideFlow(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("他爱不爱我")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*QuestionGuideState); !ok {
		t.Fatalf("state %T, want QuestionGuideState", m.state)
	}
	if m.userInput != "他爱不爱我" {
		t.Fatalf("original question = %q", m.userInput)
	}
	view := m.View()
	if !strings.Contains(view, loveSuggest) || !strings.Contains(view, "采纳改写") {
		t.Fatalf("guide view missing suggestion:\n%s", view)
	}
	assertFitsScreen(t, "guide", view, 24)
	assertLineWidths(t, "guide", view, 80)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("after accept state %T", m.state)
	}
	if m.userInput != loveSuggest {
		t.Fatalf("accepted question = %q", m.userInput)
	}
	if m.spreadAdvice.SpreadID != "three_card" {
		t.Fatalf("spread %s, want three_card", m.spreadAdvice.SpreadID)
	}
	spreadView := m.View()
	if !strings.Contains(spreadView, "← 推荐") || !strings.Contains(spreadView, spreadThreeReason) {
		t.Fatalf("spread view missing recommendation:\n%s", spreadView)
	}
	if !strings.Contains(spreadView, "esc 返回 · q 退出") {
		t.Fatal("spread keys hint changed")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(*Model)
	if _, ok := m.state.(*RevealState); !ok {
		t.Fatalf("key 1 state %T", m.state)
	}
	if m.spreadType != "single" {
		t.Fatalf("key 1 drew %s", m.spreadType)
	}
}

func TestQuestionGuideKeepOriginal(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("他爱不爱我")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("state %T", m.state)
	}
	if m.userInput != "他爱不爱我" {
		t.Fatalf("kept question = %q", m.userInput)
	}
	if m.spreadAdvice.SpreadID != "single" {
		t.Fatalf("short question spread %s", m.spreadAdvice.SpreadID)
	}
	if !strings.Contains(m.View(), "1") || !strings.Contains(m.View(), "2") || !strings.Contains(m.View(), "3") {
		t.Fatal("manual spread choices missing")
	}
}

func TestQuestionGuideEscReturnsToInput(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("我们什么时候能在一起")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*Model)
	if _, ok := m.state.(*InputState); !ok {
		t.Fatalf("state %T", m.state)
	}
	if m.input.Value() != "我们什么时候能在一起" {
		t.Fatalf("input = %q", m.input.Value())
	}
}

func TestOpenQuestionSkipsGuide(t *testing.T) {
	m := newTestModelSize(80, 24)
	q := "我和伴侣最近总是因为钱和未来规划吵架，接下来三个月该怎么走？"
	m.input.SetValue(q)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("state %T, want SpreadState", m.state)
	}
	if m.userInput != q {
		t.Fatalf("question changed: %q", m.userInput)
	}
	if m.spreadAdvice.SpreadID != "celtic_cross" {
		t.Fatalf("spread %s", m.spreadAdvice.SpreadID)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(*Model)
	if m.spreadType != "three_card" {
		t.Fatalf("key 2 drew %s", m.spreadType)
	}
}

func TestDailyCardSkipsGuide(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("他爱不爱我")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(*Model)
	if _, ok := m.state.(*RevealState); !ok {
		t.Fatalf("state %T", m.state)
	}
	if m.userInput != "今日指引" || m.spreadType != "single" {
		t.Fatalf("daily input=%q spread=%s", m.userInput, m.spreadType)
	}
}
