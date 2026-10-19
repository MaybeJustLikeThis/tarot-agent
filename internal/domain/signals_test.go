package domain

import (
	"strings"
	"testing"
)

func card(arcana ArcanaType, suit Suit, number int, element string, orient Orientation) DrawnCard {
	return DrawnCard{
		Card:        Card{Arcana: arcana, Suit: suit, Number: number, Element: element},
		Orientation: orient,
	}
}

func TestSpreadSignalsEmptyWithoutEvidence(t *testing.T) {
	one := []DrawnCard{card(MajorArcana, "", 0, "air", Upright)}
	if got := SpreadSignals(one); len(got) != 0 {
		t.Fatalf("single card signals = %v", got)
	}
	mixed := []DrawnCard{
		card(MajorArcana, "", 0, "air", Upright),
		card(MinorArcana, SuitWands, 2, "fire", Reversed),
		card(MinorArcana, SuitCups, 3, "water", Upright),
	}
	got := SpreadSignals(mixed)
	for _, line := range got {
		if line == "元素：火 1 张，水 1 张，风 1 张；没有土" || line == "逆位 1 张" {
			t.Fatalf("weak pattern should be omitted, got %v", got)
		}
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "火与水同时出现") {
		t.Fatalf("opposition missing: %v", got)
	}
	if strings.Contains(joined, "没有大牌") || strings.Contains(joined, "大牌") {
		t.Fatalf("one major of three is not a majority: %v", got)
	}
}

func TestSpreadSignalsReportsSupportedPatterns(t *testing.T) {
	cards := []DrawnCard{
		card(MajorArcana, "", 3, "fire", Reversed),
		card(MajorArcana, "", 7, "fire", Reversed),
		card(MinorArcana, SuitWands, 3, "water", Upright),
		card(MinorArcana, SuitWands, 8, "earth", Upright),
	}
	got := strings.Join(SpreadSignals(cards), "\n")
	for _, want := range []string{
		"火 2 张",
		"没有风",
		"火与水同时出现",
		"大牌 2/4",
		"数字 3 出现 2 次",
		"权杖 2 张",
		"逆位 2 张",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestSpreadSignalsAllMinorAndUpright(t *testing.T) {
	cards := []DrawnCard{
		card(MinorArcana, SuitCups, 1, "water", Upright),
		card(MinorArcana, SuitCups, 2, "water", Upright),
		card(MinorArcana, SuitSwords, 4, "air", Upright),
	}
	got := strings.Join(SpreadSignals(cards), "\n")
	for _, want := range []string{"没有大牌", "全部正位", "圣杯 2 张", "水 2 张", "没有火", "没有土"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestSpreadSignalsAirEarthAndAllReversed(t *testing.T) {
	cards := []DrawnCard{
		card(MinorArcana, SuitSwords, 5, "air", Reversed),
		card(MinorArcana, SuitPentacles, 5, "earth", Reversed),
	}
	got := strings.Join(SpreadSignals(cards), "\n")
	for _, want := range []string{"风与土同时出现", "全部逆位（2 张）", "数字 5 出现 2 次"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "元素：") || strings.Contains(got, "没有大牌") {
		t.Errorf("two different elements and no majors should stay quiet: %s", got)
	}
}
