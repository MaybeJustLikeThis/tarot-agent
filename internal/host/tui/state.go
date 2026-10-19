package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
					m.readingVP.SetContent(renderMarkdown(renderHistoryDetail(readings[0]), m.readingVP.Width-2))
				}
				return &HistoryState{}, nil
			}
		case "?":
			// Replay the first-run guide. A question can still contain "?".
			if strings.TrimSpace(m.input.Value()) == "" {
				return &WelcomeState{}, nil
			}
		case "ctrl+d":
			m.userInput = domain.DailyQuestion
			m.questionAdvice = nil
			m.boundary = Boundary{}
			m.priorMatch = nil
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
			m.priorMatch = nil
			m.boundary = ClassifyQuestion(input)
			if m.boundary.Kind == boundaryCrisis {
				m.questionAdvice = nil
				return &CrisisState{}, nil
			}
			advice, hasAdvice := AdviseQuestion(input)
			if hasAdvice {
				m.questionAdvice = &advice
			} else {
				m.questionAdvice = nil
			}
			if hasAdvice || m.boundary.Kind == boundaryHealth || m.boundary.Kind == boundaryLegal {
				return &QuestionGuideState{}, nil
			}
			return enterSpread(m), nil
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
	note := "当前是" + modeOrDefault(m.mode) + "。深入版会多讲元素和数字，简洁版用大白话、更短。"
	width := maxI(8, m.layout.RightWidth-2)
	right := renderPanelTitle("解读", colorPrimary) + "\n" + wrapPlain(note, width)
	if m.updateNotice != "" {
		right += "\n\n" + styleMuted.Render(fitLines(m.updateNotice, width, 2))
	}
	return StateView{Left: left, Right: right}
}

func modeOrDefault(mode string) string {
	if mode == "" {
		return "深入版"
	}
	return mode
}

// --- QuestionGuideState ---

// QuestionGuideState offers an optional rewrite. Enter accepts it, n keeps
// the original, esc returns to the editor. Nothing here starts a reading.
type QuestionGuideState struct{}

func (s *QuestionGuideState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch key.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "enter":
		if m.questionAdvice != nil {
			m.userInput = m.questionAdvice.Suggested
			m.input.SetValue(m.questionAdvice.Suggested)
		}
		return enterSpread(m), nil
	case "n", "N":
		return enterSpread(m), nil
	case "esc":
		return &InputState{}, nil
	default:
		return s, nil
	}
}

func (s *QuestionGuideState) View(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("等待抽牌..."))
	width := m.layout.RightWidth - 2
	if width < 10 {
		width = 10
	}
	var b strings.Builder
	b.WriteString(renderPanelTitle("问题", colorPrimary) + "\n")
	if m.boundary.Note != "" && m.boundary.Kind != boundaryHealthLight {
		b.WriteString(wrapPlain(m.boundary.Note, width) + "\n\n")
	}
	if m.questionAdvice != nil {
		b.WriteString(styleMuted.Render("原问题") + "\n")
		b.WriteString(wrapPlain(m.questionAdvice.Original, width) + "\n\n")
		b.WriteString(styleSuccess.Render("可以改成") + "\n")
		b.WriteString(wrapPlain(m.questionAdvice.Suggested, width) + "\n\n")
		b.WriteString(styleMuted.Render(wrapPlain(m.questionAdvice.Reason, width)))
	}
	return StateView{Left: left, Right: b.String()}
}

// CrisisState stops a reading. esc returns to the editor; nothing here draws.
type CrisisState struct{}

func (s *CrisisState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch key.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "esc":
		return &InputState{}, nil
	default:
		return s, nil
	}
}

func (s *CrisisState) View(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			styleMuted.Italic(true).Render("先不抽牌"))
	width := m.layout.RightWidth - 2
	if width < 10 {
		width = 10
	}
	right := renderPanelTitle("关怀", colorPrimary) + "\n" + wrapPlain(crisisNote, width)
	return StateView{Left: left, Right: right}
}

func enterSpread(m *Model) *SpreadState {
	m.spreadAdvice = RecommendSpread(m.userInput)
	m.priorMatch = domain.FindSimilarQuestion(m.userInput, recentReadings(m))
	return &SpreadState{}
}

