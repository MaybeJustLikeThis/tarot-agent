package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/agentcore"
)

func TestPresentAgentError_MaxTurns(t *testing.T) {
	err := presentAgentError(&agentcore.MaxTurnsError{Limit: 30})
	if !strings.Contains(err.Error(), "30") || !strings.Contains(err.Error(), "轮数上限") {
		t.Fatalf("got %s", err)
	}
	if got := presentAgentError(errString("boom")); got.Error() != "boom" {
		t.Fatalf("passthrough: %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestRetryIsVisibleAndNotAnError(t *testing.T) {
	m := newTestModel()
	m.state = &ReadingState{}
	updated, _ := m.Update(agentRetryMsg{note: "模型请求失败，正在重试（2/3）…"})
	mm := updated.(*Model)
	if mm.err != nil {
		t.Fatalf("retry stored as error: %v", mm.err)
	}
	view := mm.View()
	if !strings.Contains(view, "正在重试") || !strings.Contains(view, "2/3") {
		t.Fatalf("retry not visible:\n%s", view)
	}
	updated, _ = mm.Update(agentDeltaMsg{text: "甲"})
	mm = updated.(*Model)
	if mm.retryNote != "" {
		t.Fatalf("retry note stuck: %s", mm.retryNote)
	}
}

func TestStreamFlushCoalescesAndFinalMarkdownWins(t *testing.T) {
	m := newTestModel()
	m.store.Readings = store.NewReadingStore(filepath.Join(t.TempDir(), "readings.jsonl"))
	m.userInput = "这个问题怎么看？"
	m.spreadType = "single"
	m.state = &ReadingState{}

	updated, _ := m.Update(agentDeltaMsg{text: "**重点**"})
	mm := updated.(*Model)
	if strings.Contains(mm.readingVP.View(), "重点") {
		t.Fatal("delta rendered before the flush interval")
	}
	updated, _ = mm.Update(agentDeltaMsg{text: "说明"})
	mm = updated.(*Model)
	if !mm.streamFlushPending {
		t.Fatal("second delta should reuse the pending flush")
	}
	gen := mm.streamGen
	updated, _ = mm.Update(streamFlushMsg{gen: gen})
	mm = updated.(*Model)
	plain := mm.readingVP.View()
	if !strings.Contains(plain, "**重点**说明") {
		t.Fatalf("plain flush: %q", plain)
	}
	if mm.streamFlushPending {
		t.Fatal("flush did not clear the pending flag")
	}

	updated, _ = mm.Update(agentEndMsg{})
	mm = updated.(*Model)
	if _, ok := mm.state.(*FollowUpState); !ok {
		t.Fatalf("state %T", mm.state)
	}
	final := mm.readingVP.View()
	if strings.Contains(final, "**") || !strings.Contains(final, "重点") {
		t.Fatalf("final markdown: %q", final)
	}
	updated, _ = mm.Update(streamFlushMsg{gen: gen})
	mm = updated.(*Model)
	if strings.Contains(mm.readingVP.View(), "**") || !strings.Contains(mm.readingVP.View(), "重点") {
		t.Fatalf("stale flush overwrote markdown: %q", mm.readingVP.View())
	}
}

func TestChatStreamEndsWithMarkdown(t *testing.T) {
	m := newTestModel()
	m.state = &ChatState{}
	m.layoutHeights()
	m.appendChatUserMessage("追问一句")
	updated, _ := m.Update(agentDeltaMsg{text: "**可以**先写下来"})
	mm := updated.(*Model)
	updated, _ = mm.Update(streamFlushMsg{gen: mm.streamGen})
	mm = updated.(*Model)
	if strings.Contains(mm.chatVP.View(), "可以") && !strings.Contains(mm.chatVP.View(), "**可以**") {
		t.Fatal("chat flush rendered markdown early")
	}
	updated, _ = mm.Update(agentEndMsg{})
	mm = updated.(*Model)
	view := mm.chatVP.View()
	if strings.Contains(view, "**") || !strings.Contains(view, "可以") {
		t.Fatalf("chat final: %q", view)
	}
	if _, ok := mm.state.(*FollowUpState); !ok {
		t.Fatalf("state %T", mm.state)
	}
}

func TestCtrlCAbortsReadingAndChat(t *testing.T) {
	for _, state := range []State{&ReadingState{}, &ChatState{}} {
		m := newTestModel()
		m.state = state
		m.bridge.unsub = func() {}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatal("missing quit cmd")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%T: quit cmd returned %T", state, cmd())
		}
		mm := updated.(*Model)
		if !mm.bridge.aborted {
			t.Fatalf("%T: abort was not called", state)
		}
		if mm.bridge.unsub != nil {
			t.Fatalf("%T: cleanup did not unsubscribe", state)
		}
	}
}

func TestSpinnerStopsInInputAndRestartsForReveal(t *testing.T) {
	m := newTestModel()
	m.spinnerActive = true
	_, cmd := m.Update(m.spinner.Tick())
	if cmd != nil || m.spinnerActive {
		t.Fatal("input state kept the spinner chain alive")
	}

	m.state = &ReadingState{}
	before := m.spinner.View()
	updated, cmd := m.Update(m.spinner.Tick())
	mm := updated.(*Model)
	if cmd == nil {
		t.Fatal("reading spinner did not schedule the next frame")
	}
	if mm.spinner.View() == before {
		t.Fatal("spinner frame did not advance")
	}

	m = newTestModel()
	m.state = &SpreadState{}
	m.spinnerActive = false
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	mm = updated.(*Model)
	if _, ok := mm.state.(*RevealState); !ok {
		t.Fatalf("state %T err=%v", mm.state, mm.err)
	}
	if !mm.spinnerActive || cmd == nil {
		t.Fatal("entering reveal did not restart the spinner")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) < 2 {
		t.Fatalf("expected reveal tick and spinner tick, got %T", cmd())
	}
}

func TestCardFaceMatchesPlaceholderHeight(t *testing.T) {
	card := domain.DrawnCard{
		Card:        domain.Card{NameCN: "愚者", NameEN: "The Fool"},
		Orientation: domain.Upright,
	}
	front := renderCardFrontSized(card, 10)
	placeholder := renderCardPlaceholderSized(10)
	if lipgloss.Height(front) != lipgloss.Height(placeholder) {
		t.Fatalf("front height %d, placeholder %d", lipgloss.Height(front), lipgloss.Height(placeholder))
	}
}
