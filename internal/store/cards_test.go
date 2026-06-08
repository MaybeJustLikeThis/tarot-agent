package store_test

import (
	"testing"

	"github.com/voocel/tarot-agent/internal/store"
)

func TestCardStore_LoadAllCards(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}
	if cs.Count() != 78 {
		t.Errorf("expected 78 cards, got %d", cs.Count())
	}
}

func TestCardStore_MajorArcanaCount(t *testing.T) {
	cs, err := store.NewCardStore(store.MajorArcanaJSON())
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}
	if cs.Count() != 22 {
		t.Errorf("expected 22 major arcana cards, got %d", cs.Count())
	}
}

func TestCardStore_MinorArcanaPerSuit(t *testing.T) {
	suitAccessors := []struct {
		name    string
		jsonFn  func() []byte
		suit    string
	}{
		{"wands", store.MinorWandsJSON, "wands"},
		{"cups", store.MinorCupsJSON, "cups"},
		{"swords", store.MinorSwordsJSON, "swords"},
		{"pentacles", store.MinorPentaclesJSON, "pentacles"},
	}

	for _, tc := range suitAccessors {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := store.NewCardStore(tc.jsonFn())
			if err != nil {
				t.Fatalf("NewCardStore for %s failed: %v", tc.name, err)
			}
			if cs.Count() != 14 {
				t.Errorf("expected 14 %s cards, got %d", tc.name, cs.Count())
			}
			for _, card := range cs.GetAll() {
				if string(card.Suit) != tc.suit {
					t.Errorf("expected suit %s, got %s for card %s", tc.suit, card.Suit, card.ID)
				}
			}
		})
	}
}

func TestCardStore_GetByID(t *testing.T) {
	cs, err := store.NewCardStore(store.MajorArcanaJSON())
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	fool, err := cs.GetByID("major_00")
	if err != nil {
		t.Fatalf("GetByID('major_00') failed: %v", err)
	}
	if fool.NameCN != "愚者" {
		t.Errorf("expected '愚者', got '%s'", fool.NameCN)
	}
	if fool.NameEN != "The Fool" {
		t.Errorf("expected 'The Fool', got '%s'", fool.NameEN)
	}
	if len(fool.UprightKeywords) == 0 {
		t.Error("expected non-empty upright keywords")
	}
	if len(fool.ReversedKeywords) == 0 {
		t.Error("expected non-empty reversed keywords")
	}
}

func TestCardStore_GetByID_MinorArcana(t *testing.T) {
	cs, err := store.NewCardStore(store.MinorWandsJSON())
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	ace, err := cs.GetByID("minor_wands_01")
	if err != nil {
		t.Fatalf("GetByID('minor_wands_01') failed: %v", err)
	}
	if ace.NameCN != "权杖Ace" {
		t.Errorf("expected '权杖Ace', got '%s'", ace.NameCN)
	}
	if ace.Arcana != "minor" {
		t.Errorf("expected arcana 'minor', got '%s'", ace.Arcana)
	}
}

func TestCardStore_GetByID_NotFound(t *testing.T) {
	cs, err := store.NewCardStore(store.MajorArcanaJSON())
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	_, err = cs.GetByID("nonexistent_99")
	if err == nil {
		t.Error("expected error for non-existent card")
	}
}

func TestCardStore_GetAll(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	all := cs.GetAll()
	if len(all) != 78 {
		t.Errorf("expected 78 cards in GetAll, got %d", len(all))
	}
}

