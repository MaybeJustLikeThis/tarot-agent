package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/tarot-agent/internal/store"
)

func TestIO_ReadJSON(t *testing.T) {
	io := store.NewIO()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := `{"name":"test","value":42}`
	os.WriteFile(path, []byte(data), 0o644)

	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	if err := io.ReadJSON(path, &result); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	if result.Name != "test" || result.Value != 42 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestIO_WriteJSON(t *testing.T) {
	io := store.NewIO()

	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	input := struct {
		Name string `json:"name"`
	}{Name: "hello"}

	if err := io.WriteJSON(path, input); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := io.ReadJSON(path, &result); err != nil {
		t.Fatalf("ReadJSON after WriteJSON failed: %v", err)
	}
	if result.Name != "hello" {
		t.Errorf("expected 'hello', got '%s'", result.Name)
	}
}

func TestIO_WriteJSON_Atomic(t *testing.T) {
	io := store.NewIO()

	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.json")

	input := struct {
		Val int `json:"val"`
	}{Val: 100}

	if err := io.WriteJSON(path, input); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("tmp file should not exist after successful write")
	}
}
