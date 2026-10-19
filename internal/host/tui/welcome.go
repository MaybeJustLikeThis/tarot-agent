package tui

import (
	"log/slog"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/bootstrap"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	welcomeAsk = "" +
		"说具体情况，少问是或否。\n" +
		"\n" +
		"· 接下来三个月，工作和生活怎么安排？\n" +
		"· 这段关系里，我可以先改变什么？\n" +
		"· 面对这个选择，我忽略了哪一面？"

	welcomeDo = "" +
		"深入版多讲元素和数字。简洁版用大白话，更短。\n" +
		"\n" +
		"空输入 tab 看历史。解读之后可以追问。\n" +
		"历史里按 e 导出。\n" +
		"\n" +
		"daily    抽一张今日牌\n" +
		"card     离线查一张牌的含义\n" +
		"export   把最近一次导出成文本\n" +
		"update   有新版本时安装\n" +
		"\n" +
		"按任意键进入提问。空输入后再按 ? 可以再看。"
)

// WelcomeState is the short product guide. Any key on a usable terminal
// dismisses it. ctrl+c still quits.
type WelcomeState struct{}

func (s *WelcomeState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if key.String() == "ctrl+c" {
		if welcomeReadable(m) {
			markWelcomeSeen()
		}
		return s, tea.Quit
	}
	if !welcomeReadable(m) {
		return s, nil
	}
	markWelcomeSeen()
	return &InputState{}, nil
}

func (s *WelcomeState) View(m *Model) StateView {
	leftW := m.layout.LeftWidth - 2
	rightW := m.layout.RightWidth - 2
	lines := m.layout.BodyHeight - 1
	if lines < 1 {
		lines = 1
	}
	left := renderPanelTitle("怎么问", colorAccent) + "\n" + fitLines(welcomeAsk, leftW, lines)
	right := renderPanelTitle("可以做什么", colorPrimary) + "\n" + fitLines(welcomeDo, rightW, lines)
	return StateView{Left: left, Right: right}
}

func applyWelcome(m *Model) {
	if !bootstrap.WelcomeSeen() {
		m.state = &WelcomeState{}
	}
}

func welcomeReadable(m *Model) bool {
	return m.width >= 80 && m.height >= 20
}

func markWelcomeSeen() {
	if err := bootstrap.MarkWelcomeSeen(); err != nil {
		slog.Warn("save welcome marker", "error", err)
	}
}
