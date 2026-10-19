package tui

import (
	"fmt"
	"strings"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)

// CardDetailState shows local card data for the current draw.
// Agent events keep flowing to the reading underneath. esc restores that
// state, including the reading scroll position.
type CardDetailState struct {
	prev     State
	index    int
	readingY int
	chatY    int
}

func openCardDetail(m *Model, prev State) (State, bool) {
	if m.drawResult == nil || len(m.drawResult.Cards) == 0 {
		return prev, false
	}
	return &CardDetailState{
		prev:     prev,
		readingY: m.readingVP.YOffset,
		chatY:    m.chatVP.YOffset,
	}, true
}

func (s *CardDetailState) Update(m *Model, msg tea.Msg) (State, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return s.prev, nil
		case "ctrl+c":
			return s.prev.Update(m, msg)
		case "left", "h":
			s.move(-1, len(m.drawResult.Cards))
			return s, nil
		case "right", "l":
			s.move(1, len(m.drawResult.Cards))
			return s, nil
		default:
			return s, nil
		}
	}
	next, cmd := s.prev.Update(m, msg)
	if next != nil && next != s.prev {
		s.prev = next
	}
	return s, cmd
}

func (s *CardDetailState) move(delta, n int) {
	if n <= 0 {
		return
	}
	s.index += delta
	if s.index < 0 {
		s.index = 0
	}
	if s.index >= n {
		s.index = n - 1
	}
}

func (s *CardDetailState) View(m *Model) StateView {
	cards := m.drawResult.Cards
	if s.index >= len(cards) {
		s.index = len(cards) - 1
	}
	if s.index < 0 {
		s.index = 0
	}
	leftW := m.layout.LeftWidth - 3
	if leftW < 8 {
		leftW = 8
	}
	rightW := m.layout.RightWidth - 2
	if rightW < 8 {
		rightW = 8
	}

	var left strings.Builder
	left.WriteString(renderPanelTitle("牌阵", colorAccent) + "\n")
	left.WriteString(wrapPlain(fmt.Sprintf("%d/%d", s.index+1, len(cards)), leftW) + "\n")
	left.WriteString(styleMuted.Render("整体信号") + "\n")
	signals := domain.SpreadSignals(cards)
	if len(cards) < 2 {
		left.WriteString(wrapPlain("单张牌不列整体信号", leftW) + "\n")
	} else if len(signals) == 0 {
		left.WriteString(wrapPlain("没有足够依据的整体信号", leftW) + "\n")
	} else {
		for _, line := range signals {
			left.WriteString(wrapPlain("· "+line, leftW) + "\n")
		}
	}
	left.WriteString("\n")
	for i, card := range cards {
		mark := "  "
		if i == s.index {
			mark = "▸ "
		}
		name := card.Card.NameCN
		if name == "" {
			name = card.Card.ID
		}
		pos := card.Position.NameCN
		if pos == "" {
			pos = fmt.Sprintf("%d", i+1)
		}
		left.WriteString(wrapPlain(mark+pos+" "+name, leftW) + "\n")
	}

	right := renderPanelTitle("牌义", colorPrimary) + "\n" +
		formatCardDetail(cards[s.index], rightW)
	return StateView{Left: left.String(), Right: right}
}

func formatCardDetail(card domain.DrawnCard, width int) string {
	c := card.Card
	name := c.NameCN
	if c.NameEN != "" {
		name += " " + c.NameEN
	}
	if name == "" {
		name = c.ID
	}
	orient := "正位"
	keywords := c.UprightKeywords
	if card.Orientation == domain.Reversed {
		orient = "逆位"
		keywords = c.ReversedKeywords
	}
	pos := card.Position.NameCN
	if pos == "" {
		pos = "位置未记录"
	}
	var b strings.Builder
	writeDetail(&b, width, "牌", name)
	writeDetail(&b, width, "位置", pos+" · "+orient)
	writeDetail(&b, width, "关键词", joinOrDash(keywords))
	writeDetail(&b, width, "元素", elementDetail(c.Element))
	writeDetail(&b, width, "数字", numerologyDetail(c.Numerology))
	writeDetail(&b, width, "占星", astrologyDetail(c.Astrology))
	writeDetail(&b, width, "意象", textOrDash(c.Imagery))
	writeDetail(&b, width, "宫廷", courtDetail(c.CourtRole))
	return strings.TrimRight(b.String(), "\n")
}

func writeDetail(b *strings.Builder, width int, label, value string) {
	b.WriteString(wrapPlain(label+" "+value, width))
	b.WriteString("\n")
}

func joinOrDash(parts []string) string {
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "、")
}

func textOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func elementDetail(el string) string {
	switch strings.ToLower(strings.TrimSpace(el)) {
	case "":
		return "—"
	case "fire":
		return "火"
	case "water":
		return "水"
	case "air":
		return "风"
	case "earth":
		return "土"
	default:
		return el
	}
}

func numerologyDetail(n *domain.NumerologyInfo) string {
	if n == nil || (n.Number == 0 && n.Meaning == "" && n.Note == "") {
		return "—"
	}
	text := fmt.Sprintf("%d %s", n.Number, strings.TrimSpace(n.Meaning))
	if note := strings.TrimSpace(n.Note); note != "" {
		text += "。" + note
	}
	return strings.TrimSpace(text)
}

func astrologyDetail(a *domain.AstrologyInfo) string {
	if a == nil {
		return "—"
	}
	var parts []string
	for _, p := range []string{a.Planet, a.Zodiac, a.SubInfluence, a.Note} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "；")
}

func courtDetail(role *domain.CourtRole) string {
	if role == nil {
		return "—"
	}
	var parts []string
	for _, p := range []string{role.Archetype, role.Personality, role.AsPerson, role.AsMessage} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "；")
}

func readingPrompt(question, spreadType, cardInfo string, signals []string, prior, boundary string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "【用户描述】\n%s\n\n【牌阵】\n%s\n\n【已抽到的牌】\n%s\n\n", question, spreadType, cardInfo)
	if len(signals) > 0 {
		b.WriteString("【牌阵信号】\n")
		for _, line := range signals {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(prior) != "" {
		b.WriteString(strings.TrimSpace(prior))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(boundary) != "" {
		b.WriteString("【边界】\n")
		b.WriteString(strings.TrimSpace(boundary))
		b.WriteString("\n\n")
	}
	b.WriteString("请先查询每张牌的含义，再按系统要求的三段输出：一句话结论、逐张说明、1–2 条可执行建议。")
	if len(signals) > 0 {
		b.WriteString("牌阵信号只引用上面的列表，写进结论或逐张，不要新开章节。")
	}
	return b.String()
}
