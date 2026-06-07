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
		if key.String() == "ctrl+c" {
			return s, tea.Quit
		}
		if key.String() == "enter" {
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return s, nil
			}
			if isExitCmd(input) {
				return s, tea.Quit
			}
			m.userInput = input
			m.reading.Reset()
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
			lipgloss_muted().Italic(true).Render("等待抽牌..."))
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
			lipgloss_muted().Italic(true).Render("等待抽牌..."))
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
	m.reading.Reset()
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
		content = lipgloss_muted().Render("  等待抽牌...")
	} else if m.revealIndex < len(m.drawResult.Cards) {
		sp := spinnerStyle.Render(m.spinner.View())
		status := fmt.Sprintf(" 正在翻牌... (%d/%d)", m.revealIndex, len(m.drawResult.Cards))
		content = sp + status + "\n\n"
		content += renderSpreadLayout(m.drawResult.Cards, m.revealIndex, m.spreadType, m.layout.LeftWidth-4)
	} else {
		content = lipgloss_success().Render("  ✦ 牌已全部翻开") + "\n\n"
		content += renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
	}

	left := renderPanelTitle("牌面", colorAccent) + "\n" + content

	// Right: waiting
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		renderCentered(m.layout.BodyHeight-1,
			lipgloss_muted().Italic(true).Render("等待翻牌完成..."))

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
		m.reading.WriteString(msg.text)
		m.readingVP.SetContent(renderMarkdown(m.reading.String()))
		m.readingVP.GotoBottom()
		return s, m.bridge.nextEvent()

	case agentToolMsg:
		m.toolCalls = append(m.toolCalls, msg.name)
		return s, m.bridge.nextEvent()

	case agentEndMsg:
		m.bridge.cleanup()
		return &FollowUpState{}, nil

	case agentErrMsg:
		m.bridge.cleanup()
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
	// Left: cards
	left := renderPanelTitle("牌面", colorAccent) + "\n"
	if m.drawResult != nil {
		cards := renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
		left += cards
	} else {
		left += renderCentered(m.layout.BodyHeight-1, lipgloss_muted().Render("等待抽牌..."))
	}

	// Right: reading viewport
	scrollHint := lipgloss_subtle().Render("  ↑↓/jk 滚动")
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
		m.readingVP.View() + "\n" + scrollHint

	return StateView{Left: left, Right: right}
}

// --- FollowUpState ---

type FollowUpState struct{}

func (s *FollowUpState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "up", "k":
			m.readingVP.LineUp(1)
			return s, nil
		case "down", "j":
			m.readingVP.LineDown(1)
			return s, nil
		case "pgup":
			m.readingVP.HalfViewUp()
			return s, nil
		case "pgdown":
			m.readingVP.HalfViewDown()
			return s, nil
		case "enter":
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				m.bridge.clearMessages()
				m.input.Reset()
				return &InputState{}, nil
			}
			if isExitCmd(input) {
				return s, tea.Quit
			}
			m.reading.Reset()
			m.toolCalls = nil
			m.err = nil
			m.bridge.setup(input)
			return &ReadingState{}, m.bridge.nextEvent()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return s, cmd
}

func (s *FollowUpState) View(m *Model) StateView {
	// Left: cards
	left := renderPanelTitle("牌面", colorAccent) + "\n"
	if m.drawResult != nil {
		cards := renderSpreadLayout(m.drawResult.Cards, len(m.drawResult.Cards), m.spreadType, m.layout.LeftWidth-4)
		left += cards
	} else {
		left += renderCentered(m.layout.BodyHeight-1, lipgloss_muted().Render("等待抽牌..."))
	}

	scrollHint := lipgloss_subtle().Render("  ↑↓/jk 滚动")
	right := renderPanelTitle("解读", colorPrimary) + "\n" +
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

// lipgloss helpers to avoid import cycle with styles.go
func lipgloss_muted() lipgloss_style   { return lipgloss.NewStyle().Foreground(colorMuted) }
func lipgloss_subtle() lipgloss_style   { return lipgloss.NewStyle().Foreground(colorSubtle) }
func lipgloss_success() lipgloss_style  { return lipgloss.NewStyle().Foreground(colorSuccess) }

type lipgloss_style = lipgloss.Style
