package tui

import (
	"strings"
	"testing"

	"github.com/voocel/tarot-agent/internal/domain"
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
