package daily

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/agents"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	s, err := store.New()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDailyWithoutModelPrintsLocalCardAndSaves(t *testing.T) {
	s := testStore(t)
	var buf strings.Builder
	now := time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)
	err := Run(context.Background(), Options{Store: s, Mode: agents.ModeCasual, Out: &buf, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "关键词：") || !strings.Contains(out, "未配置 API Key") {
		t.Fatalf("output:\n%s", out)
	}
	if !strings.Contains(out, "正位") && !strings.Contains(out, "逆位") {
		t.Fatalf("orientation missing:\n%s", out)
	}
	rows, err := s.Readings.List(10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows %d err %v", len(rows), err)
	}
	if rows[0].Question != domain.DailyQuestion || rows[0].SpreadID != "single" || len(rows[0].Cards) != 1 {
		t.Fatalf("saved %+v", rows[0])
	}
	if rows[0].Model != "" {
		t.Fatalf("model stored without a call: %q", rows[0].Model)
	}
}

func TestDailyModelFailureStillExitsClean(t *testing.T) {
	s := testStore(t)
	var buf strings.Builder
	err := Run(context.Background(), Options{
		Store: s,
		Mode:  agents.ModeProfessional,
		Model: "deepseek-chat",
		Out:   &buf,
		Guide: func(context.Context, string) (string, error) {
			return "", context.DeadlineExceeded
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "模型请求失败") || !strings.Contains(buf.String(), "关键词：") {
		t.Fatalf("output:\n%s", buf.String())
	}
	rows, err := s.Readings.List(1)
	if err != nil || len(rows) != 1 || rows[0].Model != "" {
		t.Fatalf("history %+v err %v", rows, err)
	}
}

func TestDailyGuideSentenceIsSaved(t *testing.T) {
	s := testStore(t)
	var buf strings.Builder
	var prompt string
	err := Run(context.Background(), Options{
		Store: s,
		Mode:  agents.ModeCasual,
		Model: "deepseek-chat",
		Out:   &buf,
		Guide: func(_ context.Context, got string) (string, error) {
			prompt = got
			return "今天可以把一件小事做完。\n第二句不要用", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "大白话") || !strings.Contains(prompt, "关键词：") {
		t.Fatalf("prompt %q", prompt)
	}
	if !strings.Contains(buf.String(), "今日一句：今天可以把一件小事做完。") || strings.Contains(buf.String(), "第二句") {
		t.Fatalf("output:\n%s", buf.String())
	}
	rows, _ := s.Readings.List(1)
	if rows[0].Interpretation != "今天可以把一件小事做完。" || rows[0].Model != "deepseek-chat" {
		t.Fatalf("saved %+v", rows[0])
	}
	if domain.FindSimilarQuestion("今日指引怎么看", rows) != nil {
		t.Fatal("daily row participated in matching")
	}
}
