package tui

import (
	"fmt"
	"strings"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

// renderStatusBar renders the top status bar.
func renderStatusBar(m *Model) string {
	w := m.layout.Width
	left := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("✦ 星语 Tarot Agent")

	modeTag := ""
	if m.mode != "" {
		modeTag = lipgloss.NewStyle().Foreground(colorAccent).Render("[" + m.mode + "]")
	}

	center := ""
	if name := spreadDisplayName(m.spreadType); name != "" {
		center = lipgloss.NewStyle().Foreground(colorMuted).Render("牌阵: " + name)
	}

	right := ""
	stateName := stateDisplayName(m.state)
	if m.retryNote != "" {
		right = lipgloss.NewStyle().Foreground(colorAccent).Render("正在重试")
	} else if m.spinnerOn() {
		right = spinnerStyle.Render(m.spinner.View()) + " " + lipgloss.NewStyle().Foreground(colorAccent).Render(stateName)
	} else if stateName != "" {
		right = lipgloss.NewStyle().Foreground(colorSubtle).Render(stateName)
	}

	leftW := lipgloss.Width(left)
	modeW := lipgloss.Width(modeTag)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)

	innerW := w - 2
	gap1 := maxI(2, (innerW-leftW-modeW-centerW-rightW)/3)
	gap2 := maxI(1, (innerW-leftW-modeW-gap1-centerW-rightW)/2)
	gap3 := maxI(1, innerW-leftW-modeW-gap1-gap2-centerW-rightW)

	bar := " " + left + strings.Repeat(" ", gap1) + modeTag + strings.Repeat(" ", gap2) + center + strings.Repeat(" ", gap3) + right

	return lipgloss.NewStyle().
		Width(w).
		MaxWidth(w).
		Background(lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}).
		Render(bar)
}

// renderLeftPanel renders the left panel (cards) with right border as separator.
func renderLeftPanel(m *Model, content string) string {
	w := m.layout.LeftWidth
	h := m.layout.BodyHeight

	// Content width = total width - 1 (right border)
	panel := lipgloss.NewStyle().
		Width(w-1).
		Height(h).
		MaxWidth(w).
		MaxHeight(h).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(colorBorder).
		Render(content)

	return panel
}

// renderRightPanel renders the right panel (reading).
func renderRightPanel(m *Model, content string) string {
	w := m.layout.RightWidth
	h := m.layout.BodyHeight

	panel := lipgloss.NewStyle().
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h).
		Render(content)

	return panel
}

// renderPanelTitle renders a panel title with a separator line.
func renderPanelTitle(title string, color lipgloss.TerminalColor) string {
	styled := lipgloss.NewStyle().Foreground(color).Bold(true).Render("▍" + title)
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 20))
	return styled + " " + sep
}

// renderCentered renders content centered in a given height.
func renderCentered(height int, content string) string {
	return lipgloss.NewStyle().Height(height).Align(lipgloss.Center, lipgloss.Center).Render(content)
}

// renderSeparator renders a horizontal line.
func renderSeparator(w int) string {
	return separatorStyle.Render(strings.Repeat("─", w-2))
}

// renderInputZone renders the bottom input area with label.
func renderInputZone(m *Model, label string) string {
	labelStyled := inputLabelStyle.Render("  " + label)
	return labelStyled + "\n" + m.input.View() + "\n" + statusBarStyle.Render("  "+inputHint(m))
}

// inputHint describes the keys that actually work in the current state.
func inputHint(m *Model) string {
	switch m.state.(type) {
	case *RevealState:
		return "翻牌中，请稍候"
	case *ReadingState:
		return "↑↓/jk 滚动 · ctrl+o 牌义 · ctrl+c 退出"
	case *FollowUpState:
		return "回车发送追问 · tab 切换焦点 · ctrl+o 牌义 · 输入 q 回车退出"
	case *ChatState:
		return "tab 切换焦点 · ctrl+o 牌义 · ctrl+c 中止并退出"
	case *CardDetailState:
		return "←→/hl 换牌 · esc 返回解读"
	case *CrisisState:
		return "esc 返回，先不抽牌"
	default:
		return "回车提交 · 空输入 tab 历史 · ctrl+d 每日一牌 · card 查牌 · 输入 q 回车退出"
	}
}

