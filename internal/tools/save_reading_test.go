package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New()
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	tmpDir := t.TempDir()
	s.Readings = store.NewReadingStore(filepath.Join(tmpDir, "readings.jsonl"))
	return s
}

func TestSaveReading_AcceptsWithoutWriting(t *testing.T) {
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
	if resp["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got %q", resp["status"])
	}

	readings, err := s.Readings.List(10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("tool must not append a record, got %d", len(readings))
	}
}

func TestSaveReading_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	_, err := tool.Execute(context.Background(), json.RawMessage(`not valid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	readings, err := s.Readings.List(10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("invalid call must not append, got %d", len(readings))
	}
}

func TestSaveReading_RepeatedCallsDoNotAppend(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)
	args := json.RawMessage(`{"question":"问题","spread_id":"single","summary":"摘要"}`)

	for i := 0; i < 3; i++ {
		if _, err := tool.Execute(context.Background(), args); err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}
	}

	readings, err := s.Readings.List(10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("repeated confirmation must not append, got %d", len(readings))
	}
}

func TestSaveReading_DoesNotCreateFile(t *testing.T) {
	tmpDir := t.TempDir()
	readingsPath := filepath.Join(tmpDir, "readings.jsonl")

	s, err := store.New()
	if err != nil {
		t.Fatal(err)
	}
	s.Readings = store.NewReadingStore(readingsPath)
	tool := tools.SaveReadingTool(s)

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"question":"test","spread_id":"single","summary":"s"}`)); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if _, err := os.Stat(readingsPath); !os.IsNotExist(err) {
		t.Fatalf("tool must not create the readings file, stat err=%v", err)
	}
}

func TestSaveReading_EmptyArgsAccepted(t *testing.T) {
	s := newTestStore(t)
	tool := tools.SaveReadingTool(s)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute should not fail for empty args: %v", err)
	}
	var resp map[string]string
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got %q", resp["status"])
	}
}

func TestSaveReading_NilStore(t *testing.T) {
	tool := tools.SaveReadingTool(nil)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
}
