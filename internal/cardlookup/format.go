package cardlookup

import (
	"fmt"
	"strings"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
)

// Format renders one card from the fields stored in the deck.
// Empty extended fields are omitted.
func Format(card domain.Card) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n", display(card.NameCN, "（无中文名）"), display(card.NameEN, "（无英文名）"))
	fmt.Fprintf(&b, "牌组：%s\n", groupLabel(card))
	fmt.Fprintf(&b, "编号：%d\n", card.Number)
	b.WriteString("\n关键词\n")
	fmt.Fprintf(&b, "正位：%s\n", joinOrDash(card.UprightKeywords))
	fmt.Fprintf(&b, "逆位：%s\n", joinOrDash(card.ReversedKeywords))
	b.WriteString("\n正位含义\n")
	b.WriteString(textOrDash(card.UprightMeaning))
	b.WriteString("\n\n逆位含义\n")
	b.WriteString(textOrDash(card.ReversedMeaning))
	b.WriteString("\n")
	writeExtra(&b, card)
	return b.String()
}

// FormatList renders every card on one line, in deck order.
func FormatList(cards []domain.Card) string {
	var b strings.Builder
	for _, card := range Sort(cards) {
		fmt.Fprintf(&b, "%s  %s  %s\n", card.NameCN, card.NameEN, groupLabel(card))
	}
	return b.String()
}

func writeExtra(b *strings.Builder, card domain.Card) {
	started := false
	line := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		if !started {
			b.WriteString("\n")
			started = true
		}
		fmt.Fprintf(b, "%s：%s\n", label, strings.TrimSpace(value))
	}
	line("元素", elementLabel(card.Element))
	line("占星", astrologyLine(card.Astrology))
	line("数字", numerologyLine(card.Numerology))
	line("意象", card.Imagery)
	line("宫廷", courtLine(card.CourtRole))
	if card.KeywordsContext != nil {
		line("爱情", strings.Join(card.KeywordsContext.Love, "、"))
		line("事业", strings.Join(card.KeywordsContext.Career, "、"))
		line("成长", strings.Join(card.KeywordsContext.Growth, "、"))
	}
}

func groupLabel(card domain.Card) string {
	if card.Arcana == domain.MajorArcana {
		return "大阿卡那"
	}
	if name := suitCN[card.Suit]; name != "" {
		return "小阿卡那 · " + name
	}
	if card.Arcana == domain.MinorArcana {
		return "小阿卡那"
	}
	if strings.TrimSpace(string(card.Arcana)) == "" {
		return "未记录"
	}
	return string(card.Arcana)
}

func elementLabel(el string) string {
	switch strings.ToLower(strings.TrimSpace(el)) {
	case "":
		return ""
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

func astrologyLine(a *domain.AstrologyInfo) string {
	if a == nil {
		return ""
	}
	var parts []string
	for _, p := range []string{a.Planet, a.Zodiac, a.SubInfluence, a.Note} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "；")
}

func numerologyLine(n *domain.NumerologyInfo) string {
	if n == nil {
		return ""
	}
	meaning := strings.TrimSpace(n.Meaning)
	note := strings.TrimSpace(n.Note)
	if n.Number == 0 && meaning == "" && note == "" {
		return ""
	}
	text := fmt.Sprintf("%d", n.Number)
	if meaning != "" {
		text += " " + meaning
	}
	if note != "" {
		text += "。" + note
	}
	return text
}

func courtLine(role *domain.CourtRole) string {
	if role == nil {
		return ""
	}
	var parts []string
	for _, p := range []string{role.Archetype, role.Personality, role.AsPerson, role.AsMessage} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "；")
}

func joinOrDash(parts []string) string {
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "、")
}

func textOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return strings.TrimSpace(s)
}

func display(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
