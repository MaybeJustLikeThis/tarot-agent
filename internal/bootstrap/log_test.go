package bootstrap

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLoggerDiscardsWhenFileCannotOpen(t *testing.T) {
	restoreLogger(t)
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t)
	InitLogger("debug", filepath.Join(blocker, "tarot.log"))
	slog.Info("should not reach stderr")
	slog.Debug("neither should this")

	if out := stderr(); out != "" {
		t.Fatalf("stderr = %q", out)
	}
}

func TestInitLoggerDiscardsEmptyPath(t *testing.T) {
	restoreLogger(t)
	stderr := captureStderr(t)
	InitLogger("info", "")
	slog.Info("no file")
	if out := stderr(); out != "" {
		t.Fatalf("stderr = %q", out)
	}
}

func TestInitLoggerWritesToFile(t *testing.T) {
	restoreLogger(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "tarot.log")
	stderr := captureStderr(t)

	InitLogger("info", path)
	slog.Info("hello-from-test")

	if out := stderr(); out != "" {
		t.Fatalf("stderr = %q", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello-from-test") {
		t.Fatalf("log = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("log mode %o, want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o, want 0700", dirInfo.Mode().Perm())
	}
}

func TestRotatingLogKeepsOneBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tarot.log")
	const maxSize = 50
	w, err := openRotatingLog(path, maxSize)
	if err != nil {
		t.Fatal(err)
	}
	line := []byte("0123456789abcdef")
	for i := 0; i < 10; i++ {
		if _, err := w.Write(line); err != nil {
			t.Fatal(err)
		}
	}

	cur, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(cur)) > maxSize {
		t.Fatalf("current size %d exceeds %d", len(cur), maxSize)
	}
	if len(cur) == 0 || len(bak) == 0 {
		t.Fatalf("current %d backup %d", len(cur), len(bak))
	}
	if _, err := os.Stat(path + ".2"); !os.IsNotExist(err) {
		t.Fatal("expected only one backup")
	}
	if !bytes.Contains(bak, line) {
		t.Fatalf("backup missing written bytes: %q", bak)
	}
}

func TestRotatingLogKeepsAppendingWhenRotateFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tarot.log")
	w, err := openRotatingLog(path, 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("0123456789")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path+".1", 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("abcdefghij")
	n, err := w.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("wrote %d, want %d", n, len(payload))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, payload) {
		t.Fatalf("log = %q", data)
	}
}

func restoreLogger(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prev := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = prev
		w.Close()
		r.Close()
	})
	return func() string {
		w.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
}
