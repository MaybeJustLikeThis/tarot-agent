package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/store"
)

func TestReadingStore_AppendAndList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.jsonl")
	rs := store.NewReadingStore(path)

	r := &domain.Reading{
		ID:             "test-001",
		Question:       "Will I find love?",
		SpreadID:       "single",
		Interpretation: "The cards suggest...",
		Disclaimer:     "For entertainment only",
		CreatedAt:      time.Now(),
	}

	if err := rs.Append(r); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	readings, err := rs.List(0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}
	if readings[0].ID != "test-001" {
		t.Errorf("expected ID 'test-001', got '%s'", readings[0].ID)
	}
	if readings[0].Question != "Will I find love?" {
		t.Errorf("expected question 'Will I find love?', got '%s'", readings[0].Question)
	}
}

func TestReadingStore_ListWithLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.jsonl")
	rs := store.NewReadingStore(path)

	// Append 5 readings
	for i := 0; i < 5; i++ {
		r := &domain.Reading{
			ID:        "test-00" + string(rune('1'+i)),
			Question:  "Question " + string(rune('1'+i)),
			SpreadID:  "single",
			CreatedAt: time.Now(),
		}
		if err := rs.Append(r); err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	// List with limit 3 (should return last 3)
	readings, err := rs.List(3)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 3 {
		t.Fatalf("expected 3 readings, got %d", len(readings))
	}
	// Should be the last 3: test-003, test-004, test-005
	if readings[0].ID != "test-003" {
		t.Errorf("expected first reading ID 'test-003', got '%s'", readings[0].ID)
	}
	if readings[2].ID != "test-005" {
		t.Errorf("expected last reading ID 'test-005', got '%s'", readings[2].ID)
	}
}

func TestReadingStore_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.jsonl")
	rs := store.NewReadingStore(path)

	readings, err := rs.List(0)
	if err != nil {
		t.Fatalf("List on nonexistent file should not error: %v", err)
	}
	if len(readings) != 0 {
		t.Errorf("expected 0 readings, got %d", len(readings))
	}
}

func TestReadingStore_AppendCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "readings.jsonl")
	rs := store.NewReadingStore(path)

	r := &domain.Reading{
		ID:        "test-dir-001",
		Question:  "Test",
		SpreadID:  "single",
		CreatedAt: time.Now(),
	}

	if err := rs.Append(r); err != nil {
		t.Fatalf("Append should create parent directories: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected readings file to exist after Append")
	}
}

func TestReadingStore_AppendMultiple(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.jsonl")
	rs := store.NewReadingStore(path)

	for i := 0; i < 10; i++ {
		r := &domain.Reading{
			ID:        "multi-001",
			Question:  "Same question",
			SpreadID:  "three_card",
			CreatedAt: time.Now(),
		}
		if err := rs.Append(r); err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	readings, err := rs.List(0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 10 {
		t.Errorf("expected 10 readings, got %d", len(readings))
	}
}

func TestReadingStore_RoundTripFullReading(t *testing.T) {
	dir := t.TempDir()
	rs := store.NewReadingStore(filepath.Join(dir, "readings.jsonl"))

	original := &domain.Reading{
		ID:       "r_full",
		Question: "事业接下来怎么走？",
		SpreadID: "three_card",
		Cards: []domain.DrawnCard{
			{
				Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
				Orientation: domain.Upright,
				Position:    domain.Position{Index: 0, NameCN: "过去", NameEN: "Past"},
			},
			{
				Card:        domain.Card{ID: "minor_wands_01", NameCN: "权杖Ace", NameEN: "Ace of Wands"},
				Orientation: domain.Reversed,
				Position:    domain.Position{Index: 1, NameCN: "现在", NameEN: "Present"},
			},
		},
		Interpretation: "完整解读正文",
		Model:          "deepseek-chat",
		CreatedAt:      time.Date(2026, 6, 9, 8, 30, 0, 0, time.UTC),
	}
	if err := rs.Append(original); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := rs.List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(got))
	}
	r := got[0]
	if r.Question != original.Question || r.SpreadID != original.SpreadID || r.Model != original.Model {
		t.Fatalf("metadata mismatch: %+v", r)
	}
	if r.Interpretation != original.Interpretation {
		t.Fatalf("interpretation = %q", r.Interpretation)
	}
	if len(r.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(r.Cards))
	}
	if r.Cards[0].Card.ID != "major_00" || r.Cards[0].Orientation != domain.Upright || r.Cards[0].Position.NameCN != "过去" {
		t.Fatalf("first card mismatch: %+v", r.Cards[0])
	}
	if r.Cards[1].Orientation != domain.Reversed || r.Cards[1].Position.NameCN != "现在" {
		t.Fatalf("second card mismatch: %+v", r.Cards[1])
	}
}

func TestReadingStore_LegacyLineWithoutCards(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.jsonl")
	legacy := `{"id":"r_old","question":"旧问题","spread_id":"single","interpretation":"只有摘要","created_at":"2026-01-02T15:04:05Z"}`
	if err := os.WriteFile(path, []byte(legacy+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.NewReadingStore(path).List(10)
	if err != nil {
		t.Fatalf("List legacy line: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(got))
	}
	r := got[0]
	if r.ID != "r_old" || r.Question != "旧问题" || r.SpreadID != "single" || r.Interpretation != "只有摘要" {
		t.Fatalf("legacy fields mismatch: %+v", r)
	}
	if r.Cards != nil {
		t.Fatalf("expected nil cards, got %#v", r.Cards)
	}
	if r.Model != "" || r.Disclaimer != "" {
		t.Fatalf("expected empty model and disclaimer, got model=%q disclaimer=%q", r.Model, r.Disclaimer)
	}
	if r.CreatedAt.IsZero() {
		t.Fatal("expected created_at to parse")
	}
}

func TestReadingStore_LongLine(t *testing.T) {
	dir := t.TempDir()
	rs := store.NewReadingStore(filepath.Join(dir, "readings.jsonl"))
	original := &domain.Reading{
		ID:             "r_long",
		Question:       "长解读",
		SpreadID:       "single",
		Interpretation: strings.Repeat("测", 40000),
		CreatedAt:      time.Now().UTC(),
	}
	if err := rs.Append(original); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := rs.List(1)
	if err != nil {
		t.Fatalf("List long line: %v", err)
	}
	if len(got) != 1 || got[0].Interpretation != original.Interpretation {
		t.Fatalf("long interpretation was not read back intact (len=%d)", len(got))
	}
}
