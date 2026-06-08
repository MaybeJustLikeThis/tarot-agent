package tui

import (
	"strings"

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
	if m.spinnerOn() {
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
		Width(w - 1).
		Height(h).
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
	return labelStyled + "\n" + m.input.View() + "\n" + statusBarStyle.Render("  回车提交 · q 退出")
}

// renderSpreadOptions renders spread selection UI.
func renderSpreadOptions(m *Model) string {
	var b strings.Builder
	b.WriteString(spreadNumberStyle.Render("  1") + "  " + spreadDescStyle.Render("单张牌 — 快速指引") + "\n")
	b.WriteString(spreadNumberStyle.Render("  2") + "  " + spreadDescStyle.Render("三张牌 — 过去 / 现在 / 未来") + "\n")
	b.WriteString(spreadNumberStyle.Render("  3") + "  " + spreadDescStyle.Render("凯尔特十字 — 深度全面分析") + "\n")
	b.WriteString(statusBarStyle.Render("  q 退出"))
	return b.String()
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
	case *SpreadState:
		return "选择牌阵"
	case *RevealState:
		return "翻牌中"
	case *ReadingState:
		return "解读中"
	case *FollowUpState:
		return "追问"
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
