package tui

import "strings"

// QuestionAdvice is a local, optional rewrite for a closed or predictive question.
type QuestionAdvice struct {
	Kind      string // yesno, fate, timing
	Original  string
	Suggested string
	Reason    string
}

// SpreadAdvice is the spread to highlight. The user can still pick any spread.
type SpreadAdvice struct {
	SpreadID string
	Reason   string
}

const (
	questionYesNo  = "yesno"
	questionFate   = "fate"
	questionTiming = "timing"

	yesNoReason  = "是非题容易收成一句判决。改成开放式，才能看见处境和选择。"
	loveReason   = "是非题会把关系收成一句判决。改成看你的在意和不安，牌更有用。"
	fateReason   = "宿命式问法会把选择权交出去。改成看你能影响的部分，牌才照得见。"
	timingReason = "塔罗看的是当下趋势和你可以做的事，不是一个确定的日期。"

	loveSuggest    = "这段关系里，我真正在意和不安的是什么？"
	bondSuggest    = "这段关系现在的走向里，我需要看清什么，又可以怎么照顾自己？"
	fateSuggest    = "在这件事里，哪些是我能影响的，哪些需要放下？"
	timingFallback = "这件事上，我现在最需要看清的是什么，下一步可以怎么准备？"
	yesNoFallback  = "这件事里，我现在最需要看清的是什么？"

	spreadSingleReason = "问题比较集中，一张牌就够看今天的重点。"
	spreadThreeReason  = "这件事有来龙去脉，三张牌可以分开看过去、现在和接下来。"
	spreadCelticReason = "这里有好几条线缠在一起，凯尔特十字能分开看局面和阻碍。"
)

var (
	timingMarkers = []string{
		"什么时候", "何时", "哪一天", "哪天", "哪一年", "哪年", "哪个月", "多久",
		"几年后", "几个月后", "几号",
	}
	fateMarkers = []string{"命中注定", "命中", "注定", "天命", "天意", "命里", "缘分已定"}
	yesNoPairs  = []string{
		"会不会", "能不能", "是不是", "要不要", "该不该", "对不对", "好不好", "行不行",
		"有没有", "在不在", "可不可以", "是否", "爱不爱", "喜不喜欢",
	}
)

// AdviseQuestion suggests an open rewrite for yes/no, fate, and timing questions.
// Other questions return false and should continue unchanged.
func AdviseQuestion(raw string) (QuestionAdvice, bool) {
	q := strings.TrimSpace(raw)
	if q == "" {
		return QuestionAdvice{}, false
	}
	switch {
	case containsAny(q, timingMarkers...):
		return QuestionAdvice{
			Kind:      questionTiming,
			Original:  q,
			Suggested: suggestTiming(q),
			Reason:    timingReason,
		}, true
	case containsAny(q, fateMarkers...):
		return QuestionAdvice{
			Kind:      questionFate,
			Original:  q,
			Suggested: suggestFate(q),
			Reason:    fateReason,
		}, true
	case isYesNo(q):
		suggested, reason := suggestYesNo(q)
		return QuestionAdvice{
			Kind:      questionYesNo,
			Original:  q,
			Suggested: suggested,
			Reason:    reason,
		}, true
	default:
		return QuestionAdvice{}, false
	}
}

// RecommendSpread highlights a spread for a confirmed question.
// Simple questions stay on one or three cards. Celtic cross is only for
// situations with several threads at once.
func RecommendSpread(raw string) SpreadAdvice {
	q := strings.TrimSpace(raw)
	switch {
	case isComplexQuestion(q):
		return SpreadAdvice{SpreadID: "celtic_cross", Reason: spreadCelticReason}
	case isBriefQuestion(q):
		return SpreadAdvice{SpreadID: "single", Reason: spreadSingleReason}
	default:
		return SpreadAdvice{SpreadID: "three_card", Reason: spreadThreeReason}
	}
}

func suggestTiming(q string) string {
	if containsAny(q, "在一起", "复合", "结婚", "恋爱", "爱我", "爱情") {
		return "关于这段关系，我现在最需要看清的是什么，可以怎么准备？"
	}
	if topic := tidyTopic(stripAll(q, timingMarkers...)); topic != "" {
		return "关于「" + topic + "」，我现在最需要看清的是什么，可以怎么准备？"
	}
	return timingFallback
}

