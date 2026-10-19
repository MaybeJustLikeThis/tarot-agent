package export

// Names and position meanings match the embedded spread definitions.
// A stored position description always wins; this table only fills older
// records that kept the position name and not the sentence.
var spreadNames = map[string]string{
	"single":       "单张牌",
	"three_card":   "三张牌",
	"celtic_cross": "凯尔特十字",
}

var positionMeanings = map[string]string{
	"single\x00指引": "对你当前处境的核心指引",

	"three_card\x00过去": "影响当前处境的过去因素或根源",
	"three_card\x00现在": "当前处境的核心能量或挑战",
	"three_card\x00未来": "基于当前轨迹，可能的发展方向",

	"celtic_cross\x00现状":    "你当前处境的核心",
	"celtic_cross\x00挑战":    "当前面临的主要挑战或障碍",
	"celtic_cross\x00根源":    "当前处境的深层原因或潜意识影响",
	"celtic_cross\x00过去":    "最近的过去如何影响现在",
	"celtic_cross\x00可能":    "你希望达成的目标或最好的可能",
	"celtic_cross\x00近未来":   "近期可能发生的事",
	"celtic_cross\x00自我认知":  "你如何看待自己",
	"celtic_cross\x00环境":    "周围环境和他人的影响",
	"celtic_cross\x00希望与恐惧": "你的希望或恐惧",
	"celtic_cross\x00结果":    "如果保持当前方向的最终结果",
}

func knownMeaning(spreadID, name string) string {
	if spreadID == "" || name == "" {
		return ""
	}
	return positionMeanings[spreadID+"\x00"+name]
}
