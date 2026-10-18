package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

func newHistoryModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel()
	m.store.Readings = store.NewReadingStore(filepath.Join(t.TempDir(), "readings.jsonl"))
	m.llmModel = "deepseek-chat"
	m.userInput = "事业接下来怎么走？"
	m.spreadType = "three_card"
	m.drawResult = &domain.DrawResult{
		SpreadID: "three_card",
		Cards: []domain.DrawnCard{
			{
				Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
				Orientation: domain.Upright,
				Position:    domain.Position{NameCN: "过去"},
			},
			{
				Card:        domain.Card{ID: "minor_wands_01", NameCN: "权杖Ace", NameEN: "Ace of Wands"},
				Orientation: domain.Reversed,
				Position:    domain.Position{NameCN: "现在"},
			},
		},
	}
	return m
}

func TestRenderHistoryDetail_FullReading(t *testing.T) {
	text := renderHistoryDetail(domain.Reading{
		Question: "事业接下来怎么走？",
		SpreadID: "three_card",
		Model:    "deepseek-chat",
		Cards: []domain.DrawnCard{
			{
				Card:        domain.Card{ID: "major_00", NameCN: "愚者", NameEN: "The Fool"},
				Orientation: domain.Upright,
				Position:    domain.Position{NameCN: "过去"},
			},
		},
		Interpretation: "完整解读正文",
	})
	for _, want := range []string{
		"事业接下来怎么走？",
		"三张牌",
		"deepseek-chat",
		"过去 — 愚者 [正位]",
		"The Fool",
		"id: major_00",
		"完整解读正文",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("detail missing %q\n%s", want, text)
		}
	}
}

func TestRenderHistoryDetail_LegacyRecord(t *testing.T) {
	text := renderHistoryDetail(domain.Reading{
		Question:       "旧问题",
		SpreadID:       "single",
		Interpretation: "只有摘要",
	})
	for _, want := range []string{"旧问题", "单张牌", "未记录", "（旧记录未保存牌面）", "只有摘要"} {
		if !strings.Contains(text, want) {
			t.Errorf("legacy detail missing %q\n%s", want, text)
		}
	}
}

func TestRenderHistoryDetail_EmptyCardsAndText(t *testing.T) {
	text := renderHistoryDetail(domain.Reading{})
	if !strings.Contains(text, "（旧记录未保存牌面）") || !strings.Contains(text, "（无解读正文）") {
		t.Fatalf("empty record detail:\n%s", text)
	}
}

func TestReadingEndPersistsFullRecordOnce(t *testing.T) {
	m := newHistoryModel(t)
	m.readingBuf.WriteString("完整解读正文")

	state, _ := (&ReadingState{}).Update(m, agentEndMsg{})
	if _, ok := state.(*FollowUpState); !ok {
		t.Fatalf("expected FollowUpState, got %T", state)
	}
	if _, _ = (&ReadingState{}).Update(m, agentEndMsg{}); m.err != nil {
		t.Fatal(m.err)
	}

	got, err := m.store.Readings.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(got))
	}
	r := got[0]
	if r.Question != "事业接下来怎么走？" || r.SpreadID != "three_card" || r.Model != "deepseek-chat" {
		t.Fatalf("metadata: %+v", r)
	}
	if r.Interpretation != "完整解读正文" {
		t.Fatalf("interpretation = %q", r.Interpretation)
	}
	if len(r.Cards) != 2 || r.Cards[0].Card.ID != "major_00" || r.Cards[0].Orientation != domain.Upright || r.Cards[0].Position.NameCN != "过去" {
		t.Fatalf("cards: %+v", r.Cards)
	}
	if r.Cards[1].Orientation != domain.Reversed {
		t.Fatalf("second orientation = %s", r.Cards[1].Orientation)
	}
}

func TestFollowUpDoesNotPersist(t *testing.T) {
	m := newHistoryModel(t)
	m.readingContent = "已经看过的解读"
	m.chatStreamBuf.WriteString("追问回复")

	if _, _ = (&ChatState{}).Update(m, agentEndMsg{}); m.err != nil {
		t.Fatal(m.err)
	}
	got, err := m.store.Readings.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("follow-up must not be saved, got %d", len(got))
	}
}

func TestHistoryView_LegacyRecord(t *testing.T) {
	m := newTestModel()
	m.historyReadings = []domain.Reading{{
		Question:       "旧问题",
		SpreadID:       "single",
		Interpretation: "只有摘要",
	}}
	m.historyCursor = 0
	m.state = &HistoryState{}
	(&HistoryState{}).updateViewport(m)

	view := m.View()
	for _, want := range []string{"旧问题", "（旧记录未保存牌面）", "只有摘要", "未记录"} {
		if !strings.Contains(view, want) {
			t.Errorf("history view missing %q", want)
		}
	}
}
