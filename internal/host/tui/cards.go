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
		Width(cardWidth - 4).
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
	borderColor := colorPrimary
	orientSymbol := "↑"
	orientText := "正位"
	if card.Orientation == domain.Reversed {
		borderColor = colorSecondary
		orientSymbol = "↓"
		orientText = "逆位"
	}

	// Content width = total width - 2 (double border left+right)
	contentW := w - 2
	if contentW < 4 {
		contentW = 4
	}

	h := cardHeight
	if w < 14 {
		h = 7
	}

	style := lipgloss.NewStyle().
		Width(contentW).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(borderColor)

	innerW := contentW - 2 // padding
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

// renderCardPlaceholder renders an empty placeholder for unrevealed cards.
func renderCardPlaceholder() string {
	return renderCardPlaceholderSized(cardWidth)
}

// renderCardPlaceholderSized renders a placeholder at a specific width.
func renderCardPlaceholderSized(w int) string {
	contentW := w - 2
	if contentW < 4 {
		contentW = 4
	}

	style := lipgloss.NewStyle().
		Width(contentW).
		Height(cardHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"})

	return style.Render("")
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
		fmt.Fprintf(&b, "位置 %d（%s）：%s [%s] — %s\n",
			i+1, label, card.Card.NameCN, orient, card.Card.NameEN)
	}
	return b.String()
}
