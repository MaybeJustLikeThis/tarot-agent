package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/tarot-agent/internal/tools"
)

const revealDelay = 600 * time.Millisecond

// StateView holds left/right panel content for a state.
type StateView struct {
	Left  string
	Right string
}

// State is the interface for TUI states.
type State interface {
	Update(m *Model, msg tea.Msg) (State, tea.Cmd)
	View(m *Model) StateView
}

// --- InputState ---

type InputState struct{}

func (s *InputState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "tab":
			// History: only when input is empty to avoid conflicting with typing
			if strings.TrimSpace(m.input.Value()) == "" {
				readings, err := m.store.Readings.List(20)
				if err != nil {
					m.err = err
					return s, nil
				}
				m.historyReadings = readings
				m.historyCursor = 0
				m.drawResult = nil
				m.resetReading()
				m.resetChat()
				if len(readings) > 0 {
					m.readingVP.SetContent(renderMarkdown(readings[0].Interpretation, m.readingVP.Width-2))
				}
				return &HistoryState{}, nil
			}
		case "ctrl+d":
			m.userInput = "今日指引"
			m.resetReading()
			m.resetChat()
			m.err = nil
			return startReveal(m, "single")
		case "enter":
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return s, nil
			}
			if isExitCmd(input) {
				return s, tea.Quit
			}
			m.userInput = input
			m.resetReading()
			m.resetChat()
			m.err = nil
			return &SpreadState{}, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return s, cmd
}

func (s *InputState) View(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("等待抽牌..."))
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		renderCentered(m.layout.BodyHeight-1, "")
	return StateView{Left: left, Right: right}
}

// --- SpreadState ---

type SpreadState struct{}

func (s *SpreadState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "esc", "backspace":
			return &InputState{}, nil
		case "1":
			return startReveal(m, "single")
		case "2":
			return startReveal(m, "three_card")
		case "3":
			return startReveal(m, "celtic_cross")
		case "q", "quit", "退出":
			return s, tea.Quit
		}
	}
	return s, nil
}

func (s *SpreadState) View(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("等待抽牌..."))
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		renderCentered(m.layout.BodyHeight-1, "")
	return StateView{Left: left, Right: right}
}

func startReveal(m *Model, spreadType string) (State, tea.Cmd) {
	result, err := tools.DrawCards(m.store, spreadType)
	if err != nil {
		m.err = err
		return &SpreadState{}, nil
	}

	m.spreadType = spreadType
	m.drawResult = result
	m.revealIndex = 0
	m.bridge.guard.Reset()
	m.bridge.guard.SetExpectedCards(int32(len(result.Cards)))
	m.resetReading()
	m.resetChat()
	m.toolCalls = nil
	m.err = nil

	return &RevealState{}, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return revealNextCardMsg{}
	})
}

// --- RevealState ---

type RevealState struct{}

type revealNextCardMsg struct{}

func (s *RevealState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	switch msg.(type) {
	case revealNextCardMsg:
		if m.drawResult != nil && m.revealIndex < len(m.drawResult.Cards) {
			m.revealIndex++
			if m.revealIndex >= len(m.drawResult.Cards) {
				return s, tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
					return revealCompleteMsg{}
				})
			}
			return s, tea.Tick(revealDelay, func(time.Time) tea.Msg {
				return revealNextCardMsg{}
			})
		}
	case revealCompleteMsg:
		return startInterpretation(m)
	}
	return s, nil
}

func (s *RevealState) View(m *Model) StateView {
	// Left: card reveal animation
	var content string
	if m.drawResult == nil {
		content = styleMuted.Render("  等待抽牌...")
	} else if m.revealIndex < len(m.drawResult.Cards) {
		sp := spinnerStyle.Render(m.spinner.View())
		status := fmt.Sprintf(" 正在翻牌... (%d/%d)", m.revealIndex, len(m.drawResult.Cards))
		content = sp + status + "\n\n"
		content += renderSpreadLayout(m.drawResult.Cards, m.revealIndex, m.spreadType, m.layout.LeftWidth-4)
	} else {
		content = styleSuccess.Render("  ✦ 牌已全部翻开") + "\n\n"
		content += renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
	}

	left := renderPanelTitle("牌面", colorAccent) + "\n" + content

	// Right: waiting
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("等待翻牌完成..."))

	return StateView{Left: left, Right: right}
}