// renderQuestionGuideOptions is the accept / keep choice under the suggestion.
func renderQuestionGuideOptions(m *Model) string {
	var b strings.Builder
	if m.questionAdvice != nil {
		b.WriteString(spreadDescStyle.Render("  回车  采纳改写") + "\n")
		b.WriteString(spreadDescStyle.Render("  n     保留原问题") + "\n")
		b.WriteString(statusBarStyle.Render("  esc 返回修改"))
		return b.String()
	}
	b.WriteString(spreadDescStyle.Render("  回车  继续选牌阵") + "\n")
	b.WriteString(statusBarStyle.Render("  esc 返回修改"))
	return b.String()
}

// renderSpreadOptions renders spread selection UI.
// The recommended row is marked. Keys 1/2/3 are unchanged.
func renderSpreadOptions(m *Model) string {
	var b strings.Builder
	b.WriteString(spreadOptionLine("1", "单张牌 — 快速指引", "single", m.spreadAdvice))
	b.WriteString(spreadOptionLine("2", "三张牌 — 过去 / 现在 / 未来", "three_card", m.spreadAdvice))
	b.WriteString(spreadOptionLine("3", "凯尔特十字 — 深度全面分析", "celtic_cross", m.spreadAdvice))
	if m.spreadAdvice.Reason != "" {
		b.WriteString(styleMuted.Render("  "+m.spreadAdvice.Reason) + "\n")
	}
	if m.priorMatch != nil {
		b.WriteString(styleMuted.Render(fitLines("  "+priorSpreadHint(m.priorMatch), m.layout.Width-2, 2)))
		b.WriteString("\n")
	}
	if m.boundary.Kind == boundaryHealthLight {
		b.WriteString(styleMuted.Render(fitLines("  "+m.boundary.Note, m.layout.Width-2, 1)))
		b.WriteString("\n")
	}
	b.WriteString(statusBarStyle.Render("  esc 返回 · q 退出"))
	return b.String()
}

func priorSpreadHint(match *domain.PriorMatch) string {
	if match == nil {
		return ""
	}
	date := "日期未记录"
	if !match.Reading.CreatedAt.IsZero() {
		date = match.Reading.CreatedAt.Format("2006-01-02")
	}
	return fmt.Sprintf("上次（%s）问过类似问题，抽到%s。", date, domain.BriefCards(match.Reading.Cards, 3))
}

// fitLines wraps text and keeps at most lines rows.
func fitLines(s string, width, lines int) string {
	if lines < 1 {
		lines = 1
	}
	if width < 4 {
		width = 4
	}
	parts := strings.Split(wrapPlain(s, width), "\n")
	if len(parts) <= lines {
		return strings.Join(parts, "\n")
	}
	kept := append([]string{}, parts[:lines-1]...)
	kept = append(kept, truncateWidth(parts[lines-1], width))
	return strings.Join(kept, "\n")
}

func spreadOptionLine(num, desc, id string, advice SpreadAdvice) string {
	descStyle := spreadDescStyle
	mark := ""
	if advice.SpreadID == id {
		descStyle = styleSuccess
		mark = styleSuccess.Render("  ← 推荐")
	}
	return spreadNumberStyle.Render("  "+num) + "  " + descStyle.Render(desc) + mark + "\n"
}

// --- helpers ---

func spreadDisplayName(spreadType string) string {
	switch spreadType {
	case "single":
		return "单张牌"
	case "three_card":
		return "三张牌"
	case "celtic_cross":
		return "凯尔特十字"
	default:
		return ""
	}
}

func stateDisplayName(s State) string {
	switch s.(type) {
	case *InputState:
		return "等待输入"
	case *QuestionGuideState:
		return "确认问题"
	case *CardDetailState:
		return "牌义"
	case *CrisisState:
		return "关怀"
	case *SpreadState:
		return "选择牌阵"
	case *RevealState:
		return "翻牌中"
	case *ReadingState:
		return "解读中"
	case *FollowUpState:
		return "追问"
	case *ChatState:
		return "对话中"
	case *HistoryState:
		return "历史记录"
	default:
		return ""
	}
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
