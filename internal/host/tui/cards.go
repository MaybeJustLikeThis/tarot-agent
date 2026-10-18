package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/tarot-agent/internal/domain"
)

// Card dimensions
const (
	cardWidth  = 16
	cardHeight = 9
)

// renderCardBack renders a face-down card.
func renderCardBack(width int) string {
	style := lipgloss.NewStyle().
		Width(cardWidth).
		Height(cardHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorMuted)

	inner := lipgloss.NewStyle().
		Width(cardWidth-4).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(colorMuted)

	pattern := inner.Render("✦\n\n?\n\n✦")

	return style.Render(pattern)
}

// renderCardFront renders a face-up card with name and orientation.
func renderCardFront(card domain.DrawnCard) string {
	return renderCardFrontSized(card, cardWidth)
}

// renderCardFrontSized renders a card at a specific width.
// w is the TOTAL desired width including borders.
func renderCardFrontSized(card domain.DrawnCard, w int) string {
	return renderCardFrontSizedH(card, w, 0)
}

func cardBoxHeight(w, innerH int) int {
	if innerH > 0 {
		return innerH
	}
	if w < 14 {
		return 7
	}
	return cardHeight
}

// renderCardPlaceholder renders an empty placeholder for unrevealed cards.
func renderCardPlaceholder() string {
	return renderCardPlaceholderSized(cardWidth)
}

// renderCardPlaceholderSized renders a placeholder at a specific width.
func renderCardPlaceholderSized(w int) string {
	return renderCardPlaceholderSizedH(w, 0)
}

// renderSpreadLayout renders all cards in a spread layout, adapted to fit maxWidth.
func renderSpreadLayout(cards []domain.DrawnCard, revealed int, spreadType string, maxWidth ...int) string {
	w := 200 // default: no constraint
	if len(maxWidth) > 0 {
		w = maxWidth[0]
	}

	switch spreadType {
	case "single":
		return renderSingleLayout(cards, revealed, w)
	case "three_card":
		return renderThreeCardLayout(cards, revealed, w)
	case "celtic_cross":
		return renderCelticCrossLayout(cards, revealed, w)
	default:
		return renderSingleLayout(cards, revealed, w)
	}
}

// renderSingleLayout renders a single card centered.
func renderSingleLayout(cards []domain.DrawnCard, revealed int, maxW int) string {
	if revealed <= 0 {
		return renderCardPlaceholder()
	}
	return renderCardFront(cards[0])
}

