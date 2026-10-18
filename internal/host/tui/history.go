package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/voocel/tarot-agent/internal/domain"
)

// renderHistoryDetail formats a stored reading for the history viewport.
// Records written before cards and model were persisted still render.
func renderHistoryDetail(r domain.Reading) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**问题**：%s\n\n", displayOr(r.Question, "（无）"))
	fmt.Fprintf(&b, "**牌阵**：%s\n\n", displayOr(spreadLabel(r.SpreadID), "未记录"))
	fmt.Fprintf(&b, "**模型**：%s\n\n", displayOr(r.Model, "未记录"))

	b.WriteString("**牌面**\n\n")
	if len(r.Cards) == 0 {
		b.WriteString("（旧记录未保存牌面）\n\n")
	} else {
		for i, card := range r.Cards {
			fmt.Fprintf(&b, "%d. %s\n", i+1, formatHistoryCard(card))
		}
		b.WriteString("\n")
	}

	b.WriteString("**解读**\n\n")
	text := strings.TrimSpace(r.Interpretation)
	if text == "" {
		b.WriteString("（无解读正文）\n")
	} else {
		b.WriteString(text)
		b.WriteString("\n")
	}
	return b.String()
}

func spreadLabel(spreadID string) string {
	if name := spreadDisplayName(spreadID); name != "" {
		return name
	}
	return spreadID
}

func formatHistoryCard(card domain.DrawnCard) string {
	name := card.Card.NameCN
	if name == "" {
		name = card.Card.ID
	}
	if name == "" {
		name = "未知牌"
	}
	var orient string
	switch card.Orientation {
	case domain.Reversed:
		orient = "逆位"
	case domain.Upright:
		orient = "正位"
	default:
		if card.Orientation == "" {
			orient = "正逆位未记录"
		} else {
			orient = string(card.Orientation)
		}
	}
	position := card.Position.NameCN
	if position == "" {
		position = "位置未记录"
	}
	line := fmt.Sprintf("%s — %s [%s]", position, name, orient)
	if card.Card.NameEN != "" {
		line += " — " + card.Card.NameEN
	}
	if card.Card.ID != "" {
		line += " — id: " + card.Card.ID
	}
	return line
}

func displayOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func snapshotCards(cards []domain.DrawnCard) []domain.DrawnCard {
	if len(cards) == 0 {
		return nil
	}
	out := make([]domain.DrawnCard, len(cards))
	for i, card := range cards {
		out[i] = domain.DrawnCard{
			Card: domain.Card{
				ID:     card.Card.ID,
				NameCN: card.Card.NameCN,
				NameEN: card.Card.NameEN,
			},
			Orientation: card.Orientation,
			Position:    card.Position,
		}
	}
	return out
}

// finishReading snapshots the streamed interpretation and stores it once.
// agentErr is the model failure, if this reading ended in error.
func (m *Model) finishReading(agentErr error) {
	m.bumpStreamGen()
	m.retryNote = ""
	m.readingContent = m.readingBuf.String()
	m.readingBuf.Reset()
	m.persistCurrentReading()
	m.readingVP.SetContent(m.renderReadingMarkdown())
	if agentErr == nil {
		return
	}
	if m.err != nil {
		m.err = fmt.Errorf("%v；%w", agentErr, m.err)
		return
	}
	m.err = agentErr
}

// persistCurrentReading writes the reading the user just saw, once per draw.
// Follow-up turns do not call this.
func (m *Model) persistCurrentReading() {
	if m.historyWritten || m.store == nil || m.store.Readings == nil {
		return
	}
	text := m.readingContent
	if m.drawResult == nil && strings.TrimSpace(text) == "" && strings.TrimSpace(m.userInput) == "" {
		return
	}

	spreadID := m.spreadType
	var cards []domain.DrawnCard
	if m.drawResult != nil {
		if m.drawResult.SpreadID != "" {
			spreadID = m.drawResult.SpreadID
		}
		cards = snapshotCards(m.drawResult.Cards)
	}

	reading := &domain.Reading{
		ID:             fmt.Sprintf("r_%d", time.Now().UnixNano()),
		Question:       m.userInput,
		SpreadID:       spreadID,
		Cards:          cards,
		Interpretation: text,
		Model:          m.llmModel,
		CreatedAt:      time.Now(),
	}
	if err := m.store.Readings.Append(reading); err != nil {
		m.err = fmt.Errorf("保存占卜记录: %w", err)
		return
	}
	m.historyWritten = true
}
