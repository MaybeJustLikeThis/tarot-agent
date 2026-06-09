package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/store"
)

// Layout holds calculated zone dimensions.
type Layout struct {
	Width      int
	Height     int
	TopH       int // status bar height (measured)
	InputH     int // input zone height (measured)
	BodyHeight int // height for left/right panels
	LeftWidth  int
	RightWidth int
}

// layoutHeights measures actual rendered heights and calculates body height.
// Follows ainovel-cli pattern: render first, measure, then calculate remainder.
func (m *Model) layoutHeights() {
	w := m.width
	h := m.height

	if w == 0 || h == 0 {
		return
	}

	// Measure actual rendered heights
	topBar := renderStatusBar(m)
	topH := lipglossHeight(topBar)

	var inputBar string
	switch m.state.(type) {
	case *SpreadState:
		inputBar = "选择牌阵 (1/2/3)：\n" + renderSpreadOptions(m)
	case *HistoryState:
		inputBar = "浏览历史记录（↑↓ 选择 · esc 返回）"
	default:
		inputBar = renderInputZone(m, "说说你的情况和想问的问题：")
	}
	inputH := lipglossHeight(inputBar)

	sepH := 1 // one separator between body and input

	bodyH := h - topH - inputH - sepH
	if bodyH < 5 {
		bodyH = 5
	}

	leftW := w * 40 / 100
	if leftW < 20 {
		leftW = 20
	}
	rightW := w - leftW
	if rightW < 30 {
		rightW = 30
		leftW = w - rightW
	}

	m.layout = Layout{
		Width:      w,
		Height:     h,
		TopH:       topH,
		InputH:     inputH,
		BodyHeight: bodyH,
		LeftWidth:  leftW,
		RightWidth: rightW,
	}

	// Sync viewports to right panel dimensions
	readingH := bodyH - 3 // -3 for panel title + scroll hint
	chatH := 0

	// In FollowUpState, split the right panel: 60% reading, 40% chat
	if _, isFollowUp := m.state.(*FollowUpState); isFollowUp {
		readingH = bodyH * 60 / 100
		if readingH < 4 {
			readingH = 4
		}
		chatH = bodyH - readingH - 1 // -1 for separator between reading and chat
		if chatH < 4 {
			chatH = 4
		}
	}

	vpW := rightW - 6
	if vpW < 10 {
		vpW = 10
	}

	m.readingVP.Width = vpW
	m.readingVP.Height = readingH
	if m.readingVP.Width < 10 {
		m.readingVP.Width = 10
	}
	if m.readingVP.Height < 3 {
		m.readingVP.Height = 3
	}

	m.chatVP.Width = vpW
	m.chatVP.Height = chatH
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// Model is the main TUI model. Thin orchestrator — state logic lives in state.go.
type Model struct {
	state  State
	bridge *agentBridge
	store  *store.Store

	input     textarea.Model
	readingVP viewport.Model // top-right: tarot interpretation
	chatVP    viewport.Model // bottom-right: conversation (follow-up Q&A)
	spinner   spinner.Model
	layout    Layout
	width     int
	height    int

	// Shared state accessible by all states
	mode        string // "专业模式" or "轻松模式"
	userInput   string
	spreadType  string
	drawResult  *domain.DrawResult
	revealIndex int

	// Reading (decoupled from conversation)
	readingContent string    // the AI's interpretation text
	readingBuf     strings.Builder // streaming buffer during ReadingState

	// Chat conversation (decoupled from reading)
	chatMessages  []ChatMessage   // user + assistant messages for follow-ups
	chatStreamBuf strings.Builder // streaming buffer for current AI chat response

	toolCalls []string
	err       error

	// History state
	historyReadings []domain.Reading
	historyCursor   int
}

func NewModel(agent *agentcore.Agent, guard *reminder.ReadingGuard, s *store.Store, mode string) *Model {
	ta := textarea.New()
	ta.Placeholder = "说说你的情况和想问的问题..."
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(2)
	ta.MaxHeight = 6
	ta.ShowLineNumbers = false

	vp := viewport.New(40, 10)
	chatVP := viewport.New(40, 5)

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = spinnerStyle

	return &Model{
		state:     &InputState{},
		bridge:    newAgentBridge(agent, guard),
		store:     s,
		mode:      mode,
		input:     ta,
		readingVP: vp,
		chatVP:    chatVP,
		spinner:   sp,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
		m.layoutHeights() // measure + calculate + sync viewport
		return m, nil

	case spinner.TickMsg:
		if m.spinnerOn() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	newState, cmd := m.state.Update(m, msg)
	if newState != nil {
		m.state = newState
	}
	return m, cmd
}

func (m *Model) View() string {
	// Minimum terminal size check (like ainovel-cli)
	if m.width < 80 || m.height < 20 {
		return "\n  ✦ 星语 Tarot Agent\n\n" +
			"  终端窗口太小，请调整到至少 80×20。\n" +
			fmt.Sprintf("  当前大小：%d×%d\n", m.width, m.height)
	}

	if m.layout.Width == 0 || m.layout.Height == 0 {
		return "Loading..."
	}

	// Re-sync viewports every frame
	readingH := m.layout.BodyHeight - 3
	chatH := 0
	if _, isFollowUp := m.state.(*FollowUpState); isFollowUp {
		readingH = m.layout.BodyHeight * 60 / 100
		if readingH < 4 {
			readingH = 4
		}
		chatH = m.layout.BodyHeight - readingH - 1
		if chatH < 4 {
			chatH = 4
		}
	}
	vpW := m.layout.RightWidth - 6
	if vpW < 10 {
		vpW = 10
	}
	if m.readingVP.Width != vpW {
		m.readingVP.Width = vpW
	}
	if m.readingVP.Height != readingH {
		m.readingVP.Height = readingH
	}
	if m.chatVP.Width != vpW {
		m.chatVP.Width = vpW
	}
	if m.chatVP.Height != chatH {
		m.chatVP.Height = chatH
	}

	var b strings.Builder

	// 1. Status bar
	b.WriteString(renderStatusBar(m))
	b.WriteString("\n")

	// 2. Body: left | right side by side
	sv := m.state.View(m)
	leftPanel := renderLeftPanel(m, sv.Left)
	rightPanel := renderRightPanel(m, sv.Right)
	body := joinHorizontal(leftPanel, rightPanel)
	b.WriteString(body)

	// 3. Separator
	b.WriteString(renderSeparator(m.layout.Width))
	b.WriteString("\n")

	// 4. Input zone
	inputLabel := "说说你的情况和想问的问题："
	switch m.state.(type) {
	case *SpreadState:
		inputLabel = "选择牌阵 (1/2/3)："
	case *FollowUpState, *ChatState:
		inputLabel = "还有什么想问的？（回车开始新占卜）"
	case *HistoryState:
		inputLabel = "浏览历史记录（↑↓ 选择 · esc 返回）"
	}

	if _, isSpread := m.state.(*SpreadState); isSpread {
		b.WriteString(inputLabel + "\n")
		b.WriteString(renderSpreadOptions(m))
	} else if _, isHistory := m.state.(*HistoryState); isHistory {
		b.WriteString(statusBarStyle.Render("  " + inputLabel))
	} else {
		b.WriteString(renderInputZone(m, inputLabel))
	}

	// 5. Error
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  ⚠ " + m.err.Error()))
	}

	return b.String()
}

// joinHorizontal joins two multiline strings side by side, line by line.
func joinHorizontal(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	for len(leftLines) < maxLen {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLen {
		rightLines = append(rightLines, "")
	}

	var result strings.Builder
	for i := 0; i < maxLen; i++ {
		result.WriteString(leftLines[i])
		result.WriteString(rightLines[i])
		result.WriteString("\n")
	}

	return result.String()
}

func (m *Model) spinnerOn() bool {
	switch m.state.(type) {
	case *RevealState, *ReadingState, *ChatState:
		return true
	}
	return false
}

// resetReading clears the reading content.
func (m *Model) resetReading() {
	m.readingContent = ""
	m.readingBuf.Reset()
	m.readingVP.SetContent("")
}

// resetChat clears the chat conversation.
func (m *Model) resetChat() {
	m.chatMessages = nil
	m.chatStreamBuf.Reset()
	m.chatVP.SetContent("")
}

// appendChatUserMessage adds a user message to the chat.
func (m *Model) appendChatUserMessage(text string) {
	m.chatMessages = append(m.chatMessages, ChatMessage{Role: "user", Content: text})
}

// finalizeChatStream moves the streaming buffer into chat messages.
func (m *Model) finalizeChatStream() {
	if m.chatStreamBuf.Len() > 0 {
		m.chatMessages = append(m.chatMessages, ChatMessage{Role: "assistant", Content: m.chatStreamBuf.String()})
		m.chatStreamBuf.Reset()
	}
}

// renderReadingMarkdown renders the reading content for the viewport.
func (m *Model) renderReadingMarkdown() string {
	return renderMarkdown(m.readingContent, m.readingVP.Width-2)
}

// renderChatConversation renders the full chat history for the viewport.
// If streaming is true, includes the current stream buffer as the latest message.
func (m *Model) renderChatConversation(streaming bool) string {
	var b strings.Builder
	for _, msg := range m.chatMessages {
		if msg.Role == "user" {
			b.WriteString("> **你**：" + msg.Content + "\n\n")
		} else {
			b.WriteString(msg.Content + "\n\n")
		}
	}
	if streaming && m.chatStreamBuf.Len() > 0 {
		b.WriteString(m.chatStreamBuf.String())
	}
	return renderMarkdown(b.String(), m.chatVP.Width-2)
}

// lipglossHeight measures the rendered height of a string.
func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}
