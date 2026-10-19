package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// layoutAuditSizes are the terminals we render by hand. Below 80×20 the
// program shows a size warning instead of the full layout.
func layoutAuditSizes() [][2]int {
	return [][2]int{{80, 24}, {120, 40}, {100, 30}, {80, 20}}
}

func TestLayoutAudit_LineWidthAndVisibility(t *testing.T) {
	for _, sz := range layoutAuditSizes() {
		w, h := sz[0], sz[1]
		for _, spread := range []string{"single", "three_card", "celtic_cross"} {
			for _, name := range []string{"input", "spread", "reveal", "reading", "followup", "chat", "history", "detail"} {
				m := auditModel(t, w, h, spread, name)
				view := m.View()
				label := fmt.Sprintf("%dx%d %s %s", w, h, spread, name)
				assertLineWidths(t, label, view, w)
				assertFitsScreen(t, label, view, h)
				if name == "reading" || name == "reveal" || name == "followup" {
					assertCardsVisible(t, label, view, spread, name)
				}
				if spread == "three_card" && h >= 24 && (name == "reading" || name == "followup") && !strings.Contains(view, "╔") {
					t.Errorf("%s: three-card layout lost card frames", label)
				}
				if hint := bottomHintFor(name); hint != "" && !strings.Contains(view, hint) {
					t.Errorf("%s: bottom hint %q is off-screen", label, hint)
				}
				if name == "history" {
					assertHistoryQuestionOneLine(t, label, view)
				}
			}
		}
	}
}

func TestLayoutAudit_NarrowTerminalWarning(t *testing.T) {
	for _, sz := range [][2]int{{60, 24}, {40, 12}, {79, 20}} {
		m := newTestModelSize(sz[0], sz[1])
		view := m.View()
		if !strings.Contains(view, "终端窗口太小") {
			t.Errorf("%dx%d: expected size warning, got %q", sz[0], sz[1], view)
		}
	}
}

func TestLayoutAudit_ResizeReflowsReading(t *testing.T) {
	m := newTestModelSize(120, 40)
	m.state = &FollowUpState{}
	m.readingContent = strings.Repeat("星币八属于土元素，今天要把事情落到具体行动上。", 8)
	m.layoutHeights()
	m.readingVP.SetContent(m.renderReadingMarkdown())
	tail := "落到具体行动上。"

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	mm := updated.(*Model)
	view := mm.readingVP.View()
	if !strings.Contains(view, tail) {
		t.Errorf("resized reading viewport dropped the paragraph tail:\n%s", view)
	}
	assertLineWidths(t, "resized-reading", mm.View(), 80)
}

