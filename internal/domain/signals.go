package domain

import (
	"fmt"
	"strings"
)

// SpreadSignals lists factual patterns in a draw.
// A one-card spread returns nothing. A fact is included only when the cards
// support it; even counts and single occurrences are left out.
func SpreadSignals(cards []DrawnCard) []string {
	if len(cards) < 2 {
		return nil
	}
	var out []string
	out = append(out, elementSignals(cards)...)
	out = append(out, majorSignal(cards)...)
	out = append(out, numberSignals(cards)...)
	out = append(out, courtSignals(cards)...)
	out = append(out, suitSignals(cards)...)
	out = append(out, reversedSignal(cards)...)
	return out
}

func elementSignals(cards []DrawnCard) []string {
	counts := map[string]int{}
	for _, c := range cards {
		el := strings.ToLower(strings.TrimSpace(c.Card.Element))
		if el == "" {
			continue
		}
		counts[el]++
	}
	if len(counts) == 0 {
		return nil
	}
	var out []string
	maxN := 0
	for _, n := range counts {
		if n > maxN {
			maxN = n
		}
	}
	if maxN >= 2 {
		order := []string{"fire", "water", "air", "earth"}
		var present, missing []string
		for _, el := range order {
			name := elementName(el)
			if counts[el] == 0 {
				missing = append(missing, name)
				continue
			}
			present = append(present, fmt.Sprintf("%s %d 张", name, counts[el]))
		}
		line := "元素：" + strings.Join(present, "，")
		if len(missing) > 0 {
			var absent []string
			for _, name := range missing {
				absent = append(absent, "没有"+name)
			}
			line += "；" + strings.Join(absent, "、")
		}
		out = append(out, line)
	}
	if counts["fire"] > 0 && counts["water"] > 0 {
		out = append(out, "火与水同时出现")
	}
	if counts["air"] > 0 && counts["earth"] > 0 {
		out = append(out, "风与土同时出现")
	}
	return out
}

func majorSignal(cards []DrawnCard) []string {
	majors := 0
	for _, c := range cards {
		if c.Card.Arcana == MajorArcana {
			majors++
		}
	}
	total := len(cards)
	if majors >= 2 && majors*2 >= total {
		return []string{fmt.Sprintf("大牌 %d/%d", majors, total)}
	}
	if majors == 0 && total >= 3 {
		return []string{"没有大牌"}
	}
	return nil
}

func numberSignals(cards []DrawnCard) []string {
	counts := map[int]int{}
	for _, c := range cards {
		// Court ranks are stored in Number, but they are not the pip/major
		// numerology. Justice is 11; a Page is also 11 and must not count.
		if isCourtCard(c.Card) {
			continue
		}
		counts[c.Card.Number]++
	}
	var out []string
	for n := 0; n <= 21; n++ {
		if counts[n] >= 2 {
			out = append(out, fmt.Sprintf("数字 %d 出现 %d 次", n, counts[n]))
		}
	}
	return out
}

func courtSignals(cards []DrawnCard) []string {
	n := 0
	for _, c := range cards {
		if isCourtCard(c.Card) {
			n++
		}
	}
	if n >= 2 {
		return []string{fmt.Sprintf("宫廷牌 %d 张", n)}
	}
	return nil
}

func isCourtCard(c Card) bool {
	return c.CourtRole != nil
}

func suitSignals(cards []DrawnCard) []string {
	counts := map[Suit]int{}
	for _, c := range cards {
		if c.Card.Suit == "" {
			continue
		}
		counts[c.Card.Suit]++
	}
	var out []string
	for _, suit := range []Suit{SuitWands, SuitCups, SuitSwords, SuitPentacles} {
		if counts[suit] >= 2 {
			out = append(out, fmt.Sprintf("%s %d 张", suitName(suit), counts[suit]))
		}
	}
	return out
}

func reversedSignal(cards []DrawnCard) []string {
	rev := 0
	for _, c := range cards {
		if c.Orientation == Reversed {
			rev++
		}
	}
	total := len(cards)
	switch {
	case rev == total && total >= 2:
		return []string{fmt.Sprintf("全部逆位（%d 张）", rev)}
	case rev == 0 && total >= 3:
		return []string{"全部正位"}
	case rev >= 2:
		return []string{fmt.Sprintf("逆位 %d 张", rev)}
	default:
		return nil
	}
}

func elementName(el string) string {
	switch el {
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

func suitName(suit Suit) string {
	switch suit {
	case SuitWands:
		return "权杖"
	case SuitCups:
		return "圣杯"
	case SuitSwords:
		return "宝剑"
	case SuitPentacles:
		return "星币"
	default:
		return string(suit)
	}
}
