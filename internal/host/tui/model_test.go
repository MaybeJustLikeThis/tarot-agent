package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/store"
	"github.com/voocel/tarot-agent/internal/tools"
)

func lipglossWidth(s string) int { return lipgloss.Width(s) }

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func newTestModelSize(w, h int) *Model {
	s, _ := store.New()
	guard, _ := reminder.NewReadingGuard()
	agent := agentcore.NewAgent()

	ta := textarea.New()
	ta.Placeholder = "test"
	ta.Focus()
	ta.SetWidth(w - 4)
	ta.SetHeight(1)

	vp := viewport.New(w, h)

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot

	m := &Model{
		state:     &InputState{},
		bridge:    newAgentBridge(agent, guard),
		store:     s,
		input:     ta,
		readingVP: vp,
		spinner:   sp,
		width:     w,
		height:    h,
	}
	m.layoutHeights()
	return m
}

func newTestModel() *Model {
	return newTestModelSize(120, 40)
}

// Test: status bar is always the first line and contains expected content
func TestView_StatusBar_AlwaysFirstLine(t *testing.T) {
	m := newTestModel()
	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) == 0 {
		t.Fatal("view is empty")
	}

	firstLine := lines[0]
	if !strings.Contains(firstLine, "星语") {
		t.Errorf("first line missing '星语': %q", firstLine)
	}
}

// Test: status bar is exactly 1 line tall
func TestView_StatusBar_Height(t *testing.T) {
	m := newTestModel()
	bar := renderStatusBar(m)
	h := lipglossHeight(bar)
	if h != 1 {
		t.Errorf("status bar height: expected 1, got %d", h)
	}
}

// Test: total rendered lines should be close to terminal height
func TestView_TotalHeight_MatchesTerminal(t *testing.T) {
	tests := []struct {
		w, h int
	}{
		{120, 40},
		{80, 24},
		{100, 30},
		{200, 50},
	}

	for _, tt := range tests {
		m := newTestModelSize(tt.w, tt.h)
		view := m.View()
		actualLines := len(strings.Split(view, "\n"))
		if actualLines < tt.h-5 || actualLines > tt.h+5 {
			t.Errorf("%dx%d: expected ~%d lines, got %d", tt.w, tt.h, tt.h, actualLines)
		}
	}
}

// Test: layout body height is always positive and reasonable
func TestLayout_BodyHeight_AlwaysPositive(t *testing.T) {
	tests := []struct {
		w, h       int
		minBodyH   int
	}{
		{120, 40, 20},
		{80, 24, 10},
		{60, 20, 5},
		{40, 15, 3},
	}

	for _, tt := range tests {
		m := newTestModelSize(tt.w, tt.h)
		if m.layout.BodyHeight < tt.minBodyH {
			t.Errorf("%dx%d: body height %d < expected min %d",
				tt.w, tt.h, m.layout.BodyHeight, tt.minBodyH)
		}
	}
}

// Test: left + right widths always equal total width
func TestLayout_Widths_SumToTotal(t *testing.T) {
	tests := []struct {
		w, h int
	}{
		{120, 40},
		{80, 24},
		{100, 30},
		{150, 45},
	}

	for _, tt := range tests {
		m := newTestModelSize(tt.w, tt.h)
		sum := m.layout.LeftWidth + m.layout.RightWidth
		if sum != tt.w {
			t.Errorf("%dx%d: left(%d) + right(%d) = %d, expected %d",
				tt.w, tt.h, m.layout.LeftWidth, m.layout.RightWidth, sum, tt.w)
		}
	}
}

// Test: viewport dimensions are synced to right panel
func TestLayout_ViewportSynced(t *testing.T) {
	m := newTestModel()

	// After layoutHeights, viewport should match right panel
	if m.readingVP.Width != m.layout.RightWidth-4 {
		t.Errorf("viewport width: got %d, want %d",
			m.readingVP.Width, m.layout.RightWidth-4)
	}
	if m.readingVP.Height != m.layout.BodyHeight-3 {
		t.Errorf("viewport height: got %d, want %d",
			m.readingVP.Height, m.layout.BodyHeight-3)
	}
}

// Test: resize updates layout and viewport
func TestLayout_Resize_UpdatesAll(t *testing.T) {
	m := newTestModelSize(120, 40)

	// Simulate resize to smaller
	m.width = 80
	m.height = 24
	m.layoutHeights()

	if m.layout.Width != 80 {
		t.Errorf("width not updated: %d", m.layout.Width)
	}
	if m.layout.Height != 24 {
		t.Errorf("height not updated: %d", m.layout.Height)
	}
	if m.readingVP.Width != m.layout.RightWidth-4 {
		t.Errorf("viewport not synced after resize")
	}
}

// Test: resize multiple times doesn't break layout
func TestLayout_Resize_MultipleTimes(t *testing.T) {
	m := newTestModelSize(120, 40)

	sizes := []struct {
		w, h int
	}{
		{80, 24},
		{150, 50},
		{60, 20},
		{120, 40},
	}

	for _, s := range sizes {
		m.width = s.w
		m.height = s.h
		m.layoutHeights()

		if m.layout.LeftWidth+m.layout.RightWidth != s.w {
			t.Errorf("%dx%d: widths don't sum", s.w, s.h)
		}
		if m.layout.BodyHeight < 3 {
			t.Errorf("%dx%d: body height too small: %d", s.w, s.h, m.layout.BodyHeight)
		}
	}
}

