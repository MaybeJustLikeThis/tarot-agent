package tui

import (
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

func TestFormatDrawnCards_IncludesIDOrientationAndPosition(t *testing.T) {
	cards := []domain.DrawnCard{
		{
			Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
			Orientation: domain.Upright,
		},
		{
			Card:        domain.Card{ID: "minor_wands_01", NameCN: "权杖Ace", NameEN: "Ace of Wands"},
			Orientation: domain.Reversed,
		},
	}

	got := formatDrawnCards(cards, "three_card")
	for _, want := range []string{
		"位置 1（过去）",
		"愚者",
		"[正位]",
		"The Fool",
		"id: major_00",
		"位置 2（现在）",
		"[逆位]",
		"权杖Ace",
		"id: minor_wands_01",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("formatDrawnCards missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestNarrowCardOmitsPartialEnglish(t *testing.T) {
	card := domain.DrawnCard{
		Card:        domain.Card{NameCN: "星币骑士", NameEN: "Knight of Pentacles"},
		Orientation: domain.Upright,
	}
	narrow := renderCardFrontSized(card, 12)
	if strings.Contains(narrow, "Knight") || strings.Contains(narrow, "Pent") || strings.Contains(narrow, "…") && strings.Contains(narrow, "of") {
		t.Fatalf("partial english:\n%s", narrow)
	}
	if !strings.Contains(narrow, "星币骑士") || !strings.Contains(narrow, "正位") {
		t.Fatalf("missing chinese face:\n%s", narrow)
	}
	if lipgloss.Width(narrow) > 12 {
		t.Fatalf("card width %d > 12\n%s", lipgloss.Width(narrow), narrow)
	}

	wide := renderCardFrontSized(card, 28)
	if !strings.Contains(wide, "Knight of Pentacles") || !strings.Contains(wide, "星币骑士") {
		t.Fatalf("wide card dropped a full name:\n%s", wide)
	}
	if lipgloss.Width(wide) > 28 {
		t.Fatalf("wide card width %d", lipgloss.Width(wide))
	}

	tiny := renderCardFrontSized(card, 8)
	if strings.Contains(tiny, "Knight") || strings.Contains(tiny, "↑") {
		t.Fatalf("tiny card should keep only the chinese name and orientation:\n%s", tiny)
	}
	if lipgloss.Width(tiny) > 8 {
		t.Fatalf("tiny width %d\n%s", lipgloss.Width(tiny), tiny)
	}
}

func TestThreeCardShortPanelKeepsFrames(t *testing.T) {
	cards := []domain.DrawnCard{
		{Card: domain.Card{NameCN: "星币骑士", NameEN: "Knight of Pentacles"}, Orientation: domain.Upright},
		{Card: domain.Card{NameCN: "圣杯王后", NameEN: "Queen of Cups"}, Orientation: domain.Upright},
		{Card: domain.Card{NameCN: "宝剑骑士", NameEN: "Knight of Swords"}, Orientation: domain.Reversed},
	}
	fitted := renderSpreadForPanel(cards, 3, "three_card", 28, 9)
	if !strings.Contains(fitted, "╔") {
		t.Fatalf("short panel dropped frames:\n%s", fitted)
	}
	if strings.Contains(fitted, "Knight") {
		t.Fatalf("narrow frames kept a partial english name:\n%s", fitted)
	}
	if lipgloss.Height(fitted) > 9 {
		t.Fatalf("fitted height %d", lipgloss.Height(fitted))
	}
	for _, lab := range []string{"过去", "现在", "未来", "星币", "正位"} {
		if !strings.Contains(fitted, lab) {
			t.Errorf("missing %s\n%s", lab, fitted)
		}
	}
}
