package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// DailyQuestion is the stored question for both the ctrl+d draw and `tarot-agent daily`.
// Similar-question matching skips it.
const DailyQuestion = "今日指引"

// SimilarLookback is how many recent readings are considered.
const SimilarLookback = 20

// similarMinScore is the Dice coefficient of character bigrams required for a match.
// 1 means the same bigrams, 0 means none in common.
const similarMinScore = 0.45

// PriorMatch is the newest recent reading that looks like the same question.
type PriorMatch struct {
	Reading Reading
	Score   float64
}

// FindSimilarQuestion returns the newest reading in the last SimilarLookback
// entries whose question shares enough character bigrams with question.
// Daily-card rows and rows without cards are skipped. Nil means no match.
func FindSimilarQuestion(question string, history []Reading) *PriorMatch {
	q := strings.TrimSpace(question)
	if q == "" || q == DailyQuestion {
		return nil
	}
	qb := bigrams(q)
	if len(qb) == 0 {
		return nil
	}
	start := 0
	if len(history) > SimilarLookback {
		start = len(history) - SimilarLookback
	}
	for i := len(history) - 1; i >= start; i-- {
		reading := history[i]
		if !usablePrior(reading) {
			continue
		}
		score := dice(qb, bigrams(reading.Question))
		if score >= similarMinScore {
			return &PriorMatch{Reading: reading, Score: score}
		}
	}
	return nil
}

func usablePrior(reading Reading) bool {
	if strings.TrimSpace(reading.Question) == "" || strings.TrimSpace(reading.Question) == DailyQuestion {
		return false
	}
	return len(reading.Cards) > 0
}

// AdviceExcerpt returns the 可执行建议 section, trimmed to maxRunes.
// Readings that never used that heading contribute nothing.
func AdviceExcerpt(interpretation string, maxRunes int) string {
	text := strings.ReplaceAll(interpretation, "\r\n", "\n")
	idx := strings.Index(text, "可执行建议")
	if idx < 0 {
		return ""
	}
	text = text[idx+len("可执行建议"):]
	if next := strings.Index(text, "\n###"); next >= 0 {
		text = text[:next]
	}
	text = strings.Trim(text, "# \t\n：:")
	text = strings.Join(strings.Fields(text), " ")
	return truncateRunes(text, maxRunes)
}

// BriefCards joins up to limit card names. Extra cards are counted, not listed.
func BriefCards(cards []DrawnCard, limit int) string {
	if len(cards) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(cards) {
		limit = len(cards)
	}
	parts := make([]string, 0, limit)
	for _, card := range cards[:limit] {
		name := strings.TrimSpace(card.Card.NameCN)
		if name == "" {
			name = strings.TrimSpace(card.Card.ID)
		}
		if name == "" {
			name = "未命名"
		}
		parts = append(parts, name+orientationLabel(card.Orientation))
	}
	text := strings.Join(parts, "、")
	if len(cards) > limit {
		text += fmt.Sprintf("等 %d 张", len(cards))
	}
	return text
}

func orientationLabel(o Orientation) string {
	switch o {
	case Reversed:
		return "逆位"
	case Upright:
		return "正位"
	default:
		return ""
	}
}

// FormatPriorBlock is the bounded section added to a reading prompt.
func FormatPriorBlock(match *PriorMatch) string {
	if match == nil || len(match.Reading.Cards) == 0 {
		return ""
	}
	reading := match.Reading
	var b strings.Builder
	b.WriteString("【上次类似问题】\n")
	date := "日期未记录"
	if !reading.CreatedAt.IsZero() {
		date = reading.CreatedAt.Format("2006-01-02")
	}
	fmt.Fprintf(&b, "日期：%s\n", date)
	fmt.Fprintf(&b, "问题：%s\n", truncateRunes(strings.TrimSpace(reading.Question), 40))
	fmt.Fprintf(&b, "牌面：%s\n", BriefCards(reading.Cards, 10))
	if advice := AdviceExcerpt(reading.Interpretation, 80); advice != "" {
		fmt.Fprintf(&b, "当时的可执行建议：%s\n", advice)
	}
	b.WriteString("有这份资料时，在结论里用一句话对照变化，不要新开章节。\n")
	return b.String()
}

func bigrams(s string) map[string]struct{} {
	runes := normalizeRunes(s)
	if len(runes) < 2 {
		return nil
	}
	out := make(map[string]struct{}, len(runes)-1)
	for i := 0; i+1 < len(runes); i++ {
		out[string(runes[i:i+2])] = struct{}{}
	}
	return out
}

func normalizeRunes(s string) []rune {
	var out []rune
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return out
}

func dice(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for g := range a {
		if _, ok := b[g]; ok {
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(a)+len(b))
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
