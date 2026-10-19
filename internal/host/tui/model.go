package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/host/reminder"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/agentcore"
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

// layoutHeights calculates layout dimensions using fixed input heights
// to prevent viewport jumping when textarea content changes.
func (m *Model) layoutHeights() {
	w := m.width
	h := m.height

	if w == 0 || h == 0 {
		return
	}

	// Fixed heights — never measure dynamically to avoid layout jumps.
	const (
		topH          = 1 // status bar always 1 line
		sepH          = 1 // separator between body and input
		inputHDefault = 8 // label(1) + textarea(max 6) + hint(1)
		inputHSpread  = 6 // label(1) + 3 options + reason + hint(1)
		inputHGuide   = 4 // label(1) + two choices + hint(1)
		inputHHistory = 1 // single label line
	)

	var inputH int
	switch m.state.(type) {
	case *SpreadState:
		inputH = inputHSpread
	case *QuestionGuideState:
		inputH = inputHGuide
	case *HistoryState:
		inputH = inputHHistory
	default:
		inputH = inputHDefault
	}

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

	// In FollowUpState/ChatState, split the right panel: reading on top, chat on bottom.
	// Chat section needs: title(1) + viewport + hint(1) + separator(1) = vp + 3 lines
	switch m.state.(type) {
	case *FollowUpState, *ChatState:
		const chatOverhead = 4 // title + hint + separator + margin
		chatH = bodyH * 35 / 100
		if chatH < 4 {
			chatH = 4
		}
		readingH = bodyH - chatH - chatOverhead
		if readingH < 4 {
			readingH = 4
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
	mode       string // "专业模式" or "轻松模式"
	llmModel   string // provider model id, stored with each reading
	userInput  string
	spreadType string

	// questionAdvice is set only while a rewrite is being offered.
	questionAdvice *QuestionAdvice
	// spreadAdvice highlights one spread after the question is confirmed.
	spreadAdvice SpreadAdvice
	drawResult   *domain.DrawResult
	revealIndex  int

	// Reading (decoupled from conversation)
	readingContent string          // the AI's interpretation text
	readingBuf     strings.Builder // streaming buffer during ReadingState

	// Chat conversation (decoupled from reading)
	chatMessages  []ChatMessage   // user + assistant messages for follow-ups
	chatStreamBuf strings.Builder // streaming buffer for current AI chat response

	toolCalls []string
	err       error

	// retryNote is a transient status, cleared when text resumes or the run ends.
	retryNote string

	// timeoutOverride is set when TAROT_AGENT_TIMEOUT parsed successfully.
	timeoutOverride    time.Duration
	timeoutOverrideSet bool

	// streamGen invalidates a late plain-text flush after the final markdown render.
	streamGen          int
	streamFlushPending bool
	spinnerActive      bool

	// historyWritten is set after the current draw has been appended once.
	historyWritten bool

	// Focus zone for scroll control ("reading" or "chat")
	focusZone string

	// History state
	historyReadings []domain.Reading
	historyCursor   int
}

func NewModel(agent *agentcore.Agent, guard *reminder.ReadingGuard, s *store.Store, mode, llmModel string) *Model {
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
		llmModel:  llmModel,
		input:     ta,
		readingVP: vp,
		chatVP:    chatVP,
		spinner:   sp,
	}
}

func (m *Model) Init() tea.Cmd {
	m.spinnerActive = true
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
		m.layoutHeights()
		m.reflowViewports()
		return m, nil

	case spinner.TickMsg:
		if !m.spinnerOn() {
			m.spinnerActive = false
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.spinnerActive = cmd != nil
		return m, cmd
	}

	prevSpin := m.spinnerOn()
	newState, cmd := m.state.Update(m, msg)
	if newState != nil {
		m.state = newState
		m.layoutHeights()
	}
	if !prevSpin && m.spinnerOn() && !m.spinnerActive {
		m.spinnerActive = true
		cmd = tea.Batch(cmd, m.spinner.Tick)
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

	// Re-sync viewport widths (heights are stable from layoutHeights)
	vpW := m.layout.RightWidth - 6
	if vpW < 10 {
		vpW = 10
	}
	if m.readingVP.Width != vpW {
		m.readingVP.Width = vpW
	}
	if m.chatVP.Width != vpW {
		m.chatVP.Width = vpW
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
	case *QuestionGuideState:
		inputLabel = "可以改写这句，也可以保留原问题："
	case *SpreadState:
		inputLabel = "选择牌阵 (1/2/3)："
	case *RevealState:
		inputLabel = "正在翻牌"
	case *ReadingState:
		inputLabel = "正在解读"
	case *FollowUpState:
		inputLabel = "还有什么想问的？（空回车开始新占卜）"
	case *ChatState:
		inputLabel = "正在回答"
	case *HistoryState:
		inputLabel = "浏览历史记录（↑↓ 选择 · esc 返回）"
	}

	switch m.state.(type) {
	case *SpreadState:
		b.WriteString(inputLabel + "\n")
		b.WriteString(renderSpreadOptions(m))
	case *QuestionGuideState:
		b.WriteString(inputLabel + "\n")
		b.WriteString(renderQuestionGuideOptions())
	case *HistoryState:
		b.WriteString(statusBarStyle.Render("  " + inputLabel))
	default:
		b.WriteString(renderInputZone(m, inputLabel))
	}

	// 5. Retry / error. These sit under the input so a running reading can show them.
	if m.retryNote != "" {
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(wrapPlain("  "+m.retryNote, m.layout.Width)))
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(wrapPlain("  ⚠ "+m.err.Error(), m.layout.Width)))
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
	m.bumpStreamGen()
	m.readingContent = ""
	m.readingBuf.Reset()
	m.readingVP.SetContent("")
	m.historyWritten = false
	m.retryNote = ""
}

// resetChat clears the chat conversation.
func (m *Model) resetChat() {
	m.chatMessages = nil
	m.chatStreamBuf.Reset()
	m.chatVP.SetContent("")
	m.focusZone = "reading"
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

// focusedVP returns the currently focused viewport for scroll control.
func (m *Model) focusedVP() *viewport.Model {
	if m.focusZone == "chat" {
		return &m.chatVP
	}
	return &m.readingVP
}

// toggleFocus switches focus between "reading" and "chat".
func (m *Model) toggleFocus() {
	if m.focusZone == "chat" {
		m.focusZone = "reading"
	} else {
		m.focusZone = "chat"
	}
}

// chatTranscript is the conversation plus the in-progress reply when streaming.
func (m *Model) chatTranscript(streaming bool) string {
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
	return b.String()
}

// renderChatConversation renders the full chat history for the viewport.
// If streaming is true, includes the current stream buffer as the latest message.
func (m *Model) renderChatConversation(streaming bool) string {
	return renderMarkdown(m.chatTranscript(streaming), viewportWrapWidth(m.chatVP.Width))
}

const streamFlushInterval = 80 * time.Millisecond

type streamFlushMsg struct{ gen int }

func (m *Model) agentTimeout() time.Duration {
	return agentTimeoutForSpread(m.spreadType, m.timeoutOverride, m.timeoutOverrideSet)
}

func (m *Model) scheduleStreamFlush() tea.Cmd {
	if m.streamFlushPending {
		return nil
	}
	m.streamFlushPending = true
	gen := m.streamGen
	return tea.Tick(streamFlushInterval, func(time.Time) tea.Msg {
		return streamFlushMsg{gen: gen}
	})
}

func (m *Model) bumpStreamGen() {
	m.streamGen++
	m.streamFlushPending = false
}

func (m *Model) flushReadingPlain() {
	m.streamFlushPending = false
	m.readingVP.SetContent(wrapPlain(m.readingBuf.String(), viewportWrapWidth(m.readingVP.Width)))
	m.readingVP.GotoBottom()
}

func (m *Model) flushChatPlain() {
	m.streamFlushPending = false
	m.chatVP.SetContent(wrapPlain(m.chatTranscript(true), viewportWrapWidth(m.chatVP.Width)))
	m.chatVP.GotoBottom()
}

func (m *Model) reflowViewports() {
	switch m.state.(type) {
	case *ReadingState:
		if m.readingBuf.Len() > 0 {
			m.flushReadingPlain()
		}
	case *ChatState:
		if m.readingContent != "" {
			m.readingVP.SetContent(m.renderReadingMarkdown())
		}
		m.flushChatPlain()
	case *FollowUpState:
		if m.readingContent != "" {
			m.readingVP.SetContent(m.renderReadingMarkdown())
		}
		if len(m.chatMessages) > 0 || m.chatStreamBuf.Len() > 0 {
			m.chatVP.SetContent(m.renderChatConversation(m.chatStreamBuf.Len() > 0))
		}
	case *HistoryState:
		if m.historyCursor >= 0 && m.historyCursor < len(m.historyReadings) {
			r := m.historyReadings[m.historyCursor]
			m.readingVP.SetContent(renderMarkdown(renderHistoryDetail(r), viewportWrapWidth(m.readingVP.Width)))
		}
	}
}

func viewportWrapWidth(width int) int {
	w := width - 2
	if w < 1 {
		return 1
	}
	return w
}

func (m *Model) quitAgent() tea.Cmd {
	m.bumpStreamGen()
	m.bridge.abort()
	m.bridge.cleanup()
	return tea.Quit
}

// lipglossHeight measures the rendered height of a string.
func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}
