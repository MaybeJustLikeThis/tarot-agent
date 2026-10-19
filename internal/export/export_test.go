package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/tools"
)

func sampleReading() domain.Reading {
	when := time.Date(2026, 10, 19, 8, 30, 0, 0, time.UTC)
	return domain.Reading{
		ID:             "r_1",
		Question:       "事业接下来怎么走？",
		SpreadID:       "three_card",
		Model:          "deepseek-chat",
		CreatedAt:      when,
		Interpretation: "先把这一步写下来。",
		Disclaimer:     "仅供参考",
		Cards: []domain.DrawnCard{{
			Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
			Orientation: domain.Upright,
			Position:    domain.Position{NameCN: "过去", NameEN: "Past", Description: "影响当前处境的过去因素或根源"},
		}},
		FollowUps: []domain.FollowUpTurn{
			{Role: "user", Content: "那下一步呢", CreatedAt: when.Add(time.Hour)},
			{Role: "assistant", Content: "先写下来"},
		},
	}
}

func TestRenderMarkdownGolden(t *testing.T) {
	got, err := Render(sampleReading(), FormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	want := `# 占卜记录

- 时间：2026-10-19 08:30
- 模型：deepseek-chat
- 问题：事业接下来怎么走？
- 牌阵：三张牌

## 牌面

1. **过去** — 愚者（The Fool） · 正位
   - 位置含义：影响当前处境的过去因素或根源

## 解读

先把这一步写下来。

## 追问

**你**（2026-10-19 09:30）

那下一步呢

**助手**

先写下来

## 免责声明

仅供参考
`
	if got != want {
		t.Fatalf("markdown:\n%s", got)
	}
}

func TestRenderPlainGolden(t *testing.T) {
	got, err := Render(sampleReading(), "txt")
	if err != nil {
		t.Fatal(err)
	}
	want := `占卜记录

时间：2026-10-19 08:30
模型：deepseek-chat
问题：事业接下来怎么走？
牌阵：三张牌

牌面

1. 过去 — 愚者（The Fool） · 正位
   位置含义：影响当前处境的过去因素或根源

解读

先把这一步写下来。

追问

你（2026-10-19 09:30）
那下一步呢

助手
先写下来

免责声明

仅供参考
`
	if got != want {
		t.Fatalf("plain:\n%s", got)
	}
	if strings.Contains(got, "#") || strings.Contains(got, "**") {
		t.Fatalf("plain text kept markdown marks:\n%s", got)
	}
}

func TestRenderLegacyWithoutFollowUps(t *testing.T) {
	got, err := Render(domain.Reading{
		Question:       "旧问题",
		SpreadID:       "single",
		Interpretation: "只有摘要",
	}, "md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"旧问题", "单张牌", "未记录", "（未保存牌面）", "只有摘要", "## 免责声明"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q\n%s", want, got)
		}
	}
	if strings.Contains(got, "## 追问") {
		t.Fatalf("empty follow-ups should not add a section:\n%s", got)
	}
	if !strings.Contains(got, "不应被视为") {
		t.Fatalf("missing default disclaimer:\n%s", got)
	}
	if !strings.Contains(got, strings.TrimSpace(tools.DisclaimerText())) {
		t.Fatal("default disclaimer text drifted")
	}
}

func TestRenderFillsPositionMeaningFromSpread(t *testing.T) {
	got, err := Render(domain.Reading{
		SpreadID: "three_card",
		Cards: []domain.DrawnCard{{
			Card:        domain.Card{NameCN: "愚者", NameEN: "The Fool"},
			Orientation: domain.Reversed,
			Position:    domain.Position{NameCN: "过去"},
		}},
	}, "txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "愚者（The Fool） · 逆位") {
		t.Fatalf("card line:\n%s", got)
	}
	if !strings.Contains(got, "位置含义：影响当前处境的过去因素或根源") {
		t.Fatalf("meaning was not filled:\n%s", got)
	}
}

func TestRenderRejectsFormat(t *testing.T) {
	if _, err := Render(domain.Reading{}, "json"); err == nil {
		t.Fatal("expected format error")
	}
}

func TestFindAndWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.jsonl")
	rs := store.NewReadingStore(path)
	older := &domain.Reading{ID: "r_old", Question: "旧", Interpretation: "一", CreatedAt: time.Now().Add(-time.Hour)}
	newer := &domain.Reading{ID: "r_new", Question: "新", Interpretation: "二", CreatedAt: time.Now()}
	if err := rs.Append(older); err != nil {
		t.Fatal(err)
	}
	if err := rs.Append(newer); err != nil {
		t.Fatal(err)
	}
	all, err := rs.List(0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Find(all, "latest")
	if err != nil || got.ID != "r_new" {
		t.Fatalf("latest: %+v err %v", got, err)
	}
	got, err = Find(all, "r_old")
	if err != nil || got.Question != "旧" {
		t.Fatalf("id: %+v err %v", got, err)
	}
	if _, err := Find(all, "missing"); err == nil || !strings.Contains(err.Error(), "找不到占卜记录") {
		t.Fatalf("missing: %v", err)
	}
	if _, err := Find(nil, "r_old"); err == nil || !strings.Contains(err.Error(), "还没有") {
		t.Fatalf("empty: %v", err)
	}

	out := filepath.Join(dir, "nested", "out.md")
	var buf bytes.Buffer
	if err := Execute(Options{Readings: rs, Query: "r_old", Format: "md", Stdout: &buf}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "问题：旧") || strings.Contains(buf.String(), "## 追问") {
		t.Fatalf("stdout:\n%s", buf.String())
	}
	if err := Execute(Options{Readings: rs, Query: "latest", Format: "txt", OutputPath: out}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %o", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o", dirInfo.Mode().Perm())
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := Execute(Options{Readings: rs, Query: "r_old", Format: "md", OutputPath: out}); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("overwrite: %v", err)
	}
	again, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(body) {
		t.Fatal("existing file was modified without --force")
	}
	if err := Execute(Options{Readings: rs, Query: "r_old", Format: "md", OutputPath: out, Force: true}); err != nil {
		t.Fatal(err)
	}
	forced, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(forced), "问题：旧") {
		t.Fatalf("forced write:\n%s", forced)
	}
	info, err = os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode after force %o", info.Mode().Perm())
	}
}

func TestSaveMarkdownPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := SaveMarkdown(sampleReading())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".tarot-agent", "exports", "r_1.md")
	if path != want {
		t.Fatalf("path %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %o", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o", dirInfo.Mode().Perm())
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "那下一步呢") {
		t.Fatalf("saved:\n%s", body)
	}
	if _, err := SaveMarkdown(domain.Reading{Question: "无 id"}); err == nil {
		t.Fatal("expected missing id to fail")
	}
}

func TestSafeID(t *testing.T) {
	if got := SafeID("../r_1"); got != "r_1" {
		t.Fatalf("got %q", got)
	}
	if got := SafeID(".."); got != "reading" {
		t.Fatalf("got %q", got)
	}
}