func recentReadings(m *Model) []domain.Reading {
	if m.store == nil || m.store.Readings == nil {
		return nil
	}
	list, err := m.store.Readings.List(domain.SimilarLookback)
	if err != nil {
		return nil
	}
	return list
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
		content += m.renderSpread(m.revealIndex, 3)
	} else {
		content = styleSuccess.Render("  ✦ 牌已全部翻开") + "\n\n"
		content += m.renderSpread(len(m.drawResult.Cards), 3)
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
	signals := domain.SpreadSignals(m.drawResult.Cards)
	prompt := readingPrompt(m.userInput, m.spreadType, cardInfo, signals, domain.FormatPriorBlock(m.priorMatch), m.boundary.PromptLine())

	m.bridge.setup(prompt, m.agentTimeout())
	return &ReadingState{}, m.bridge.nextEvent()
}

type revealCompleteMsg struct{}

// --- ReadingState ---

type ReadingState struct{}

func (s *ReadingState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case agentDeltaMsg:
		m.retryNote = ""
		m.readingBuf.WriteString(msg.text)
		return s, tea.Batch(m.scheduleStreamFlush(), m.bridge.nextEvent())

	case streamFlushMsg:
		if msg.gen != m.streamGen {
			return s, nil
		}
		m.flushReadingPlain()
		return s, nil

	case agentRetryMsg:
		m.retryNote = msg.note
		return s, m.bridge.nextEvent()

	case agentToolMsg:
		m.toolCalls = append(m.toolCalls, msg.name)
		return s, m.bridge.nextEvent()

	case agentEndMsg:
		m.bridge.cleanup()
		m.finishReading(nil)
		return &FollowUpState{}, nil

	case agentErrMsg:
		m.bridge.cleanup()
		m.finishReading(msg.err)
		return &FollowUpState{}, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, m.quitAgent()
		case "up", "k":
			m.readingVP.LineUp(1)
		case "down", "j":
			m.readingVP.LineDown(1)
		case "pgup":
			m.readingVP.HalfViewUp()
		case "pgdown":
			m.readingVP.HalfViewDown()
		case "ctrl+o":
			if next, ok := openCardDetail(m, s); ok {
				return next, nil
			}
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
		case "tab":
			m.toggleFocus()
			return s, nil
		case "up", "k":
			m.focusedVP().LineUp(1)
			return s, nil
		case "down", "j":
			m.focusedVP().LineDown(1)
			return s, nil
		case "pgup":
			m.focusedVP().HalfViewUp()
			return s, nil
		case "pgdown":
			m.focusedVP().HalfViewDown()
			return s, nil
		case "ctrl+o":
			if next, ok := openCardDetail(m, s); ok {
				return next, nil
			}
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
			m.bridge.setup(m.nextFollowUpPrompt(input), m.agentTimeout())
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
		m.retryNote = ""
		m.chatStreamBuf.WriteString(msg.text)
		return s, tea.Batch(m.scheduleStreamFlush(), m.bridge.nextEvent())

	case streamFlushMsg:
		if msg.gen != m.streamGen {
			return s, nil
		}
		m.flushChatPlain()
		return s, nil

	case agentRetryMsg:
		m.retryNote = msg.note
		return s, m.bridge.nextEvent()

	case agentToolMsg:
		m.toolCalls = append(m.toolCalls, msg.name)
		return s, m.bridge.nextEvent()

	case agentEndMsg:
		m.bridge.cleanup()
		m.bumpStreamGen()
		m.retryNote = ""
		m.finalizeChatStream()
		m.persistFollowUps()
		m.chatVP.SetContent(m.renderChatConversation(false))
		return &FollowUpState{}, nil

	case agentErrMsg:
		m.bridge.cleanup()
		m.bumpStreamGen()
		m.retryNote = ""
		m.finalizeChatStream()
		m.persistFollowUps()
		m.chatVP.SetContent(m.renderChatConversation(false))
		if m.err != nil {
			m.err = fmt.Errorf("%v；%w", msg.err, m.err)
			return &FollowUpState{}, nil
		}
		m.err = msg.err
		return &FollowUpState{}, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, m.quitAgent()
		case "tab":
			m.toggleFocus()
		case "up", "k":
			m.focusedVP().LineUp(1)
		case "down", "j":
			m.focusedVP().LineDown(1)
		case "pgup":
			m.focusedVP().HalfViewUp()
		case "pgdown":
			m.focusedVP().HalfViewDown()
		case "ctrl+o":
			if next, ok := openCardDetail(m, s); ok {
				return next, nil
			}
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
				m.err = nil
				m.exportNote = ""
				s.updateViewport(m)
			}
			return s, nil
		case "down", "j":
			if m.historyCursor < len(m.historyReadings)-1 {
				m.historyCursor++
				m.err = nil
				m.exportNote = ""
				s.updateViewport(m)
			}
			return s, nil
		case "pgup":
			m.readingVP.HalfViewUp()
			return s, nil
		case "pgdown":
			m.readingVP.HalfViewDown()
			return s, nil
		case "e", "E":
			m.exportSelectedReading()
			return s, nil
		case "enter":
			r, ok := selectedReading(m)
			if !ok || len(r.Cards) == 0 {
				return s, nil
			}
			m.resumeReading(r)
			return &FollowUpState{}, nil
		}
	}
	return s, nil
}

