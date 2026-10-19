// Package export renders a stored reading as Markdown or plain text.
// It does not import the TUI, so the same text can be tested and written from the CLI.
package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
)

const (
	FormatMarkdown = "md"
	FormatText     = "txt"
)

// Render turns one reading into Markdown (md) or plain text (txt).
func Render(r domain.Reading, format string) (string, error) {
	switch normalizeFormat(format) {
	case FormatMarkdown:
		return markdown(r), nil
	case FormatText:
		return plain(r), nil
	default:
		return "", fmt.Errorf("不认识的导出格式 %q，可用 md 或 txt", format)
	}
}

func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "md", "markdown":
		return FormatMarkdown
	case "txt", "text", "plain":
		return FormatText
	default:
		return ""
	}
}

func markdown(r domain.Reading) string {
	var b strings.Builder
	b.WriteString("# 占卜记录\n\n")
	fmt.Fprintf(&b, "- 时间：%s\n", displayTime(r.CreatedAt))
	fmt.Fprintf(&b, "- 模型：%s\n", displayOr(r.Model, "未记录"))
	fmt.Fprintf(&b, "- 问题：%s\n", displayOr(r.Question, "（无）"))
	fmt.Fprintf(&b, "- 牌阵：%s\n\n", spreadName(r.SpreadID))
	b.WriteString("## 牌面\n\n")
	writeCards(&b, r, true)
	b.WriteString("## 解读\n\n")
	b.WriteString(bodyOr(r.Interpretation, "（无解读正文）"))
	b.WriteString("\n\n")
	if len(r.FollowUps) > 0 {
		b.WriteString("## 追问\n\n")
		for _, turn := range r.FollowUps {
			fmt.Fprintf(&b, "**%s**%s\n\n", turnLabel(turn.Role), timeSuffix(turn.CreatedAt))
			b.WriteString(bodyOr(turn.Content, "（空）"))
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## 免责声明\n\n")
	b.WriteString(disclaimer(r))
	b.WriteString("\n")
	return b.String()
}

func plain(r domain.Reading) string {
	var b strings.Builder
	b.WriteString("占卜记录\n\n")
	fmt.Fprintf(&b, "时间：%s\n", displayTime(r.CreatedAt))
	fmt.Fprintf(&b, "模型：%s\n", displayOr(r.Model, "未记录"))
	fmt.Fprintf(&b, "问题：%s\n", displayOr(r.Question, "（无）"))
	fmt.Fprintf(&b, "牌阵：%s\n\n", spreadName(r.SpreadID))
	b.WriteString("牌面\n\n")
	writeCards(&b, r, false)
	b.WriteString("解读\n\n")
	b.WriteString(bodyOr(r.Interpretation, "（无解读正文）"))
	b.WriteString("\n\n")
	if len(r.FollowUps) > 0 {
		b.WriteString("追问\n\n")
		for _, turn := range r.FollowUps {
			fmt.Fprintf(&b, "%s%s\n", turnLabel(turn.Role), timeSuffix(turn.CreatedAt))
			b.WriteString(bodyOr(turn.Content, "（空）"))
			b.WriteString("\n\n")
		}
	}
	b.WriteString("免责声明\n\n")
	b.WriteString(disclaimer(r))
	b.WriteString("\n")
	return b.String()
}

func writeCards(b *strings.Builder, r domain.Reading, markdown bool) {
	if len(r.Cards) == 0 {
		b.WriteString("（未保存牌面）\n\n")
		return
	}
	for i, card := range r.Cards {
		name, meaning := positionDetail(r.SpreadID, card.Position)
		title := cardTitle(card)
		if markdown {
			fmt.Fprintf(b, "%d. **%s** — %s\n", i+1, name, title)
			fmt.Fprintf(b, "   - 位置含义：%s\n", meaning)
		} else {
			fmt.Fprintf(b, "%d. %s — %s\n", i+1, name, title)
			fmt.Fprintf(b, "   位置含义：%s\n", meaning)
		}
	}
	b.WriteString("\n")
}

func cardTitle(card domain.DrawnCard) string {
	name := strings.TrimSpace(card.Card.NameCN)
	if name == "" {
		name = strings.TrimSpace(card.Card.ID)
	}
	if name == "" {
		name = "未知牌"
	}
	if en := strings.TrimSpace(card.Card.NameEN); en != "" {
		name += "（" + en + "）"
	}
	return name + " · " + orientationLabel(card.Orientation)
}

func orientationLabel(o domain.Orientation) string {
	switch o {
	case domain.Upright:
		return "正位"
	case domain.Reversed:
		return "逆位"
	case "":
		return "正逆位未记录"
	default:
		return string(o)
	}
}

func positionDetail(spreadID string, pos domain.Position) (string, string) {
	name := strings.TrimSpace(pos.NameCN)
	if name == "" {
		name = strings.TrimSpace(pos.NameEN)
	}
	if name == "" {
		name = "位置未记录"
	}
	meaning := strings.TrimSpace(pos.Description)
	if meaning == "" {
		meaning = knownMeaning(spreadID, strings.TrimSpace(pos.NameCN))
	}
	if meaning == "" {
		meaning = "未记录"
	}
	return name, meaning
}

func spreadName(id string) string {
	if name, ok := spreadNames[id]; ok {
		return name
	}
	if strings.TrimSpace(id) == "" {
		return "未记录"
	}
	return id
}

func turnLabel(role string) string {
	switch role {
	case "user":
		return "你"
	case "assistant":
		return "助手"
	default:
		if strings.TrimSpace(role) == "" {
			return "未知"
		}
		return role
	}
}

func displayTime(t time.Time) string {
	if t.IsZero() {
		return "未记录"
	}
	return t.Format("2006-01-02 15:04")
}

func timeSuffix(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "（" + t.Format("2006-01-02 15:04") + "）"
}

func disclaimer(r domain.Reading) string {
	if text := strings.TrimSpace(r.Disclaimer); text != "" {
		return text
	}
	return strings.TrimSpace(tools.DisclaimerText())
}

func displayOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func bodyOr(value, fallback string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return fallback
	}
	return text
}