func startInterpretation(m *Model) (State, tea.Cmd) {
	cardInfo := formatDrawnCards(m.drawResult.Cards, m.spreadType)
	prompt := fmt.Sprintf(
		"【用户描述】\n%s\n\n【牌阵】\n%s\n\n【已抽到的牌】\n%s\n\n请查询每张牌的含义，然后给出深度个性化解读。",
		m.userInput, m.spreadType, cardInfo,
	)

	m.bridge.setup(prompt)
	return &ReadingState{}, m.bridge.nextEvent()
}

type revealCompleteMsg struct{}

// --- ReadingState ---

type ReadingState struct{}

func (s *ReadingState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case agentDeltaMsg:
		m.readingBuf.WriteString(msg.text)
		m.readingVP.SetContent(renderMarkdown(m.readingBuf.String(), m.readingVP.Width-2))
		m.readingVP.GotoBottom()
		return s, m.bridge.nextEvent()

	case agentToolMsg:
		m.toolCalls = append(m.toolCalls, msg.name)
		return s, m.bridge.nextEvent()

	case agentEndMsg:
		m.bridge.cleanup()
		// Finalize: save reading content
		m.readingContent = m.readingBuf.String()
		m.readingBuf.Reset()
		m.readingVP.SetContent(m.renderReadingMarkdown())
		return &FollowUpState{}, nil

	case agentErrMsg:
		m.bridge.cleanup()
		m.readingContent = m.readingBuf.String()
		m.readingBuf.Reset()
		m.readingVP.SetContent(m.renderReadingMarkdown())
		m.err = msg.err
		return &FollowUpState{}, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "up", "k":
			m.readingVP.LineUp(1)
		case "down", "j":
			m.readingVP.LineDown(1)
		case "pgup":
			m.readingVP.HalfViewUp()
		case "pgdown":
			m.readingVP.HalfViewDown()
		}
	}

	return s, nil
}

func (s *ReadingState) View(m *Model) StateView {
	return renderReadingView(m)
}

// --- FollowUpState ---

type FollowUpState struct{}

func (s *FollowUpState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "up", "k":
			m.chatVP.LineUp(1)
			return s, nil
		case "down", "j":
			m.chatVP.LineDown(1)
			return s, nil
		case "pgup":
			m.chatVP.HalfViewUp()
			return s, nil
		case "pgdown":
			m.chatVP.HalfViewDown()
			return s, nil
		case "enter":
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				m.bridge.clearMessages()
				m.input.Reset()
				m.resetReading()
				m.resetChat()
				return &InputState{}, nil
			}
			if isExitCmd(input) {
				return s, tea.Quit
			}
			// Add user message to chat and start new AI response
			m.appendChatUserMessage(input)
			m.chatStreamBuf.Reset()
			m.chatVP.SetContent(m.renderChatConversation(false))
			m.chatVP.GotoBottom()
			m.input.Reset()
			m.toolCalls = nil
			m.err = nil
			m.bridge.setup(input)
			return &ChatState{}, m.bridge.nextEvent()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return s, cmd
}

func (s *FollowUpState) View(m *Model) StateView {
	return renderSplitView(m)
}

// --- ChatState ---
// ChatState streams AI responses for follow-up questions into the chat viewport.

type ChatState struct{}

func (s *ChatState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case agentDeltaMsg:
		m.chatStreamBuf.WriteString(msg.text)
		m.chatVP.SetContent(m.renderChatConversation(true))
		m.chatVP.GotoBottom()
		return s, m.bridge.nextEvent()

	case agentToolMsg:
		m.toolCalls = append(m.toolCalls, msg.name)
		return s, m.bridge.nextEvent()

	case agentEndMsg:
		m.bridge.cleanup()
		m.finalizeChatStream()
		m.chatVP.SetContent(m.renderChatConversation(false))
		return &FollowUpState{}, nil

	case agentErrMsg:
		m.bridge.cleanup()
		m.finalizeChatStream()
		m.chatVP.SetContent(m.renderChatConversation(false))
		m.err = msg.err
		return &FollowUpState{}, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "up", "k":
			m.chatVP.LineUp(1)
		case "down", "j":
			m.chatVP.LineDown(1)
		case "pgup":
			m.chatVP.HalfViewUp()
		case "pgdown":
			m.chatVP.HalfViewDown()
		}
	}

	return s, nil
}

