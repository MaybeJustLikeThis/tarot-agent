package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestHistoryDetailLongEnglishWraps(t *testing.T) {
	text := renderHistoryDetail(domain.Reading{
		Question: strings.Repeat("事业与感情", 12),
		Cards: []domain.DrawnCard{{
			Card:        domain.Card{ID: "minor_pentacles_12", NameCN: "星币骑士", NameEN: "Knight of Pentacles"},
			Orientation: domain.Upright,
			Position:    domain.Position{NameCN: "过去"},
		}},
	})
	question := truncateWidth(strings.Repeat("事业与感情", 12), 16)
	if strings.Contains(question, "\n") || lipgloss.Width(question) > 16 || !strings.Contains(question, "…") {
		t.Fatalf("question line %q width %d", question, lipgloss.Width(question))
	}
	rendered := renderMarkdown(text, 22)
	for i, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > 22 {
			t.Errorf("detail line %d width %d: %q", i, lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(text, "Knight of Pentacles") {
		t.Fatal("stored english name was dropped")
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
	if strings.Contains(text, "追问") {
		t.Fatalf("legacy detail should not add a follow-up section:\n%s", text)
	}
}

func TestRenderHistoryDetail_FollowUps(t *testing.T) {
	text := renderHistoryDetail(domain.Reading{
		Question:       "事业接下来怎么走？",
		Interpretation: "完整解读正文",
		FollowUps: []domain.FollowUpTurn{
			{Role: "user", Content: "那下一步呢"},
			{Role: "assistant", Content: "先写下来"},
		},
	})
	body := strings.Index(text, "完整解读正文")
	section := strings.Index(text, "**追问**")
	if body < 0 || section < body {
		t.Fatalf("follow-ups should follow the interpretation:\n%s", text)
	}
	if !strings.Contains(text, "**你**：那下一步呢") || !strings.Contains(text, "**助手**：先写下来") {
		t.Fatalf("turns missing:\n%s", text)
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
		t.Fatalf("follow-up without a reading id must not append, got %d", len(got))
	}
}

func TestFollowUpUpdatesSameReading(t *testing.T) {
	m := newHistoryModel(t)
	m.readingBuf.WriteString("完整解读正文")
	if _, _ = (&ReadingState{}).Update(m, agentEndMsg{}); m.err != nil {
		t.Fatal(m.err)
	}
	id := m.readingID
	if id == "" || !m.historyWritten {
		t.Fatalf("reading id not kept: %q written=%v", id, m.historyWritten)
	}

	m.appendChatUserMessage("那下一步呢")
	m.chatStreamBuf.WriteString("先写下来")
	if _, _ = (&ChatState{}).Update(m, agentEndMsg{}); m.err != nil {
		t.Fatal(m.err)
	}
	m.appendChatUserMessage("还有呢")
	m.chatStreamBuf.WriteString("再小一点")
	if _, _ = (&ChatState{}).Update(m, agentEndMsg{}); m.err != nil {
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
	if r.ID != id || r.Interpretation != "完整解读正文" {
		t.Fatalf("record changed: id=%s text=%q", r.ID, r.Interpretation)
	}
	if len(r.FollowUps) != 4 {
		t.Fatalf("followups = %+v", r.FollowUps)
	}
	if r.FollowUps[0].Role != "user" || r.FollowUps[0].Content != "那下一步呢" || r.FollowUps[1].Role != "assistant" {
		t.Fatalf("first turn: %+v", r.FollowUps[:2])
	}
	if r.FollowUps[2].Content != "还有呢" || r.FollowUps[3].Content != "再小一点" {
		t.Fatalf("second turn: %+v", r.FollowUps[2:])
	}
}

func TestHistoryEnterContinuesSameReading(t *testing.T) {
	m := newHistoryModel(t)
	when := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	original := &domain.Reading{
		ID:             "r_hist",
		Question:       "事业接下来怎么走？",
		SpreadID:       "three_card",
		Model:          "deepseek-chat",
		Interpretation: "旧解读",
		CreatedAt:      when,
		Cards:          snapshotCards(m.drawResult.Cards),
		FollowUps: []domain.FollowUpTurn{
			{Role: "user", Content: "上一问", CreatedAt: when},
			{Role: "assistant", Content: "上一答"},
		},
	}
	if err := m.store.Readings.Append(original); err != nil {
		t.Fatal(err)
	}
	readings, err := m.store.Readings.List(10)
	if err != nil {
		t.Fatal(err)
	}
	m.historyReadings = readings
	m.historyCursor = 0
	m.state = &HistoryState{}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := updated.(*Model)
	if _, ok := mm.state.(*FollowUpState); !ok {
		t.Fatalf("state %T", mm.state)
	}
	if mm.readingID != "r_hist" || !mm.historyWritten || !mm.followUpNeedsContext {
		t.Fatalf("resume id=%q written=%v context=%v", mm.readingID, mm.historyWritten, mm.followUpNeedsContext)
	}
	if mm.userInput != original.Question || mm.spreadType != "three_card" || mm.readingContent != "旧解读" {
		t.Fatalf("restored reading: q=%q spread=%q text=%q", mm.userInput, mm.spreadType, mm.readingContent)
	}
	if mm.drawResult == nil || len(mm.drawResult.Cards) != 2 || len(mm.drawResult.Cards[0].Card.UprightKeywords) == 0 {
		t.Fatalf("cards were not restored from the deck: %+v", mm.drawResult)
	}
	if len(mm.chatMessages) != 2 || mm.chatMessages[0].Content != "上一问" || !mm.chatMessages[0].CreatedAt.Equal(when) {
		t.Fatalf("chat: %+v", mm.chatMessages)
	}

	mm.appendChatUserMessage("继续问")
	mm.chatStreamBuf.WriteString("继续答")
	mm.state = &ChatState{}
	updated, _ = mm.Update(agentEndMsg{})
	mm = updated.(*Model)
	if mm.err != nil {
		t.Fatal(mm.err)
	}
	got, err := mm.store.Readings.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "r_hist" || got[0].Interpretation != "旧解读" {
		t.Fatalf("record: %+v", got)
	}
	if len(got[0].FollowUps) != 4 || got[0].FollowUps[2].Content != "继续问" || got[0].FollowUps[3].Content != "继续答" {
		t.Fatalf("followups: %+v", got[0].FollowUps)
	}
	if !got[0].FollowUps[0].CreatedAt.Equal(when) {
		t.Fatalf("old turn restamped: %s", got[0].FollowUps[0].CreatedAt)
	}
}

func TestHistoryEnterWithoutCardsStaysReadOnly(t *testing.T) {
	m := newHistoryModel(t)
	m.historyReadings = []domain.Reading{{
		ID:             "r_old",
		Question:       "旧问题",
		SpreadID:       "single",
		Interpretation: "只有摘要",
	}}
	m.historyCursor = 0
	m.state = &HistoryState{}
	(&HistoryState{}).updateViewport(m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := updated.(*Model)
	if _, ok := mm.state.(*HistoryState); !ok {
		t.Fatalf("state %T", mm.state)
	}
	if mm.readingID != "" || mm.historyWritten {
		t.Fatalf("read-only record became writable: id=%q written=%v", mm.readingID, mm.historyWritten)
	}
	view := mm.View()
	if !strings.Contains(view, "旧记录无牌面，无法继续") || !strings.Contains(view, "esc 返回") {
		t.Fatalf("hint missing:\n%s", view)
	}
	if !strings.Contains(view, "只有摘要") {
		t.Fatal("detail was dropped")
	}
	got, err := mm.store.Readings.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("enter appended %d readings", len(got))
	}
}

func TestHistoryNoCardsHintFits(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {80, 20}} {
		m := newTestModelSize(sz[0], sz[1])
		m.store.Readings = store.NewReadingStore(filepath.Join(t.TempDir(), "readings.jsonl"))
		m.historyReadings = []domain.Reading{{
			Question:       "旧问题",
			SpreadID:       "single",
			Interpretation: "只有摘要",
		}}
		m.historyCursor = 0
		m.state = &HistoryState{}
		m.layoutHeights()
		(&HistoryState{}).updateViewport(m)
		view := m.View()
		lines := strings.Split(view, "\n")
		n := len(lines)
		if n > 0 && lines[n-1] == "" {
			n--
		}
		if n > sz[1] {
			t.Errorf("%dx%d: %d rows", sz[0], sz[1], n)
		}
		for i, line := range lines {
			if lipgloss.Width(line) > sz[0] {
				t.Errorf("%dx%d line %d width %d: %q", sz[0], sz[1], i, lipgloss.Width(line), line)
			}
		}
		if !strings.Contains(view, "旧记录无牌面，无法继续") || !strings.Contains(view, "esc 返回") {
			t.Errorf("%dx%d: hint missing", sz[0], sz[1])
		}
	}
}

func TestFollowUpSaveFailureKeepsChat(t *testing.T) {
	m := newHistoryModel(t)
	m.readingID = "r_missing"
	m.historyWritten = true
	m.appendChatUserMessage("那下一步呢")
	m.chatStreamBuf.WriteString("先写下来")
	m.state = &ChatState{}

	updated, _ := m.Update(agentEndMsg{})
	mm := updated.(*Model)
	if _, ok := mm.state.(*FollowUpState); !ok {
		t.Fatalf("state %T", mm.state)
	}
	if mm.err == nil || !strings.Contains(mm.err.Error(), "保存追问") {
		t.Fatalf("err = %v", mm.err)
	}
	if len(mm.chatMessages) != 2 || mm.chatMessages[1].Content != "先写下来" {
		t.Fatalf("chat dropped: %+v", mm.chatMessages)
	}
	if mm.readingID != "r_missing" {
		t.Fatalf("reading id cleared: %q", mm.readingID)
	}
}

func TestResumedFollowUpPromptIncludesReadingOnce(t *testing.T) {
	m := newHistoryModel(t)
	m.userInput = "事业接下来怎么走？"
	m.spreadType = "three_card"
	m.readingContent = "旧解读"
	m.chatMessages = []ChatMessage{
		{Role: "user", Content: "上一问"},
		{Role: "assistant", Content: "上一答"},
		{Role: "user", Content: "继续问"},
	}
	m.followUpNeedsContext = true

	got := m.nextFollowUpPrompt("继续问")
	if m.followUpNeedsContext {
		t.Fatal("context flag should clear after the first resumed turn")
	}
	for _, want := range []string{"旧解读", "上一问", "上一答", "继续问", "不要重新抽牌", "愚者"} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt missing %q\n%s", want, got)
		}
	}
	if strings.Count(got, "继续问") != 1 {
		t.Fatalf("new question repeated:\n%s", got)
	}
	if again := m.nextFollowUpPrompt("再问一句"); again != "再问一句" {
		t.Fatalf("later turn should be the raw question, got %q", again)
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
