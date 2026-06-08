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

	// Sync viewport to right panel dimensions
	m.readingVP.Width = rightW - 4  // -4 for padding
	m.readingVP.Height = bodyH - 3  // -3 for panel title + scroll hint
	if m.readingVP.Width < 10 {
		m.readingVP.Width = 10
	}
	if m.readingVP.Height < 3 {
		m.readingVP.Height = 3
	}
}

// Model is the main TUI model. Thin orchestrator — state logic lives in state.go.
type Model struct {
	state  State
	bridge *agentBridge
	store  *store.Store

	input     textarea.Model
	readingVP viewport.Model
	spinner   spinner.Model
	layout    Layout
	width     int
	height    int

	// Shared state accessible by all states
	mode      string // "专业模式" or "轻松模式"
	userInput   string
	spreadType  string
	drawResult  *domain.DrawResult
	revealIndex int
	reading     strings.Builder
	toolCalls   []string
	err         error
}

func NewModel(agent *agentcore.Agent, guard *reminder.ReadingGuard, s *store.Store, mode string) *Model {
	ta := textarea.New()
	ta.Placeholder = "说说你的情况和想问的问题..."
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.MaxHeight = 3
	ta.ShowLineNumbers = false

	vp := viewport.New(40, 10)

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

	// Re-sync viewport size every frame (ainovel-cli pattern: View() has inline guard)
	if m.readingVP.Width != m.layout.RightWidth-4 {
		m.readingVP.Width = m.layout.RightWidth - 4
	}
	if m.readingVP.Height != m.layout.BodyHeight-3 {
		m.readingVP.Height = m.layout.BodyHeight - 3
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
	case *FollowUpState:
		inputLabel = "还有什么想问的？（回车开始新占卜）"
	}

	if _, isSpread := m.state.(*SpreadState); isSpread {
		b.WriteString(inputLabel + "\n")
		b.WriteString(renderSpreadOptions(m))
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
	case *RevealState, *ReadingState:
		return true
	}
	return false
}

// lipglossHeight measures the rendered height of a string.
func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}
