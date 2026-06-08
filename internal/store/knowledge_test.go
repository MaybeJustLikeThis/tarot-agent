package store_test

import (
	"fmt"
	"testing"

	"github.com/voocel/tarot-agent/internal/store"
)

func TestElementInteractions(t *testing.T) {
	ek, err := store.NewElementKnowledge(store.ElementsJSON())
	if err != nil {
		t.Fatalf("NewElementKnowledge failed: %v", err)
	}

	elements := []string{"fire", "water", "air", "earth"}
	for _, a := range elements {
		for _, b := range elements {
			key1 := a + "_" + b
			key2 := b + "_" + a
			if _, ok1 := ek.Interactions[key1]; !ok1 {
				if _, ok2 := ek.Interactions[key2]; !ok2 {
					t.Errorf("missing element interaction for %s and %s", a, b)
				}
			}
		}
	}
}

func TestNumerologyEntries(t *testing.T) {
	ek, err := store.NewElementKnowledge(store.ElementsJSON())
	if err != nil {
		t.Fatalf("NewElementKnowledge failed: %v", err)
	}

	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("%d", i)
		if _, ok := ek.Numerology[key]; !ok {
			t.Errorf("missing numerology entry for %d", i)
		}
	}
}

func TestSuitMeanings(t *testing.T) {
	ek, err := store.NewElementKnowledge(store.ElementsJSON())
	if err != nil {
		t.Fatalf("NewElementKnowledge failed: %v", err)
	}

	expectedSuits := map[string]string{
		"wands": "fire", "cups": "water", "swords": "air", "pentacles": "earth",
	}
	for suit, expectedElement := range expectedSuits {
		info, ok := ek.SuitMeanings[suit]
		if !ok {
			t.Errorf("missing suit meaning for %s", suit)
			continue
		}
		if info.Element != expectedElement {
			t.Errorf("suit %s: expected element %s, got %s", suit, expectedElement, info.Element)
		}
		if info.Domain == "" {
			t.Errorf("suit %s: empty domain", suit)
		}
	}
}
