package store_test

import (
	"os"
	"path/filepath"
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
