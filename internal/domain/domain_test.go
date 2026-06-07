package domain_test

import (
	"testing"
	"time"

	"github.com/voocel/tarot-agent/internal/domain"
)

func TestCard_Fields(t *testing.T) {
	card := domain.Card{
		ID:              "major_00",
		NameCN:          "愚者",
		NameEN:          "The Fool",
		Arcana:          domain.MajorArcana,
		Suit:            domain.SuitNone,
		Number:          0,
		UprightKeywords: []string{"新开始", "冒险", "天真", "自由", "信仰"},
		ReversedKeywords: []string{"鲁莽", "冒失", "恐惧未知", "不计后果", "停滞"},
		UprightMeaning:  "新开始、冒险精神、天真无邪、自由、信仰的飞跃",
		ReversedMeaning: "鲁莽行事、冒失、恐惧未知、不计后果、停滞不前",
		ImageFile:       "00-fool.png",
	}

	if card.NameCN != "愚者" {
		t.Errorf("expected NameCN '愚者', got '%s'", card.NameCN)
	}
	if card.Arcana != domain.MajorArcana {
		t.Errorf("expected MajorArcana, got '%s'", card.Arcana)
	}
	if len(card.UprightKeywords) != 5 {
		t.Errorf("expected 5 upright keywords, got %d", len(card.UprightKeywords))
	}
	if card.ID != "major_00" {
		t.Errorf("expected ID 'major_00', got '%s'", card.ID)
	}
}

func TestArcanaType_Constants(t *testing.T) {
	if domain.MajorArcana != "major" {
		t.Errorf("expected 'major', got '%s'", domain.MajorArcana)
	}
	if domain.MinorArcana != "minor" {
		t.Errorf("expected 'minor', got '%s'", domain.MinorArcana)
	}
}

func TestOrientation_Constants(t *testing.T) {
	if domain.Upright != "upright" {
		t.Errorf("expected 'upright', got '%s'", domain.Upright)
	}
	if domain.Reversed != "reversed" {
		t.Errorf("expected 'reversed', got '%s'", domain.Reversed)
	}
}

func TestSpread_Fields(t *testing.T) {
	spread := domain.Spread{
		ID:          "three_card",
		NameCN:      "三牌阵",
		NameEN:      "Three Card Spread",
		Description: "过去、现在、未来",
		Positions: []domain.Position{
			{Index: 0, NameCN: "过去", NameEN: "Past", Description: "影响当前情况的过去因素"},
			{Index: 1, NameCN: "现在", NameEN: "Present", Description: "当前的情况和挑战"},
			{Index: 2, NameCN: "未来", NameEN: "Future", Description: "可能的发展方向"},
		},
		MinCards: 3,
		MaxCards: 3,
	}

	if spread.ID != "three_card" {
		t.Errorf("expected 'three_card', got '%s'", spread.ID)
	}
	if len(spread.Positions) != 3 {
		t.Errorf("expected 3 positions, got %d", len(spread.Positions))
	}
	if spread.MinCards != 3 {
		t.Errorf("expected MinCards 3, got %d", spread.MinCards)
	}
}

func TestDrawnCard_Fields(t *testing.T) {
	dc := domain.DrawnCard{
		Card: domain.Card{
			ID:     "major_00",
			NameCN: "愚者",
			NameEN: "The Fool",
			Arcana: domain.MajorArcana,
		},
		Orientation: domain.Upright,
		Position: domain.Position{
			Index:  0,
			NameCN: "过去",
			NameEN: "Past",
		},
	}

	if dc.Orientation != domain.Upright {
		t.Errorf("expected upright, got '%s'", dc.Orientation)
	}
	if dc.Card.NameEN != "The Fool" {
		t.Errorf("expected 'The Fool', got '%s'", dc.Card.NameEN)
	}
}

func TestReading_Fields(t *testing.T) {
	now := time.Now()
	reading := domain.Reading{
		ID:       "test-reading-1",
		Question: "我最近的事业发展如何？",
		SpreadID: "three_card",
		Cards: []domain.DrawnCard{
			{
				Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
				Orientation: domain.Upright,
				Position:    domain.Position{Index: 0, NameCN: "过去", NameEN: "Past"},
			},
		},
		Interpretation: "愚者出现在过去位置...",
		Disclaimer:     "塔罗是一种自我探索的工具",
		CreatedAt:      now,
	}

	if reading.Question != "我最近的事业发展如何？" {
		t.Errorf("unexpected question: %s", reading.Question)
	}
	if reading.SpreadID != "three_card" {
		t.Errorf("expected spread id 'three_card', got '%s'", reading.SpreadID)
	}
	if reading.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}
