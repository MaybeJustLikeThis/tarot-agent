package daily

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/agents"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
)

// GuideTimeout bounds the one-sentence model call.
const GuideTimeout = 20 * time.Second

// Guide turns the local card text into one sentence. Nil means the model is unavailable.
type Guide func(ctx context.Context, prompt string) (string, error)

// Options is one non-interactive daily draw.
type Options struct {
	Store   *store.Store
	Mode    agents.AgentMode
	Model   string
	Out     io.Writer
	Guide   Guide
	Now     time.Time
	Timeout time.Duration
}

// Run draws one card, prints local keywords, optionally asks the model for
// one sentence, and appends a history row. A missing model or a failed call
// still prints the card and returns nil.
func Run(ctx context.Context, opt Options) error {
	if opt.Store == nil {
		return fmt.Errorf("card store is nil")
	}
	if opt.Out == nil {
		opt.Out = io.Discard
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	result, err := tools.DrawCards(opt.Store, "single")
	if err != nil {
		return err
	}
	if len(result.Cards) == 0 {
		return fmt.Errorf("daily draw returned no card")
	}
	card := result.Cards[0]
	fmt.Fprintln(opt.Out, formatCardLine(card))
	fmt.Fprintln(opt.Out, "关键词："+keywordsOf(card))

	guidance, note := guidanceFor(ctx, opt, card)
	if guidance != "" {
		fmt.Fprintln(opt.Out, "今日一句："+guidance)
	}
	if note != "" {
		fmt.Fprintln(opt.Out, "（没有一句话指引："+note+"）")
	}

	text := guidance
	if text == "" {
		text = "本地牌义：" + keywordsOf(card)
		if note != "" {
			text += "。" + note
		}
	}
	if opt.Store.Readings == nil {
		fmt.Fprintln(opt.Out, "（这次没有写入历史：记录文件不可用）")
		return nil
	}
	reading := &domain.Reading{
		ID:             fmt.Sprintf("r_%d", opt.Now.UnixNano()),
		Question:       domain.DailyQuestion,
		SpreadID:       "single",
		Cards:          snapshot(result.Cards),
		Interpretation: text,
		Model:          opt.Model,
		CreatedAt:      opt.Now,
	}
	if guidance == "" {
		reading.Model = ""
	}
	if err := opt.Store.Readings.Append(reading); err != nil {
		fmt.Fprintf(opt.Out, "（这次没有写入历史：%s）\n", err.Error())
	}
	return nil
}

func guidanceFor(ctx context.Context, opt Options, card domain.DrawnCard) (string, string) {
	if opt.Guide == nil {
		return "", "未配置 API Key"
	}
	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = GuideTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	text, err := opt.Guide(ctx, guidePrompt(card, opt.Mode))
	if err != nil {
		return "", "模型请求失败"
	}
	line := firstLine(text)
	if line == "" {
		return "", "模型没有返回文字"
	}
	return truncate(line, 80), ""
}

func guidePrompt(card domain.DrawnCard, mode agents.AgentMode) string {
	style := "用大白话说一句，不要术语，不要预测。"
	if mode != agents.ModeCasual {
		style = "可以点一下这张牌的关键词，仍然只要一句，不要预测。"
	}
	return fmt.Sprintf("根据这一张牌给今天一句可做的小事。%s不要列清单，不要新开章节。\n牌：%s\n关键词：%s",
		style, formatCardLine(card), keywordsOf(card))
}

func formatCardLine(card domain.DrawnCard) string {
	name := strings.TrimSpace(card.Card.NameCN)
	if name == "" {
		name = card.Card.ID
	}
	if en := strings.TrimSpace(card.Card.NameEN); en != "" {
		name += " " + en
	}
	orient := "正位"
	if card.Orientation == domain.Reversed {
		orient = "逆位"
	}
	return name + " · " + orient
}

func keywordsOf(card domain.DrawnCard) string {
	words := card.Card.UprightKeywords
	if card.Orientation == domain.Reversed {
		words = card.Card.ReversedKeywords
	}
	if len(words) == 0 {
		return "（牌库没有关键词）"
	}
	return strings.Join(words, "、")
}

func snapshot(cards []domain.DrawnCard) []domain.DrawnCard {
	out := make([]domain.DrawnCard, len(cards))
	for i, card := range cards {
		out[i] = domain.DrawnCard{
			Card: domain.Card{
				ID:     card.Card.ID,
				NameCN: card.Card.NameCN,
				NameEN: card.Card.NameEN,
			},
			Orientation: card.Orientation,
			Position:    card.Position,
		}
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, " \"“”")
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
