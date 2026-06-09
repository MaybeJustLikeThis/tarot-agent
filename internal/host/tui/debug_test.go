package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/store"
	"github.com/voocel/tarot-agent/internal/tools"
)

func captureModel(w, h int) *Model {
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

func dumpView(t *testing.T, name string, m *Model) {
	t.Helper()
	view := m.View()

	// Strip ANSI codes for readable output
	clean := stripAnsi(view)

	var b strings.Builder
	b.WriteString("=== " + name + " ===\n")
	b.WriteString("Terminal: ")
	b.WriteString("Width=" + fmt.Sprint(m.width) + " Height=" + fmt.Sprint(m.height) + "\n")
	b.WriteString("Layout: ")
	b.WriteString("LeftW=" + fmt.Sprint(m.layout.LeftWidth) + " RightW=" + fmt.Sprint(m.layout.RightWidth))
	b.WriteString(" BodyH=" + fmt.Sprint(m.layout.BodyHeight) + "\n")
	b.WriteString("State: " + stateDisplayName(m.state) + "\n")
	b.WriteString("---\n")
	b.WriteString(clean)
	b.WriteString("\n\n")

	// Write to file
	f, err := os.OpenFile("testdata/actual_output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()
	f.WriteString(b.String())
}

func stripAnsi(s string) string {
	// Simple ANSI escape code stripper
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func TestDump_ActualOutput(t *testing.T) {
	os.MkdirAll("testdata", 0755)
	os.Remove("testdata/actual_output.txt")

	// 1. InputState (120x40)
	m := captureModel(120, 40)
	dumpView(t, "InputState (120x40)", m)

	// 2. SpreadState
	m.state = &SpreadState{}
	dumpView(t, "SpreadState (120x40)", m)

	// 3. RevealState — during reveal
	m.spreadType = "three_card"
	m.layoutHeights()
	result, _ := tools.DrawCards(m.store, "three_card")
	m.drawResult = result
	m.revealIndex = 1
	m.state = &RevealState{}
	dumpView(t, "RevealState partial (120x40)", m)

	// 4. RevealState — all revealed
	m.revealIndex = len(result.Cards)
	dumpView(t, "RevealState complete (120x40)", m)

	// 5. ReadingState
	m.readingBuf.WriteString("根据牌面显示，你最近在工作上遇到了一个重要的选择。女皇牌出现在过去的位置，说明你之前的状态是丰盛和稳定的。\n\n塔牌逆位出现在现在的位置，暗示你可能正在经历一些内在的转变，旧的结构正在被打破。\n\n星星牌正位在未来位置，这是一个非常积极的信号，说明你最终会找到新的方向和希望。")
	m.readingVP.SetContent(m.readingBuf.String())
	m.readingVP.GotoBottom()
	m.state = &ReadingState{}
	dumpView(t, "ReadingState (120x40)", m)

	// 6. Small terminal
	m2 := captureModel(60, 15)
	dumpView(t, "SmallTerminal (60x15)", m2)

	// 7. Medium terminal
	m3 := captureModel(80, 24)
	m3.spreadType = "three_card"
	m3.layoutHeights()
	result3, _ := tools.DrawCards(m3.store, "three_card")
	m3.drawResult = result3
	m3.revealIndex = len(result3.Cards)
	m3.readingBuf.WriteString("测试解读内容")
	m3.readingVP.SetContent(m3.readingBuf.String())
	m3.state = &ReadingState{}
	dumpView(t, "ReadingState (80x24)", m3)

	t.Log("Output written to testdata/actual_output.txt")
}
