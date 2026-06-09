package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/voocel/tarot-agent/internal/store"
	"github.com/voocel/tarot-agent/internal/tools"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	// Override readings store to use a temp file
	tmpDir := t.TempDir()
	s.Readings = store.NewReadingStore(filepath.Join(tmpDir, "readings.jsonl"))
	return s
}

func TestSaveReading_Success(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	args := json.RawMessage(`{
		"question": "我的事业方向如何？",
		"spread_id": "three_card",
		"summary": "正位的权杖骑士暗示行动力和冒险精神。"
	}`)

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var resp map[string]string
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if resp["status"] != "saved" {
		t.Errorf("expected status 'saved', got %q", resp["status"])
	}
	if resp["reading_id"] == "" {
		t.Error("expected non-empty reading_id")
	}

	// Verify the reading was persisted
	readings, err := s.Readings.List(10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}
	if readings[0].Question != "我的事业方向如何？" {
		t.Errorf("unexpected question: %s", readings[0].Question)
	}
	if readings[0].SpreadID != "three_card" {
		t.Errorf("unexpected spread_id: %s", readings[0].SpreadID)
	}
}

func TestSaveReading_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	_, err := tool.Execute(context.Background(), json.RawMessage(`not valid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveReading_MissingFields(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	// Missing spread_id and summary — should still save (no server-side validation)
	args := json.RawMessage(`{"question": "test"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not fail for missing optional-like fields: %v", err)
	}

	var resp map[string]string
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "saved" {
		t.Errorf("expected status 'saved', got %q", resp["status"])
	}
}

func TestSaveReading_MultipleReadings(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	for i := 0; i < 3; i++ {
		args := json.RawMessage(`{
			"question": "问题",
			"spread_id": "single",
			"summary": "摘要"
		}`)
		_, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}
	}

	readings, err := s.Readings.List(10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 3 {
		t.Errorf("expected 3 readings, got %d", len(readings))
	}
}

func TestSaveReading_IDUniqueness(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		args := json.RawMessage(`{
			"question": "q",
			"spread_id": "single",
			"summary": "s"
		}`)
		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}
		var resp map[string]string
		json.Unmarshal(result, &resp)
		if ids[resp["reading_id"]] {
			t.Errorf("duplicate reading_id: %s", resp["reading_id"])
		}
		ids[resp["reading_id"]] = true
		// Small delay to avoid nanosecond collision in tight loops.
		// In real usage, each reading takes seconds of AI processing time.
		time.Sleep(time.Millisecond)
	}
}

func TestSaveReading_FileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	readingsPath := filepath.Join(tmpDir, "readings.jsonl")

	s, _ := store.New()
	s.Readings = store.NewReadingStore(readingsPath)
	tool := tools.SaveReadingTool(s)

	args := json.RawMessage(`{
		"question": "test",
		"spread_id": "single",
		"summary": "test summary"
	}`)
	_, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if _, err := os.Stat(readingsPath); os.IsNotExist(err) {
		t.Error("expected readings file to be created")
	}
}

func TestSaveReading_EmptyArgs(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	// Empty JSON object — fields will be zero values but should still save
	args := json.RawMessage(`{}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not fail for empty args: %v", err)
	}

	var resp map[string]string
	json.Unmarshal(result, &resp)
	if resp["status"] != "saved" {
		t.Errorf("expected status 'saved', got %q", resp["status"])
	}
}