func (s *ChatState) View(m *Model) StateView {
	return renderSplitView(m)
}

// --- HistoryState ---

type HistoryState struct{}

func (s *HistoryState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "esc", "backspace":
			return &InputState{}, nil
		case "up", "k":
			if m.historyCursor > 0 {
				m.historyCursor--
				s.updateViewport(m)
			}
			return s, nil
		case "down", "j":
			if m.historyCursor < len(m.historyReadings)-1 {
				m.historyCursor++
				s.updateViewport(m)
			}
			return s, nil
		case "pgup":
			m.readingVP.HalfViewUp()
			return s, nil
		case "pgdown":
			m.readingVP.HalfViewDown()
			return s, nil
		}
	}
	return s, nil
}

func (s *HistoryState) updateViewport(m *Model) {
	if m.historyCursor >= 0 && m.historyCursor < len(m.historyReadings) {
		r := m.historyReadings[m.historyCursor]
		m.readingVP.SetContent(renderMarkdown(r.Interpretation, m.readingVP.Width-2))
	}
}

func (s *HistoryState) View(m *Model) StateView {
	// Left: history list
	left := renderPanelTitle("历史记录", colorAccent) + "\n"

	if len(m.historyReadings) == 0 {
		left += renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("暂无占卜记录"))
	} else {
		listStyle := lipgloss.NewStyle().Padding(0, 1)
		var items string
		for i, r := range m.historyReadings {
			cursor := "  "
			if i == m.historyCursor {
				cursor = styleSuccess.Render("▸ ")
			}
			date := r.CreatedAt.Format("01/02 15:04")
			spread := spreadDisplayName(r.SpreadID)
			question := truncate(r.Question, 20)
			line := fmt.Sprintf("%s%s %s — %s",
				cursor,
				styleSubtle.Render(date),
				styleMuted.Render(spread),
				question)
			items += line + "\n"
		}
		left += listStyle.Render(items)
	}

	// Right: selected reading
	scrollHint := styleSubtle.Render("  ↑↓/jk 滚动 · esc 返回")
	right := renderPanelTitle("解读详情", colorPrimary) + "\n" +
		m.readingVP.View() + "\n" + scrollHint

	return StateView{Left: left, Right: right}
}

// --- helpers ---

func isExitCmd(s string) bool {
	switch strings.ToLower(s) {
	case "q", "quit", "exit", "退出":
		return true
	}
	return false
}

// renderReadingView renders the reading-only view (ReadingState, RevealState).
// Reading viewport takes the full right panel height.
func renderReadingView(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n"
	if m.drawResult != nil {
		left += renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
	} else {
		left += renderCentered(m.layout.BodyHeight-1, styleMuted.Render("等待抽牌..."))
	}
	scrollHint := styleSubtle.Render("  ↑↓/jk 滚动")
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		m.readingVP.View() + "\n" + scrollHint
	return StateView{Left: left, Right: right}
}

// renderSplitView renders the split view (FollowUpState, ChatState).
// Reading viewport on top, chat viewport on bottom.
func renderSplitView(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n"
	if m.drawResult != nil {
		left += renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
	} else {
		left += renderCentered(m.layout.BodyHeight-1, styleMuted.Render("等待抽牌..."))
	}

	// Right panel: reading (top) + separator + chat (bottom)
	readingPart := renderPanelTitle("解读", colorPrimary) + "\n" +
		m.readingVP.View()

	chatContent := m.chatVP.View()
	if chatContent == "" || len(m.chatMessages) == 0 {
		chatContent = styleMuted.Italic(true).Render("  解读后可以继续追问...")
	}
	chatHint := styleSubtle.Render("  ↑↓/jk 滚动")
	chatPart := renderPanelTitle("对话", colorAccent) + "\n" +
		chatContent + "\n" + chatHint

	chatSep := separatorStyle.Render(strings.Repeat("─", m.layout.RightWidth-2))
	right := readingPart + "\n" + chatSep + "\n" + chatPart

	return StateView{Left: left, Right: right}
}