func suggestFate(q string) string {
	if topic := tidyTopic(stripAll(q, append(fateMarkers, yesNoPairs...)...)); topic != "" {
		return "关于「" + topic + "」，哪些是我能影响的，哪些需要放下？"
	}
	return fateSuggest
}

func suggestYesNo(q string) (string, string) {
	if isLoveQuestion(q) {
		return loveSuggest, loveReason
	}
	if containsAny(q, "回来", "复合", "分手", "离开") {
		return bondSuggest, yesNoReason
	}
	if topic := tidyTopic(stripYesNo(q)); topic != "" {
		return "关于「" + topic + "」，我现在最需要看清的是什么？", yesNoReason
	}
	return yesNoFallback, yesNoReason
}

func isYesNo(q string) bool {
	if containsAny(q, yesNoPairs...) {
		return true
	}
	return endsYesNoParticle(q)
}

func isLoveQuestion(q string) bool {
	if strings.Contains(q, "爱不爱") || strings.Contains(q, "爱我") || strings.Contains(q, "还爱") {
		return true
	}
	return strings.Contains(q, "喜欢我")
}

func endsYesNoParticle(q string) bool {
	s := strings.TrimRight(q, "？?。！!… \t")
	if strings.HasSuffix(s, "吗") {
		return true
	}
	if !strings.HasSuffix(s, "么") {
		return false
	}
	r := []rune(s)
	if len(r) >= 2 {
		switch r[len(r)-2] {
		case '怎', '那', '什', '要', '多':
			return false
		}
	}
	return true
}

func isBriefQuestion(q string) bool {
	n := len([]rune(q))
	if n == 0 {
		return true
	}
	if n <= 8 {
		return true
	}
	return n <= 16 && containsAny(q, "今天", "今日", "此刻", "眼下")
}

func isComplexQuestion(q string) bool {
	hits := 0
	for _, w := range []string{"同时", "一方面", "另一方面", "既要", "又要", "还要", "以及", "另外", "多方", "交织", "好几"} {
		if strings.Contains(q, w) {
			hits++
		}
	}
	switch strings.Count(q, "和") {
	case 0:
	case 1:
		if len([]rune(q)) >= 18 {
			hits++
		}
	default:
		hits += 2
	}
	if strings.Count(q, "，")+strings.Count(q, ",") >= 2 && len([]rune(q)) >= 24 {
		hits++
	}
	domains := 0
	groups := [][]string{
		{"工作", "事业", "职业", "同事", "老板"},
		{"感情", "伴侣", "对象", "恋爱", "婚姻", "老公", "老婆", "男朋友", "女朋友"},
		{"钱", "财务", "收入", "房贷", "经济"},
		{"家人", "父母", "孩子", "家庭"},
		{"健康", "身体"},
	}
	for _, g := range groups {
		if containsAny(q, g...) {
			domains++
		}
	}
	if domains >= 2 {
		hits += 2
	}
	return hits >= 2
}

func stripYesNo(q string) string {
	s := stripAll(q, yesNoPairs...)
	s = strings.TrimRight(s, "吗么")
	return s
}

func stripAll(q string, markers ...string) string {
	s := q
	for _, m := range markers {
		s = strings.ReplaceAll(s, m, "")
	}
	return s
}

func tidyTopic(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []string{"请问", "想问一下", "想问问", "想问", "我想知道", "我想问", "帮我看看", "帮我看"} {
		s = strings.TrimPrefix(s, p)
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "我") && !strings.HasPrefix(s, "我们") && !strings.HasPrefix(s, "我的") {
		s = strings.TrimPrefix(s, "我")
	}
	s = strings.Trim(s, "的了吗呢吧啊呀么？?。！!，,、 能会要是不")
	s = strings.TrimSpace(s)
	n := len([]rune(s))
	if n < 2 || n > 18 {
		return ""
	}
	return s
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if p != "" && strings.Contains(s, p) {
			return true
		}
	}
	return false
}