func (s *HistoryState) updateViewport(m *Model) {
	if m.historyCursor >= 0 && m.historyCursor < len(m.historyReadings) {
		r := m.historyReadings[m.historyCursor]
		m.readingVP.SetContent(renderMarkdown(renderHistoryDetail(r), m.readingVP.Width-2))
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
			prefix := fmt.Sprintf("%s%s %s — ",
				cursor,
				styleSubtle.Render(date),
				styleMuted.Render(spread))
			question := truncateWidth(r.Question, historyQuestionWidth(m, prefix))
			line := prefix + question
			items += line + "\n"
		}
		left += listStyle.Render(items)
	}

	// Right: selected reading
	scrollHint := styleSubtle.Render(historyScrollHint())
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
		left += m.renderSpread(len(m.drawResult.Cards), 1)
	} else {
		left += renderCentered(m.layout.BodyHeight-1, styleMuted.Render("等待抽牌..."))
	}
	scrollHint := styleSubtle.Render("  ↑↓/jk 滚动")
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		m.readingVP.View() + "\n" + scrollHint
	return StateView{Left: left, Right: right}
}

// renderSplitView renders the split view (FollowUpState, ChatState).
// Reading viewport on top, chat viewport on bottom — each constrained to its own height.
func renderSplitView(m *Model) StateView {
	left := renderPanelTitle("牌面", colorAccent) + "\n"
	if m.drawResult != nil {
		left += m.renderSpread(len(m.drawResult.Cards), 1)
	} else {
		left += renderCentered(m.layout.BodyHeight-1, styleMuted.Render("等待抽牌..."))
	}

	// Reading section: title + viewport, constrained to readingVP.Height
	readingColor := colorPrimary
	if m.focusZone == "reading" {
		readingColor = colorSuccess // highlight focused zone
	}
	readingTitle := renderPanelTitle("解读", readingColor)
	readingContent := m.readingVP.View()
	readingPart := lipgloss.NewStyle().
		Width(m.layout.RightWidth).
		Height(m.readingVP.Height + 1).
		Render(readingTitle + "\n" + readingContent)

	// Separator
	chatSep := separatorStyle.Render(strings.Repeat("─", m.layout.RightWidth-2))

	// Chat section: title + content, constrained to chatVP.Height
	chatColor := colorAccent
	if m.focusZone == "chat" {
		chatColor = colorSuccess
	}
	chatTitle := renderPanelTitle("对话", chatColor)
	var chatBody string
	if len(m.chatMessages) == 0 && m.chatStreamBuf.Len() == 0 {
		chatBody = "\n" + styleMuted.Italic(true).Render("  解读后可以继续追问...")
	} else {
		chatBody = "\n" + m.chatVP.View()
	}
	focusLabel := "解读"
	if m.focusZone == "chat" {
		focusLabel = "对话"
	}
	chatHint := "\n" + styleSubtle.Render("  ↑↓/jk 滚动 · tab 切换 · 焦点: "+focusLabel)
	chatPart := lipgloss.NewStyle().
		Width(m.layout.RightWidth).
		Height(m.chatVP.Height + 2). // +2 for title + hint
		Render(chatTitle + chatBody + chatHint)

	right := readingPart + "\n" + chatSep + "\n" + chatPart
	return StateView{Left: left, Right: right}
}
