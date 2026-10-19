package domain

import (
	"strings"
	"testing"
	"time"
)

func reading(question string, when time.Time, cards []DrawnCard, interpretation string) Reading {
	return Reading{
		Question:       question,
		CreatedAt:      when,
		Cards:          cards,
		Interpretation: interpretation,
		SpreadID:       "three_card",
	}
}

func named(name string, orient Orientation) DrawnCard {
	return DrawnCard{
		Card:        Card{NameCN: name},
		Orientation: orient,
		Position:    Position{NameCN: "现在"},
	}
}

func TestFindSimilarQuestion(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	older := reading("我和伴侣最近总是因为钱吵架", day, []DrawnCard{named("女皇", Reversed)}, "### 可执行建议\n本周把开支写下来。")
	newer := reading("我和伴侣最近还是因为钱吵架", day.AddDate(0, 0, 3), []DrawnCard{named("战车", Upright)}, "### 可执行建议\n先约一次不谈对错的谈话。")
	unrelated := reading("今天午餐吃什么", day.AddDate(0, 0, 1), []DrawnCard{named("愚者", Upright)}, "")
	daily := reading(DailyQuestion, day.AddDate(0, 0, 4), []DrawnCard{named("太阳", Upright)}, "一句指引")
	noCards := reading("我和伴侣最近总是因为钱吵架", day.AddDate(0, 0, 5), nil, "### 可执行建议\n不该被用到。")

	history := []Reading{older, unrelated, newer, daily, noCards}
	got := FindSimilarQuestion("我和伴侣因为钱吵架，接下来怎么办", history)
	if got == nil {
		t.Fatal("expected a match")
	}
	if got.Reading.CreatedAt != newer.CreatedAt || !strings.Contains(BriefCards(got.Reading.Cards, 3), "战车正位") {
		t.Fatalf("newest similar reading not chosen: %+v", got.Reading)
	}
	if FindSimilarQuestion("今天的工作重点是什么", history) != nil {
		t.Fatal("unrelated question matched")
	}
	if FindSimilarQuestion(DailyQuestion, history) != nil {
		t.Fatal("daily question should not search history")
	}
}

func TestFindSimilarSkipsOldBeyondLookback(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var history []Reading
	history = append(history, reading("我和伴侣最近总是因为钱吵架", day, []DrawnCard{named("女皇", Reversed)}, ""))
	for i := 0; i < SimilarLookback; i++ {
		history = append(history, reading("完全不同的题目"+string(rune('a'+i%26)), day.AddDate(0, 0, i+1), []DrawnCard{named("愚者", Upright)}, ""))
	}
	if got := FindSimilarQuestion("我和伴侣最近总是因为钱吵架", history); got != nil {
		t.Fatalf("match outside the lookback: %+v", got.Reading.Question)
	}
}

func TestAdviceExcerptAndPriorBlock(t *testing.T) {
	text := "### 逐张\n很多内容。\n### 可执行建议\n本周列出三件满意的小事，并且不要再追加。\n### 其他\n忽略"
	got := AdviceExcerpt(text, 12)
	if !strings.HasPrefix(got, "本周列出三件") || strings.Contains(got, "忽略") {
		t.Fatalf("excerpt %q", got)
	}
	if AdviceExcerpt("只有正文，没有那个小节", 40) != "" {
		t.Fatal("missing heading should not invent an excerpt")
	}

	match := FindSimilarQuestion("我和伴侣因为钱吵架", []Reading{
		reading("我和伴侣最近总是因为钱吵架", time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), []DrawnCard{named("女皇", Reversed)}, text),
	})
	block := FormatPriorBlock(match)
	for _, want := range []string{"【上次类似问题】", "2026-10-02", "女皇逆位", "本周列出三件", "一句话对照", "不要新开章节"} {
		if !strings.Contains(block, want) {
			t.Errorf("block missing %q\n%s", want, block)
		}
	}
	if FormatPriorBlock(nil) != "" {
		t.Fatal("nil match produced a block")
	}
}

func TestBriefCardsTruncates(t *testing.T) {
	cards := []DrawnCard{named("女皇", Reversed), named("战车", Upright), named("太阳", Upright), named("星星", Upright)}
	got := BriefCards(cards, 3)
	if got != "女皇逆位、战车正位、太阳正位等 4 张" {
		t.Fatalf("brief %q", got)
	}
}
