package store_test

import (
	"testing"

	"github.com/voocel/tarot-agent/internal/store"
)

func TestSpreadStore_LoadSpreads(t *testing.T) {
	ss, err := store.NewSpreadStore(store.SpreadsJSON())
	if err != nil {
		t.Fatalf("NewSpreadStore failed: %v", err)
	}

	all := ss.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 spreads, got %d", len(all))
	}
}

func TestSpreadStore_ThreeCard(t *testing.T) {
	ss, err := store.NewSpreadStore(store.SpreadsJSON())
	if err != nil {
		t.Fatalf("NewSpreadStore failed: %v", err)
	}

	threeCard, err := ss.GetByID("three_card")
	if err != nil {
		t.Fatalf("GetByID('three_card') failed: %v", err)
	}
	if threeCard.NameCN != "三张牌" {
		t.Errorf("expected '三张牌', got '%s'", threeCard.NameCN)
	}
	if len(threeCard.Positions) != 3 {
		t.Errorf("expected 3 positions, got %d", len(threeCard.Positions))
	}
	if threeCard.MinCards != 3 {
		t.Errorf("expected MinCards 3, got %d", threeCard.MinCards)
	}
}

func TestSpreadStore_GetByID_NotFound(t *testing.T) {
	ss, err := store.NewSpreadStore(store.SpreadsJSON())
	if err != nil {
		t.Fatalf("NewSpreadStore failed: %v", err)
	}

	_, err = ss.GetByID("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent spread")
	}
}

func TestSpreadStore_CelticCross(t *testing.T) {
	ss, err := store.NewSpreadStore(store.SpreadsJSON())
	if err != nil {
		t.Fatalf("NewSpreadStore failed: %v", err)
	}

	cc, err := ss.GetByID("celtic_cross")
	if err != nil {
		t.Fatalf("GetByID('celtic_cross') failed: %v", err)
	}
	if len(cc.Positions) != 10 {
		t.Errorf("expected 10 positions for celtic cross, got %d", len(cc.Positions))
	}
}

func TestSpreadStore_Single(t *testing.T) {
	ss, err := store.NewSpreadStore(store.SpreadsJSON())
	if err != nil {
		t.Fatalf("NewSpreadStore failed: %v", err)
	}

	single, err := ss.GetByID("single")
	if err != nil {
		t.Fatalf("GetByID('single') failed: %v", err)
	}
	if len(single.Positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(single.Positions))
	}
}