func TestCardStore_AllIDsUnique(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	all := cs.GetAll()
	seen := make(map[string]bool)
	for _, c := range all {
		if seen[c.ID] {
			t.Errorf("duplicate card ID: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestCardStore_FirstAndLast(t *testing.T) {
	cs, err := store.NewCardStore(store.MajorArcanaJSON())
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	first, err := cs.GetByID("major_00")
	if err != nil {
		t.Fatalf("GetByID('major_00') failed: %v", err)
	}
	if first.NameCN != "愚者" {
		t.Errorf("expected first card '愚者', got '%s'", first.NameCN)
	}

	last, err := cs.GetByID("major_21")
	if err != nil {
		t.Fatalf("GetByID('major_21') failed: %v", err)
	}
	if last.NameCN != "世界" {
		t.Errorf("expected last card '世界', got '%s'", last.NameCN)
	}
}

func TestCardStore_MinorCardFields(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	// Check a cups card
	cupsKing, err := cs.GetByID("minor_cups_14")
	if err != nil {
		t.Fatalf("GetByID('minor_cups_14') failed: %v", err)
	}
	if cupsKing.Number != 14 {
		t.Errorf("expected number 14, got %d", cupsKing.Number)
	}
	if cupsKing.NameEN != "King of Cups" {
		t.Errorf("expected 'King of Cups', got '%s'", cupsKing.NameEN)
	}
	if cupsKing.ImageFile == "" {
		t.Error("expected non-empty image_file")
	}

	// Check a swords card
	swordsAce, err := cs.GetByID("minor_swords_01")
	if err != nil {
		t.Fatalf("GetByID('minor_swords_01') failed: %v", err)
	}
	if swordsAce.NameCN != "宝剑Ace" {
		t.Errorf("expected '宝剑Ace', got '%s'", swordsAce.NameCN)
	}
	if len(swordsAce.UprightKeywords) == 0 {
		t.Error("expected non-empty upright keywords for swords ace")
	}
	if swordsAce.UprightMeaning == "" {
		t.Error("expected non-empty upright_meaning for swords ace")
	}
	if swordsAce.ReversedMeaning == "" {
		t.Error("expected non-empty reversed_meaning for swords ace")
	}
}

func TestCardEnrichedFields(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	for _, card := range cs.GetAll() {
		t.Run(card.ID, func(t *testing.T) {
			if card.Element == "" {
				t.Errorf("card %s: missing element", card.ID)
			}
			if card.Astrology == nil {
				t.Errorf("card %s: missing astrology", card.ID)
			}
			if card.Numerology == nil {
				t.Errorf("card %s: missing numerology", card.ID)
			}
			if card.Imagery == "" {
				t.Errorf("card %s: missing imagery", card.ID)
			}

			// Major arcana must have keywords_context
			if card.Arcana == "major" {
				if card.KeywordsContext == nil {
					t.Errorf("major arcana %s: missing keywords_context", card.ID)
				}
			}

			// Court cards (number 11-14, minor arcana only) must have court_role
			if card.Number >= 11 && card.Number <= 14 && card.Arcana == "minor" {
				if card.CourtRole == nil {
					t.Errorf("court card %s: missing court_role", card.ID)
				}
			}
		})
	}
}

func TestElementConsistency(t *testing.T) {
	validElements := map[string]bool{
		"fire": true, "water": true, "air": true, "earth": true,
	}

	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	suitElements := map[string]string{
		"wands": "fire", "cups": "water", "swords": "air", "pentacles": "earth",
	}

	for _, card := range cs.GetAll() {
		t.Run(card.ID, func(t *testing.T) {
			if !validElements[card.Element] {
				t.Errorf("card %s: invalid element %q", card.ID, card.Element)
			}
			if expected, ok := suitElements[string(card.Suit)]; ok {
				if card.Element != expected {
					t.Errorf("card %s: suit %s should have element %s, got %s",
						card.ID, card.Suit, expected, card.Element)
				}
			}
		})
	}
}

func TestNumerologyRange(t *testing.T) {
	cs, err := store.NewCardStore(
		store.MajorArcanaJSON(),
		store.MinorWandsJSON(),
		store.MinorCupsJSON(),
		store.MinorSwordsJSON(),
		store.MinorPentaclesJSON(),
	)
	if err != nil {
		t.Fatalf("NewCardStore failed: %v", err)
	}

	for _, card := range cs.GetAll() {
		if card.Number < 1 || card.Number > 10 {
			continue // Skip court cards and major arcana
		}
		if card.Arcana == "major" {
			continue // Skip major arcana for this test
		}
		t.Run(card.ID, func(t *testing.T) {
			if card.Numerology == nil {
				t.Fatalf("card %s: missing numerology", card.ID)
			}
			if card.Numerology.Number != card.Number {
				t.Errorf("card %s: numerology.number=%d but card.number=%d",
					card.ID, card.Numerology.Number, card.Number)
			}
		})
	}
}
