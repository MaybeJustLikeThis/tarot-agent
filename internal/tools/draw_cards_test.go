package tools_test

import (
	"testing"

	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/store"
	"github.com/voocel/tarot-agent/internal/tools"
)

func TestDrawCards_NoDuplicates(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	for run := 0; run < 100; run++ {
		result, err := tools.DrawCards(s, "celtic_cross")
		if err != nil {
			t.Fatalf("draw failed on run %d: %v", run, err)
		}

		seen := make(map[string]bool)
		for _, dc := range result.Cards {
			if seen[dc.Card.ID] {
				t.Errorf("run %d: duplicate card %s", run, dc.Card.ID)
			}
			seen[dc.Card.ID] = true
		}
	}
}

func TestDrawCards_Orientation_Random(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	uprightCount := 0
	total := 1000

	for i := 0; i < total; i++ {
		result, err := tools.DrawCards(s, "single")
		if err != nil {
			t.Fatalf("draw failed: %v", err)
		}
		if result.Cards[0].Orientation == domain.Upright {
			uprightCount++
		}
	}

	ratio := float64(uprightCount) / float64(total)
	if ratio < 0.35 || ratio > 0.65 {
		t.Errorf("upright ratio %.2f outside expected range [0.35, 0.65]", ratio)
	}
}

func TestDrawCards_SingleCard(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	result, err := tools.DrawCards(s, "single")
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if len(result.Cards) != 1 {
		t.Errorf("expected 1 card, got %d", len(result.Cards))
	}
	if result.SpreadID != "single" {
		t.Errorf("expected spread 'single', got '%s'", result.SpreadID)
	}
}

func TestDrawCards_ThreeCard(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	result, err := tools.DrawCards(s, "three_card")
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if len(result.Cards) != 3 {
		t.Errorf("expected 3 cards, got %d", len(result.Cards))
	}
}

func TestDrawCards_CelticCross(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	result, err := tools.DrawCards(s, "celtic_cross")
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if len(result.Cards) != 10 {
		t.Errorf("expected 10 cards, got %d", len(result.Cards))
	}
}

func TestDrawCards_InvalidSpread(t *testing.T) {
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}

	_, err = tools.DrawCards(s, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent spread")
	}
}
