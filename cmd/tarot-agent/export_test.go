package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

func TestParseExport(t *testing.T) {
	opt, err := parseCLI([]string{"export"})
	if err != nil || !opt.export || opt.daily || opt.update || len(opt.exportArgs) != 0 {
		t.Fatalf("bare export: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"--mode=pro", "export", "r_1", "--format", "txt", "-o", "out.md", "--force"})
	if err != nil || !opt.export || opt.mode != "pro" {
		t.Fatalf("flags before export: %+v err %v", opt, err)
	}
	if strings.Join(opt.exportArgs, " ") != "r_1 --format txt -o out.md --force" {
		t.Fatalf("export args %q", opt.exportArgs)
	}
	opt, err = parseCLI([]string{"export", "latest", "--format=md"})
	if err != nil || !opt.export || opt.showVersion {
		t.Fatalf("latest: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"--version", "export"})
	if err != nil || !opt.showVersion || !opt.export {
		t.Fatalf("version still wins parse: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"foo"})
	if err != nil || opt.export || opt.daily || opt.update {
		t.Fatalf("positional: %+v err %v", opt, err)
	}
	if _, err = parseCLI([]string{"--format", "md"}); !errors.Is(err, errFlagParse) {
		t.Fatalf("--format without export should stay unknown: %v", err)
	}
	if _, err = parseCLI([]string{"-o", "x"}); !errors.Is(err, errFlagParse) {
		t.Fatalf("-o without export should stay unknown: %v", err)
	}
	if err = run([]string{"export", "-h"}); !errors.Is(err, errFlagHelp) {
		t.Fatalf("export help: %v", err)
	}

	parsed, err := parseExportArgs([]string{"r_9", "--format", "txt", "-o", "a.txt"})
	if err != nil || parsed.Query != "r_9" || parsed.Format != "txt" || parsed.OutputPath != "a.txt" || parsed.Force {
		t.Fatalf("parsed %+v err %v", parsed, err)
	}
	if _, err = parseExportArgs([]string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "一个记录") {
		t.Fatalf("two ids: %v", err)
	}
	if _, err = parseExportArgs([]string{"--format", "json"}); err == nil || !strings.Contains(err.Error(), "md") {
		t.Fatalf("bad format: %v", err)
	}
	if _, err = parseExportArgs([]string{"--nope"}); !errors.Is(err, errFlagParse) {
		t.Fatalf("bad export flag: %v", err)
	}
}

func TestExportCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := store.DefaultReadingsPath()
	if err != nil {
		t.Fatal(err)
	}
	rs := store.NewReadingStore(path)
	when := time.Date(2026, 10, 19, 8, 30, 0, 0, time.UTC)
	if err := rs.Append(&domain.Reading{
		ID:             "r_old",
		Question:       "旧问题",
		SpreadID:       "single",
		Interpretation: "只有摘要",
		CreatedAt:      when.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := rs.Append(&domain.Reading{
		ID:             "r_new",
		Question:       "新问题",
		SpreadID:       "three_card",
		Model:          "deepseek-chat",
		Interpretation: "完整解读",
		CreatedAt:      when,
		FollowUps:      []domain.FollowUpTurn{{Role: "user", Content: "追问一句"}, {Role: "assistant", Content: "回答一句"}},
	}); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error { return run([]string{"export"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "新问题") || !strings.Contains(out, "## 追问") || !strings.Contains(out, "回答一句") {
		t.Fatalf("latest md:\n%s", out)
	}
	if strings.Contains(out, "旧问题") {
		t.Fatalf("latest included the older reading:\n%s", out)
	}

	out, err = captureStdout(t, func() error { return run([]string{"export", "r_old", "--format", "txt"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "旧问题") || !strings.Contains(out, "（未保存牌面）") || strings.Contains(out, "##") {
		t.Fatalf("old txt:\n%s", out)
	}

	err = run([]string{"export", "missing"})
	if err == nil || !strings.Contains(err.Error(), "找不到占卜记录 missing") {
		t.Fatalf("missing: %v", err)
	}

	dest := filepath.Join(home, "out.md")
	if err := os.WriteFile(dest, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"export", "r_new", "-o", dest})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("exists: %v", err)
	}
	kept, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != "keep" {
		t.Fatalf("file changed: %q", kept)
	}
	if err := run([]string{"export", "r_new", "-o", dest, "--force"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "新问题") || !strings.Contains(string(body), "追问一句") {
		t.Fatalf("forced file:\n%s", body)
	}

	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)
	err = run([]string{"export", "latest"})
	if err == nil || !strings.Contains(err.Error(), "还没有可导出的占卜记录") {
		t.Fatalf("empty: %v", err)
	}
}