// Test: reading state renders both panels
func TestView_ReadingState_BothPanels(t *testing.T) {
	m := newTestModel()
	m.spreadType = "three_card"

	result, _ := tools.DrawCards(m.store, "three_card")
	m.drawResult = result
	m.revealIndex = len(result.Cards)
	m.reading.WriteString("测试解读内容很长很长很长很长很长很长很长很长很长很长")
	m.readingVP.SetContent(m.reading.String())

	m.state = &ReadingState{}
	view := m.View()

	if !strings.Contains(view, "牌面") {
		t.Error("missing left panel")
	}
	if !strings.Contains(view, "解读") {
		t.Error("missing right panel")
	}
}

// Test: no duplicate content in any state
func TestView_NoDuplicateContent(t *testing.T) {
	states := []struct {
		name  string
		state State
	}{
		{"Input", &InputState{}},
		{"Spread", &SpreadState{}},
	}

	for _, tc := range states {
		m := newTestModel()
		m.state = tc.state
		view := m.View()

		inputCount := strings.Count(view, "说说你的情况和想问的问题")
		if inputCount > 1 {
			t.Errorf("%s: input label appears %d times", tc.name, inputCount)
		}
	}
}

// Test: each line of the view should not exceed terminal width
func TestView_NoLineExceedsWidth(t *testing.T) {
	m := newTestModelSize(120, 40)
	m.spreadType = "three_card"

	result, _ := tools.DrawCards(m.store, "three_card")
	m.drawResult = result
	m.revealIndex = len(result.Cards)
	m.reading.WriteString("测试解读内容")
	m.readingVP.SetContent(m.reading.String())

	m.state = &ReadingState{}
	view := m.View()

	lines := strings.Split(view, "\n")
	for i, line := range lines {
		lineW := lipglossWidth(line)
		if lineW > 120 {
			t.Errorf("line %d exceeds width 120: actual %d chars: %q", i, lineW, truncateString(line, 60))
		}
	}
}

// Test: full flow simulation — input → spread → reveal → reading → followup → input
func TestView_FullFlow_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic during full flow: %v", r)
		}
	}()

	m := newTestModel()

	// 1. InputState — render view
	_ = m.View()

	// 2. Transition to SpreadState
	m.state = &SpreadState{}
	_ = m.View()

	// 3. Transition to RevealState (draw cards)
	result, _ := tools.DrawCards(m.store, "three_card")
	m.drawResult = result
	m.spreadType = "three_card"
	m.revealIndex = 0
	m.state = &RevealState{}

	// Render during reveal
	_ = m.View()

	// Reveal all cards
	m.revealIndex = len(result.Cards)
	_ = m.View()

	// 4. Transition to ReadingState
	m.reading.WriteString("这是一段很长的解读内容，模拟AI生成的解读文本。")
	m.readingVP.SetContent(m.reading.String())
	m.state = &ReadingState{}
	_ = m.View()

	// 5. Transition to FollowUpState
	m.state = &FollowUpState{}
	_ = m.View()

	// 6. Back to InputState (new reading)
	m.drawResult = nil
	m.reading.Reset()
	m.state = &InputState{}
	_ = m.View()
}

// Test: RevealState with nil drawResult
func TestView_RevealState_NilDrawResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic with nil drawResult: %v", r)
		}
	}()

	m := newTestModel()
	m.drawResult = nil
	m.state = &RevealState{}
	_ = m.View()
}

// Test: ReadingState with nil drawResult
func TestView_ReadingState_NilDrawResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic with nil drawResult: %v", r)
		}
	}()

	m := newTestModel()
	m.drawResult = nil
	m.state = &ReadingState{}
	_ = m.View()
}

// Test: FollowUpState with nil drawResult
func TestView_FollowUpState_NilDrawResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic with nil drawResult: %v", r)
		}
	}()

	m := newTestModel()
	m.drawResult = nil
	m.state = &FollowUpState{}
	_ = m.View()
}

// Test: small terminal shows warning instead of layout
func TestView_SmallTerminal_ShowsWarning(t *testing.T) {
	smallSizes := []struct {
		w, h int
	}{
		{40, 10},
		{60, 15},
		{70, 19},
	}

	for _, s := range smallSizes {
		m := newTestModelSize(s.w, s.h)
		view := m.View()

		if !strings.Contains(view, "终端窗口太小") {
			t.Errorf("%dx%d: should show terminal size warning", s.w, s.h)
		}
	}
}

// Test: small terminal doesn't panic
func TestView_SmallTerminal_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic on small terminal: %v", r)
		}
	}()

	sizes := []struct {
		w, h int
	}{
		{40, 10},
		{30, 8},
		{20, 5},
	}

	for _, s := range sizes {
		m := newTestModelSize(s.w, s.h)
		_ = m.View()
	}
}
