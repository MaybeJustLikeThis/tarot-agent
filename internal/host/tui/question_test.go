package tui

import "testing"

func TestAdviseQuestion(t *testing.T) {
	cases := []struct {
		in        string
		kind      string
		suggested string
		reason    string
	}{
		{"他爱不爱我", questionYesNo, loveSuggest, loveReason},
		{"  他爱不爱我？ ", questionYesNo, loveSuggest, loveReason},
		{"他还会回来吗", questionYesNo, bondSuggest, yesNoReason},
		{"我会不会升职", questionYesNo, "关于「升职」，我现在最需要看清的是什么？", yesNoReason},
		{"我们什么时候能在一起", questionTiming, "关于这段关系，我现在最需要看清的是什么，可以怎么准备？", timingReason},
		{"我什么时候能找到工作", questionTiming, "关于「找到工作」，我现在最需要看清的是什么，可以怎么准备？", timingReason},
		{"这是命中注定的吗", questionFate, fateSuggest, fateReason},
	}
	for _, tc := range cases {
		got, ok := AdviseQuestion(tc.in)
		if !ok {
			t.Fatalf("%q: expected advice", tc.in)
		}
		if got.Kind != tc.kind || got.Suggested != tc.suggested || got.Reason != tc.reason {
			t.Fatalf("%q:\n got kind=%s suggested=%q reason=%q\nwant kind=%s suggested=%q reason=%q",
				tc.in, got.Kind, got.Suggested, got.Reason, tc.kind, tc.suggested, tc.reason)
		}
		if got.Original == "" {
			t.Fatalf("%q: original not kept", tc.in)
		}
	}
}

func TestAdviseQuestionLeavesOpenQuestions(t *testing.T) {
	for _, q := range []string{
		"我该如何面对这次分手",
		"接下来三个月该怎么走",
		"工作压力从哪来",
		"今日指引",
		"我想看看工作下一步怎么走",
	} {
		if _, ok := AdviseQuestion(q); ok {
			t.Errorf("%q: open question should not be rewritten", q)
		}
	}
}

func TestRecommendSpread(t *testing.T) {
	cases := []struct {
		in     string
		spread string
	}{
		{"升职", "single"},
		{"今天适合做什么", "single"},
		{"他爱不爱我", "single"},
		{"我想看看工作下一步怎么走", "three_card"},
		{"这段关系里，我真正在意和不安的是什么？", "three_card"},
		{"我和伴侣最近总是因为钱和未来规划吵架，接下来三个月该怎么走？", "celtic_cross"},
		{"工作和感情同时出了问题，家里的钱也不够", "celtic_cross"},
	}
	for _, tc := range cases {
		got := RecommendSpread(tc.in)
		if got.SpreadID != tc.spread || got.Reason == "" {
			t.Fatalf("%q: got %s %q, want %s", tc.in, got.SpreadID, got.Reason, tc.spread)
		}
	}
}
