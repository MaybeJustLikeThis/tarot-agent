package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSimilarHistoryShowsOnSpreadAndPrompt(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.store.Readings = store.NewReadingStore(filepath.Join(t.TempDir(), "readings.jsonl"))
	when := time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC)
	cards := []domain.DrawnCard{{
		Card:        domain.Card{NameCN: "女皇"},
		Orientation: domain.Reversed,
		Position:    domain.Position{NameCN: "过去"},
	}}
	if err := m.store.Readings.Append(&domain.Reading{
		Question:       "我和伴侣最近总是因为钱吵架",
		CreatedAt:      when,
		Cards:          cards,
		SpreadID:       "three_card",
		Interpretation: "### 逐张\n略。\n### 可执行建议\n本周把开支写下来，先不谈对错。\n",
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.store.Readings.Append(&domain.Reading{
		Question:       "我和伴侣最近总是因为钱吵架",
		CreatedAt:      when.Add(time.Hour),
		Interpretation: "### 可执行建议\n这条没有牌面，不该被用到。",
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.store.Readings.Append(&domain.Reading{
		Question:  domain.DailyQuestion,
		CreatedAt: when.Add(2 * time.Hour),
		Cards:     cards,
	}); err != nil {
		t.Fatal(err)
	}

	m.input.SetValue("我和伴侣最近还是因为钱吵架")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("state %T", m.state)
	}
	view := m.View()
	if !strings.Contains(view, "上次（2026-10-02）") || !strings.Contains(view, "女皇逆位") {
		t.Fatalf("spread hint:\n%s", view)
	}
	assertFitsScreen(t, "spread-prior", view, 24)
	assertLineWidths(t, "spread-prior", view, 80)
	prompt := readingPrompt(m.userInput, "three_card", "牌", nil, domain.FormatPriorBlock(m.priorMatch), m.boundary.PromptLine())
	if !strings.Contains(prompt, "【上次类似问题】") || !strings.Contains(prompt, "本周把开支写下来") || strings.Contains(prompt, "没有牌面") {
		t.Fatalf("prompt:\n%s", prompt)
	}
	if strings.Contains(prompt, "【边界】") {
		t.Fatal("plain question should not add a boundary section")
	}
}

func TestSpreadHintsFitSmallScreen(t *testing.T) {
	for _, h := range []int{24, 20} {
		m := newTestModelSize(80, h)
		m.userInput = "我和伴侣最近因为钱和未来规划吵架"
		m.spreadAdvice = RecommendSpread(m.userInput)
		cards := make([]domain.DrawnCard, 10)
		for i := range cards {
			cards[i] = domain.DrawnCard{Card: domain.Card{NameCN: "权杖国王"}, Orientation: domain.Upright}
		}
		m.priorMatch = &domain.PriorMatch{Reading: domain.Reading{
			Question:  "旧问题",
			CreatedAt: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
			Cards:     cards,
		}}
		m.boundary = ClassifyQuestion("身体健康运势")
		m.state = &SpreadState{}
		m.layoutHeights()
		view := m.View()
		assertFitsScreen(t, "spread-hints", view, h)
		assertLineWidths(t, "spread-hints", view, 80)
		if !strings.Contains(view, "等 10 张") || !strings.Contains(view, "闲谈") || !strings.Contains(view, "esc 返回 · q 退出") {
			t.Fatalf("h=%d hints:\n%s", h, view)
		}
	}
}

func TestCrisisStopsBeforeDraw(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("我不想活了")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*CrisisState); !ok {
		t.Fatalf("state %T", m.state)
	}
	if m.drawResult != nil {
		t.Fatal("crisis started a draw")
	}
	view := m.View()
	for _, want := range []string{"120", "110", "信任", "esc 返回"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q\n%s", want, view)
		}
	}
	assertFitsScreen(t, "crisis", view, 24)
	assertLineWidths(t, "crisis", view, 80)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(*Model)
	if _, ok := m.state.(*CrisisState); !ok || m.drawResult != nil {
		t.Fatal("a number key left the care screen")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*Model)
	if _, ok := m.state.(*InputState); !ok || m.input.Value() != "我不想活了" {
		t.Fatalf("esc state %T input %q", m.state, m.input.Value())
	}
}

func TestCtrlDStillSkipsCrisisText(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("我不想活了")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(*Model)
	if _, ok := m.state.(*RevealState); !ok || m.userInput != domain.DailyQuestion {
		t.Fatalf("state %T input %q", m.state, m.userInput)
	}
}

func TestHealthAndLegalConfirmBeforeSpread(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("我是不是得了抑郁症")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*QuestionGuideState); !ok {
		t.Fatalf("state %T", m.state)
	}
	view := m.View()
	if !strings.Contains(view, "不能代替医生") || !strings.Contains(view, "采纳改写") {
		t.Fatalf("health guide:\n%s", view)
	}
	assertFitsScreen(t, "health-guide", view, 24)
	assertLineWidths(t, "health-guide", view, 80)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("after accept %T", m.state)
	}
	if !strings.Contains(m.boundary.PromptLine(), "用药") {
		t.Fatal("health prompt line dropped")
	}

	m = newTestModelSize(80, 24)
	m.input.SetValue("他已经起诉我了")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	view = m.View()
	if !strings.Contains(view, "不能代替律师") || !strings.Contains(view, "继续选牌阵") {
		t.Fatalf("legal guide:\n%s", view)
	}
	assertFitsScreen(t, "legal-guide", view, 24)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("legal continue %T", m.state)
	}
	if !strings.Contains(m.View(), "1") || !strings.Contains(m.View(), "esc 返回 · q 退出") {
		t.Fatal("spread keys changed")
	}
}

func TestHealthLightDoesNotBlock(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.input.SetValue("身体健康运势")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if _, ok := m.state.(*SpreadState); !ok {
		t.Fatalf("state %T", m.state)
	}
	view := m.View()
	if !strings.Contains(view, "闲谈") || strings.Contains(view, "先不抽牌") {
		t.Fatalf("light hint:\n%s", view)
	}
	assertFitsScreen(t, "health-light", view, 24)
	assertLineWidths(t, "health-light", view, 80)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(*Model)
	if _, ok := m.state.(*RevealState); !ok || m.spreadType != "single" {
		t.Fatalf("light question did not draw: %T %s", m.state, m.spreadType)
	}
}