// renderThreeCardLayout renders three cards, adapting width to fit.
func renderThreeCardLayout(cards []domain.DrawnCard, revealed int, maxW int) string {
	labels := []string{"过去", "现在", "未来"}

	// Calculate card width to fit: maxW = cardW * 3 + gaps
	gaps := 2 // 2 spaces between 3 cards
	cardW := (maxW - gaps) / 3
	if cardW < cardWidth {
		cardW = maxI(10, cardW) // minimum 10 chars wide
	}
	if cardW > cardWidth {
		cardW = cardWidth // don't exceed default
	}

	var rendered []string
	for i := 0; i < 3; i++ {
		var card string
		if i < revealed {
			card = renderCardFrontSized(cards[i], cardW)
		} else {
			card = renderCardPlaceholderSized(cardW)
		}

		labelStyle := lipgloss.NewStyle().
			Width(cardW).
			Align(lipgloss.Center).
			Foreground(colorAccent).
			Bold(true)

		if i < revealed {
			rendered = append(rendered, lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render(labels[i]), card))
		} else {
			rendered = append(rendered, lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render(" "), card))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// renderCelticCrossLayout renders 10 cards, adapting to width.
func renderCelticCrossLayout(cards []domain.DrawnCard, revealed int, maxW int) string {
	labels := []string{
		"现况", "挑战", "根基", "过去", "可能", "近况",
		"自我", "环境", "希望", "结果",
	}

	// Celtic cross needs a lot of width. Calculate cards per row.
	gaps := 2
	cardsPerRow := 6
	cardW := (maxW - gaps) / cardsPerRow
	if cardW < 10 {
		// Too narrow for 6 across, try 3
		cardsPerRow = 3
		cardW = (maxW - gaps) / cardsPerRow
	}
	if cardW < 10 {
		cardW = 10
	}
	if cardW > cardWidth {
		cardW = cardWidth
	}

	var rows []string

	// Split into rows
	for rowStart := 0; rowStart < len(cards) && rowStart < 10; rowStart += cardsPerRow {
		rowEnd := rowStart + cardsPerRow
		if rowEnd > 10 {
			rowEnd = 10
		}
		if rowEnd > len(cards) {
			rowEnd = len(cards)
		}

		var rowCards []string
		for i := rowStart; i < rowEnd; i++ {
			var card string
			if i < revealed {
				card = renderCardFrontSized(cards[i], cardW)
			} else {
				card = renderCardPlaceholderSized(cardW)
			}

			labelStyle := lipgloss.NewStyle().
				Width(cardW).
				Align(lipgloss.Center).
				Foreground(colorAccent).
				Bold(true)

			label := " "
			if i < revealed && i < len(labels) {
				label = labels[i]
			}
			rowCards = append(rowCards, lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render(label), card))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) spreadContentWidth() int {
	w := m.layout.LeftWidth - 4
	if w < 8 {
		w = 8
	}
	return w
}

// spreadContentHeight is the room left for the card block inside the left panel.
// usedLines counts the title and any status lines already placed above the cards.
func (m *Model) spreadContentHeight(usedLines int) int {
	h := m.layout.BodyHeight - usedLines
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) renderSpread(revealed, usedLines int) string {
	if m.drawResult == nil {
		return ""
	}
	return renderSpreadForPanel(m.drawResult.Cards, revealed, m.spreadType, m.spreadContentWidth(), m.spreadContentHeight(usedLines))
}

func historyQuestionWidth(m *Model, prefix string) int {
	// Left panel content is LeftWidth-1, and the list style pads one column on each side.
	avail := m.layout.LeftWidth - 3 - lipgloss.Width(prefix)
	if avail < 2 {
		return 2
	}
	return avail
}

// renderSpreadForPanel keeps a spread inside the left panel.
// Celtic cross drops to a text list when bordered cards would push the input off screen.
func renderSpreadForPanel(cards []domain.DrawnCard, revealed int, spreadType string, maxW, maxH int) string {
	if maxW < 1 {
		maxW = 1
	}
	if spreadType == "celtic_cross" {
		return renderCelticFitted(cards, revealed, maxW, maxH)
	}
	layout := renderSpreadLayout(cards, revealed, spreadType, maxW)
	if maxH > 0 && lipgloss.Height(layout) > maxH {
		return renderCardsCompact(cards, revealed, spreadType, maxW, maxH)
	}
	return layout
}

func renderCelticFitted(cards []domain.DrawnCard, revealed, maxW, maxH int) string {
	n := len(cards)
	if n > 10 {
		n = 10
	}
	if n == 0 {
		return ""
	}
	// Prefer fewer cards per row so Chinese names stay on one line,
	// as long as the block still fits in the panel.
	var best string
	bestW, bestInner := 0, 0
	for _, perRow := range []int{2, 3, 4, 5} {
		if perRow > n {
			continue
		}
		cardW := maxW / perRow
		if cardW > cardWidth {
			cardW = cardWidth
		}
		if cardW < 8 || cardW*perRow > maxW {
			continue
		}
		for _, inner := range []int{9, 7, 5, 3} {
			rendered := renderCelticRows(cards, revealed, cardW, perRow, inner)
			if !blockFits(rendered, maxW, maxH) {
				continue
			}
			if cardW > bestW || (cardW == bestW && inner > bestInner) {
				best = rendered
				bestW = cardW
				bestInner = inner
			}
		}
	}
	if best != "" {
		return best
	}
	return renderCardsCompact(cards, revealed, "celtic_cross", maxW, maxH)
}

func renderCelticRows(cards []domain.DrawnCard, revealed, cardW, perRow, innerH int) string {
	labels := []string{
		"现况", "挑战", "根基", "过去", "可能", "近况",
		"自我", "环境", "希望", "结果",
	}
	limit := len(cards)
	if limit > 10 {
		limit = 10
	}
	var rows []string
	for rowStart := 0; rowStart < limit; rowStart += perRow {
		rowEnd := rowStart + perRow
		if rowEnd > limit {
			rowEnd = limit
		}
		var rowCards []string
		for i := rowStart; i < rowEnd; i++ {
			var card string
			if i < revealed {
				card = renderCardFrontSizedH(cards[i], cardW, innerH)
			} else {
				card = renderCardPlaceholderSizedH(cardW, innerH)
			}
			labelStyle := lipgloss.NewStyle().
				Width(cardW).
				Align(lipgloss.Center).
				Foreground(colorAccent).
				Bold(true)
			label := " "
			if i < revealed && i < len(labels) {
				label = labels[i]
			}
			rowCards = append(rowCards, lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render(label), card))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderCardFrontSizedH(card domain.DrawnCard, w, innerH int) string {
	borderColor := colorPrimary
	orientSymbol := "↑"
	orientText := "正位"
	if card.Orientation == domain.Reversed {
		borderColor = colorSecondary
		orientSymbol = "↓"
		orientText = "逆位"
	}

	contentW := w - 2
	if contentW < 4 {
		contentW = 4
	}
	h := cardBoxHeight(w, innerH)

	style := lipgloss.NewStyle().
		Width(contentW).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(borderColor)

	innerW := contentW - 2
	if innerW < 2 {
		innerW = 2
	}
	nameStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true).
		Width(innerW).
		Align(lipgloss.Center)
	orientStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(innerW).
		Align(lipgloss.Center)
	symbolStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true).
		Width(innerW).
		Align(lipgloss.Center)

	maxNameLen := innerW
	if maxNameLen < 3 {
		maxNameLen = 3
	}
	content := lipgloss.JoinVertical(lipgloss.Center,
		symbolStyle.Render(orientSymbol),
		nameStyle.Render(truncate(card.Card.NameCN, maxNameLen)),
		orientStyle.Render(orientText),
		nameStyle.Render(truncate(card.Card.NameEN, maxNameLen)),
	)
	return style.Render(content)
}

func renderCardPlaceholderSizedH(w, innerH int) string {
	contentW := w - 2
	if contentW < 4 {
		contentW = 4
	}
	style := lipgloss.NewStyle().
		Width(contentW).
		Height(cardBoxHeight(w, innerH)).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"})
	return style.Render("")
}

func blockFits(rendered string, maxW, maxH int) bool {
	rendered = strings.TrimRight(rendered, "\n")
	if rendered == "" {
		return false
	}
	if lipgloss.Width(rendered) > maxW {
		return false
	}
	if maxH > 0 && lipgloss.Height(rendered) > maxH {
		return false
	}
	return true
}

func renderCardsCompact(cards []domain.DrawnCard, revealed int, spreadType string, maxW, maxH int) string {
	lines := compactCardLines(cards, revealed, spreadType, maxW)
	if maxH <= 0 || len(lines) <= maxH {
		return strings.Join(lines, "\n")
	}
	// One separating column, so each side is at most (maxW-1)/2.
	colW := (maxW - 1) / 2
	if colW >= 8 {
		colLines := compactCardLines(cards, revealed, spreadType, colW)
		var rows []string
		for i := 0; i < len(colLines); i += 2 {
			left := colLines[i]
			gap := colW - lipgloss.Width(left) + 1
			if gap < 1 {
				gap = 1
			}
			row := left + strings.Repeat(" ", gap)
			if i+1 < len(colLines) {
				row += colLines[i+1]
			}
			rows = append(rows, row)
		}
		if len(rows) <= maxH && lipgloss.Width(strings.Join(rows, "\n")) <= maxW {
			return strings.Join(rows, "\n")
		}
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	return strings.Join(lines, "\n")
}

func compactCardLines(cards []domain.DrawnCard, revealed int, spreadType string, maxW int) []string {
	labels := spreadLabels(spreadType)
	limit := len(cards)
	if limit > 10 {
		limit = 10
	}
	var lines []string
	for i := 0; i < limit; i++ {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		var line string
		if i < revealed && i < len(cards) {
			orient := "正"
			if cards[i].Orientation == domain.Reversed {
				orient = "逆"
			}
			name := cards[i].Card.NameCN
			if name == "" {
				name = cards[i].Card.ID
			}
			if name == "" {
				name = "未知牌"
			}
			if label == "" {
				line = fmt.Sprintf("%s %s", name, orient)
			} else {
				line = fmt.Sprintf("%s %s %s", label, name, orient)
			}
		} else if label != "" {
			line = label + " ·"
		} else {
			line = "·"
		}
		lines = append(lines, truncateWidth(line, maxW))
	}
	return lines
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// spreadLabels returns position labels for a spread type.
func spreadLabels(spreadType string) []string {
	switch spreadType {
	case "single":
		return []string{"指引"}
	case "three_card":
		return []string{"过去", "现在", "未来"}
	case "celtic_cross":
		return []string{"现况", "挑战", "根基", "过去", "可能", "近况", "自我", "环境", "希望", "结果"}
	default:
		return []string{}
	}
}

// formatDrawnCards formats drawn cards as text for the agent prompt.
func formatDrawnCards(cards []domain.DrawnCard, spreadType string) string {
	labels := spreadLabels(spreadType)
	var b strings.Builder
	for i, card := range cards {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		orient := "正位"
		if card.Orientation == domain.Reversed {
			orient = "逆位"
		}
		fmt.Fprintf(&b, "位置 %d（%s）：%s [%s] — %s — id: %s\n",
			i+1, label, card.Card.NameCN, orient, card.Card.NameEN, card.Card.ID)
	}
	return b.String()
}
