package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
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
	if len(r.FollowUps) > 0 {
		b.WriteString("\n**追问**\n\n")
		for _, turn := range r.FollowUps {
			who := "助手"
			if turn.Role == "user" {
				who = "你"
			}
			fmt.Fprintf(&b, "**%s**：%s\n\n", who, strings.TrimSpace(turn.Content))
		}
	}
	return b.String()
}

func historyStatusLabel(m *Model) string {
	if r, ok := selectedReading(m); ok && len(r.Cards) == 0 {
		return "旧记录无牌面，无法继续 · esc 返回"
	}
	return "↑↓ 选择 · enter 继续 · esc 返回"
}

func historyScrollHint(m *Model) string {
	if r, ok := selectedReading(m); ok && len(r.Cards) == 0 {
		return "  ↑↓/jk 滚动 · 旧记录无牌面，无法继续"
	}
	return "  ↑↓/jk 滚动 · enter 继续 · esc 返回"
}

func selectedReading(m *Model) (domain.Reading, bool) {
	if m.historyCursor < 0 || m.historyCursor >= len(m.historyReadings) {
		return domain.Reading{}, false
	}
	return m.historyReadings[m.historyCursor], true
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
// Follow-up turns update that same record via persistFollowUps.
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
	m.readingID = reading.ID
}

// persistFollowUps writes the current chat onto the reading saved for this draw.
// A missing id does not create a second record. A write error stays on m.err
// and leaves the conversation in place.
func (m *Model) persistFollowUps() {
	if m.readingID == "" || m.store == nil || m.store.Readings == nil {
		return
	}
	if !chatHasAssistant(m.chatMessages) {
		return
	}
	turns := followUpsFromChat(m.chatMessages)
	if err := m.store.Readings.UpdateFollowUps(m.readingID, turns); err != nil {
		m.err = fmt.Errorf("保存追问: %w", err)
	}
}

func chatHasAssistant(messages []ChatMessage) bool {
	for _, msg := range messages {
		if msg.Role == "assistant" && strings.TrimSpace(msg.Content) != "" {
			return true
		}
	}
	return false
}

func followUpsFromChat(messages []ChatMessage) []domain.FollowUpTurn {
	turns := make([]domain.FollowUpTurn, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		turns = append(turns, domain.FollowUpTurn{
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}
	return turns
}

// resumeReading opens a stored reading as a follow-up chat.
// Later turns update this id. The agent is cleared so the next question
// carries the saved interpretation.
func (m *Model) resumeReading(r domain.Reading) {
	m.userInput = r.Question
	m.spreadType = r.SpreadID
	m.drawResult = &domain.DrawResult{
		SpreadID: r.SpreadID,
		Cards:    hydrateCards(m, r.Cards),
	}
	m.readingContent = r.Interpretation
	m.readingBuf.Reset()
	m.readingVP.SetContent(m.renderReadingMarkdown())
	m.readingVP.GotoTop()
	m.chatMessages = chatFromFollowUps(r.FollowUps)
	m.chatStreamBuf.Reset()
	if len(m.chatMessages) > 0 {
		m.chatVP.SetContent(m.renderChatConversation(false))
		m.chatVP.GotoBottom()
	} else {
		m.chatVP.SetContent("")
	}
	m.focusZone = "reading"
	m.historyWritten = true
	m.readingID = r.ID
	m.followUpNeedsContext = true
	m.toolCalls = nil
	m.err = nil
	m.retryNote = ""
	m.input.Reset()
	if m.bridge != nil {
		m.bridge.clearMessages()
		if m.bridge.guard != nil {
			m.bridge.guard.AllowFollowUp()
		}
	}
}

func chatFromFollowUps(turns []domain.FollowUpTurn) []ChatMessage {
	if len(turns) == 0 {
		return nil
	}
	msgs := make([]ChatMessage, 0, len(turns))
	for _, turn := range turns {
		msgs = append(msgs, ChatMessage{
			Role:      turn.Role,
			Content:   turn.Content,
			CreatedAt: turn.CreatedAt,
		})
	}
	return msgs
}

func hydrateCards(m *Model, cards []domain.DrawnCard) []domain.DrawnCard {
	if len(cards) == 0 {
		return nil
	}
	out := make([]domain.DrawnCard, len(cards))
	for i, card := range cards {
		full := card.Card
		if m.store != nil && m.store.Cards != nil && card.Card.ID != "" {
			if looked, err := m.store.Cards.GetByID(card.Card.ID); err == nil {
				full = looked
			}
		}
		out[i] = domain.DrawnCard{
			Card:        full,
			Orientation: card.Orientation,
			Position:    card.Position,
		}
	}
	return out
}

// nextFollowUpPrompt returns the text sent to the agent.
// A resumed history chat includes the saved reading once; a live follow-up
// sends only the new question, because that session already has the reading.
func (m *Model) nextFollowUpPrompt(question string) string {
	if !m.followUpNeedsContext {
		return question
	}
	m.followUpNeedsContext = false
	return buildFollowUpContext(m.userInput, m.spreadType, m.drawResult, m.readingContent, m.chatMessages, question)
}

func buildFollowUpContext(question, spreadID string, draw *domain.DrawResult, interpretation string, messages []ChatMessage, followUp string) string {
	var b strings.Builder
	b.WriteString("这是对下面这次已完成占卜的追问。不要重新抽牌，直接基于原解读和已有对话回答。\n\n")
	fmt.Fprintf(&b, "【问题】\n%s\n\n", strings.TrimSpace(question))
	fmt.Fprintf(&b, "【牌阵】\n%s\n\n", displayOr(spreadLabel(spreadID), "未记录"))
	if draw != nil && len(draw.Cards) > 0 {
		b.WriteString("【牌面】\n")
		for _, card := range draw.Cards {
			b.WriteString(formatHistoryCard(card))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("【原解读】\n")
	text := strings.TrimSpace(interpretation)
	if text == "" {
		text = "（无）"
	}
	b.WriteString(text)
	b.WriteString("\n\n")

	prior := messages
	if n := len(prior); n > 0 && prior[n-1].Role == "user" && strings.TrimSpace(prior[n-1].Content) == strings.TrimSpace(followUp) {
		prior = prior[:n-1]
	}
	if len(prior) > 0 {
		b.WriteString("【已有追问】\n")
		for _, msg := range prior {
			who := "助手"
			if msg.Role == "user" {
				who = "你"
			}
			fmt.Fprintf(&b, "%s：%s\n\n", who, strings.TrimSpace(msg.Content))
		}
	}
	fmt.Fprintf(&b, "【新的追问】\n%s", strings.TrimSpace(followUp))
	return b.String()
}
