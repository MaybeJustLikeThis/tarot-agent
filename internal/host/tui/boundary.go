package tui

import "strings"

// Boundary is a local check before a reading starts.
// Crisis stops the draw. Health and legal ask for a confirmation.
// HealthLight is a one-line note and does not add a step.
type Boundary struct {
	Kind string
	Note string
}

const (
	boundaryNone        = ""
	boundaryHealth      = "health"
	boundaryLegal       = "legal"
	boundaryCrisis      = "crisis"
	boundaryHealthLight = "health_light"

	healthNote = "塔罗不能代替医生。这里不给诊断，也不建议用药；身体上的判断请交给医生。可以继续看牌。"
	legalNote  = "塔罗不能代替律师。这里不给法律结论；权利和义务请咨询律师。可以继续看牌。"
	// healthLightNote stays one line on an 80-column screen.
	healthLightNote = "身体上的具体问题请听医生的。这里只当闲谈，不拦这次占卜。"
	crisisNote      = "我听到你现在很痛苦。塔罗没法代替帮助。\n如果情况紧急，请拨打 120 或 110。\n也可以联系当地的心理援助热线，或告诉一个你信任的人。\n按 esc 返回，先不抽牌。"
)

// ClassifyQuestion labels a question. Crisis wins over health and legal.
// General wellness wording such as 身体健康运势 is only a light note.
func ClassifyQuestion(raw string) Boundary {
	q := strings.TrimSpace(raw)
	if q == "" {
		return Boundary{}
	}
	switch {
	case isCrisisQuestion(q):
		return Boundary{Kind: boundaryCrisis, Note: crisisNote}
	case isHealthQuestion(q):
		return Boundary{Kind: boundaryHealth, Note: healthNote}
	case isLegalQuestion(q):
		return Boundary{Kind: boundaryLegal, Note: legalNote}
	case isHealthLight(q):
		return Boundary{Kind: boundaryHealthLight, Note: healthLightNote}
	default:
		return Boundary{}
	}
}

// PromptLine is the instruction added to the model prompt. Crisis never draws.
func (b Boundary) PromptLine() string {
	switch b.Kind {
	case boundaryHealth, boundaryHealthLight:
		return "这是健康相关的问题。不要给诊断或用药建议，提醒不能代替医生。写进现有三段，不要新开章节。"
	case boundaryLegal:
		return "这是法律相关的问题。不要给法律结论，提醒不能代替律师。写进现有三段，不要新开章节。"
	default:
		return ""
	}
}

func isCrisisQuestion(q string) bool {
	if containsAny(q, "自杀", "轻生", "自残", "割腕", "不想活", "活不下去", "结束生命", "结束自己", "想结束一切") {
		return true
	}
	if strings.Contains(q, "想死") && !strings.Contains(q, "想死了") && !strings.Contains(q, "想死你") {
		return true
	}
	return false
}

func isHealthQuestion(q string) bool {
	return containsAny(q,
		"抑郁症", "抑郁", "焦虑症", "癌症", "肿瘤", "确诊", "诊断", "处方",
		"用药", "吃药", "停药", "手术", "流产", "病情", "生病", "病了",
		"去医院", "看医生", "住院", "症状", "发烧",
	)
}

func isLegalQuestion(q string) bool {
	if containsAny(q, "起诉", "官司", "律师", "判刑", "合法吗", "违法", "抚养权", "离婚协议", "告上法庭", "犯罪", "违约") {
		return true
	}
	return strings.Contains(q, "合同") && containsAny(q, "签", "纠纷", "条款")
}

func isHealthLight(q string) bool {
	return containsAny(q, "健康运势", "身体健康", "身体运势", "养生运势")
}