func auditModel(t *testing.T, w, h int, spread, stateName string) *Model {
	t.Helper()
	m := newTestModelSize(w, h)
	m.mode = "专业模式"
	m.llmModel = "deepseek-chat"
	m.userInput = "我和伴侣最近总是因为钱和未来规划吵架，接下来三个月该怎么走？"
	m.spreadType = spread

	result, err := tools.DrawCards(m.store, spread)
	if err != nil {
		t.Fatalf("draw %s: %v", spread, err)
	}
	m.drawResult = result
	m.revealIndex = len(result.Cards)

	long := strings.Repeat("这是一段很长的中文解读，用来检查折行、宽字符和视口是否把内容推出屏幕。", 6)
	switch stateName {
	case "input":
		m.drawResult = nil
		m.spreadType = ""
		m.state = &InputState{}
	case "spread":
		m.drawResult = nil
		m.state = &SpreadState{}
	case "reveal":
		m.revealIndex = 1
		if m.revealIndex > len(result.Cards) {
			m.revealIndex = len(result.Cards)
		}
		m.state = &RevealState{}
	case "reading":
		m.readingBuf.WriteString(long)
		m.state = &ReadingState{}
		m.layoutHeights()
		m.readingVP.SetContent(wrapPlain(m.readingBuf.String(), viewportWrapWidth(m.readingVP.Width)))
	case "followup":
		m.readingContent = long
		m.chatMessages = []ChatMessage{{Role: "user", Content: "那我这个月具体可以做哪三件小事？"}}
		m.state = &FollowUpState{}
		m.layoutHeights()
		m.readingVP.SetContent(m.renderReadingMarkdown())
		m.chatVP.SetContent(m.renderChatConversation(false))
	case "chat":
		m.readingContent = long
		m.chatMessages = []ChatMessage{{Role: "user", Content: "那我这个月具体可以做哪三件小事？"}}
		m.chatStreamBuf.WriteString(strings.Repeat("先把这个月的开支写下来，再约一次不谈对错的谈话。", 4))
		m.state = &ChatState{}
		m.layoutHeights()
		m.readingVP.SetContent(m.renderReadingMarkdown())
		m.chatVP.SetContent(wrapPlain(m.chatTranscript(true), viewportWrapWidth(m.chatVP.Width)))
	case "detail":
		m.state = &CardDetailState{}
	case "history":
		m.drawResult = nil
		m.historyReadings = []domain.Reading{
			{
				Question:       strings.Repeat("事业与感情的长期问题", 4),
				SpreadID:       spread,
				Model:          "deepseek-chat",
				Interpretation: long,
				CreatedAt:      time.Date(2026, 10, 18, 15, 4, 0, 0, time.UTC),
				Cards:          result.Cards,
			},
		}
		m.historyCursor = 0
		m.state = &HistoryState{}
		m.layoutHeights()
		m.readingVP.SetContent(renderMarkdown(renderHistoryDetail(m.historyReadings[0]), m.readingVP.Width-2))
	default:
		t.Fatalf("unknown state %s", stateName)
	}
	m.layoutHeights()
	return m
}

func bottomHintFor(stateName string) string {
	switch stateName {
	case "input":
		return "输入 q 回车退出"
	case "spread":
		return "esc 返回 · q 退出"
	case "reading":
		return "ctrl+c 退出"
	case "followup":
		return "tab 切换焦点"
	case "chat":
		return "ctrl+c 中止并退出"
	case "history":
		return "esc 返回"
	case "detail":
		return "esc 返回"
	default:
		return ""
	}
}

func assertHistoryQuestionOneLine(t *testing.T, label, view string) {
	t.Helper()
	ok := false
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "15:04") && strings.Contains(line, "…") {
			ok = true
		}
	}
	if !ok {
		t.Errorf("%s: long history question should stay on the date line with an ellipsis", label)
	}
}

func assertFitsScreen(t *testing.T, label, view string, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	// A trailing newline produces one empty line past the last content row.
	n := len(lines)
	if n > 0 && lines[n-1] == "" {
		n--
	}
	if n > height {
		t.Errorf("%s: rendered %d rows > terminal %d (bottom is off-screen)", label, n, height)
	}
	visible := view
	if len(lines) > height {
		visible = strings.Join(lines[:height], "\n")
	}
	if !strings.Contains(visible, "星语") {
		t.Errorf("%s: status bar not in the visible window", label)
	}
}

func assertLineWidths(t *testing.T, label, view string, width int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		got := lipgloss.Width(line)
		if got > width {
			t.Errorf("%s line %d width %d > %d: %q", label, i, got, width, truncateString(line, 80))
		}
	}
}

func assertCardsVisible(t *testing.T, label, view, spread, stateName string) {
	t.Helper()
	var labels []string
	switch spread {
	case "single":
		return
	case "three_card":
		labels = []string{"过去", "现在", "未来"}
	case "celtic_cross":
		labels = []string{"现况", "挑战", "根基", "过去", "可能", "近况", "自我", "环境", "希望", "结果"}
	}
	if stateName == "reveal" && spread != "single" {
		// Only the first card is face up; later labels may be blank placeholders.
		if len(labels) > 1 {
			labels = labels[:1]
		}
	}
	for _, want := range labels {
		if !strings.Contains(view, want) {
			t.Errorf("%s: missing card label %q\n%s", label, want, truncateString(view, 400))
		}
	}
}
