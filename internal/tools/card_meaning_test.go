package tools

import (
	"testing"

	"github.com/voocel/tarot-agent/internal/store"
)

func createStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	return s
}

func TestNormalizeCardID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"major_0", "major_00"},
		{"major_1", "major_01"},
		{"major_9", "major_09"},
		{"major_00", "major_00"},
		{"major_10", "major_10"},
		{"major_21", "major_21"},
		{"wands_1", "minor_wands_01"},
		{"cups_3", "minor_cups_03"},
		{"swords_9", "minor_swords_09"},
		{"pentacles_1", "minor_pentacles_01"},
		{"wands_01", "minor_wands_01"},
		{"cups_10", "minor_cups_10"},
		{"", ""},
		{"unknown", ""},
		{"_01", "_01"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeCardID(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCardID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCardMeaning_FuzzyIDLookup(t *testing.T) {
	s := createStore(t)
	tool := GetCardMeaningTool(s)

	tests := []struct {
		name    string
		cardID  string
		wantErr bool
	}{
		{"exact major", "major_00", false},
		{"short major_0", "major_0", false},
		{"short major_1", "major_1", false},
		{"exact minor", "minor_wands_01", false},
		{"short wands_01", "wands_01", false},
		{"short wands_1", "wands_1", false},
		{"short cups_3", "cups_3", false},
		{"exact cups", "minor_cups_03", false},
		{"nonexistent", "fake_99", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := `{"card_id": "` + tt.cardID + `"}`
			_, err := tool.Execute(t.Context(), []byte(args))
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tt.cardID)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.cardID, err)
			}
		})
	}
}

func TestCardMeaning_NameWithAlias(t *testing.T) {
	s := createStore(t)
	tool := GetCardMeaningTool(s)

	tests := []struct {
		name     string
		cardName string
		wantErr  bool
	}{
		{"exact cn", "愚者", false},
		{"exact en", "The Fool", false},
		{"alias wands ace", "权杖一", false},
		{"alias cups ace", "圣杯一", false},
		{"alias swords ace", "宝剑一", false},
		{"alias pentacles ace", "星币一", false},
		{"nonexistent", "不存在的牌", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := `{"card_name": "` + tt.cardName + `"}`
			_, err := tool.Execute(t.Context(), []byte(args))
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tt.cardName)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.cardName, err)
			}
		})
	}
}
