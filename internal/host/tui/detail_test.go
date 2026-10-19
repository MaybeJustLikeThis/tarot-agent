package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleDetailCards() []domain.DrawnCard {
	return []domain.DrawnCard{
		{
			Card: domain.Card{
				NameCN:           "女皇",
				NameEN:           "The Empress",
				Arcana:           domain.MajorArcana,
				Number:           3,
				Element:          "fire",
				UprightKeywords:  []string{"丰盛"},
				ReversedKeywords: []string{"依赖"},
				Imagery:          "一位女皇坐在麦田边。",
				Numerology:       &domain.NumerologyInfo{Number: 3, Meaning: "创造的完成"},
				Astrology:        &domain.AstrologyInfo{Planet: "金星", Note: "生长"},
			},
			Orientation: domain.Reversed,
			Position:    domain.Position{NameCN: "过去"},
		},
		{
			Card: domain.Card{
				NameCN:          "战车",
				NameEN:          "The Chariot",
				Arcana:          domain.MajorArcana,
				Number:          7,
				Element:         "fire",
				UprightKeywords: []string{"意志"},
				Imagery:         "战士驾着两匹马。",
				Numerology:      &domain.NumerologyInfo{Number: 7, Meaning: "意志的考验"},
				Astrology:       &domain.AstrologyInfo{Zodiac: "巨蟹座"},
			},
			Orientation: domain.Upright,
			Position:    domain.Position{NameCN: "现在"},
		},
		{
			Card: domain.Card{
				NameCN:          "权杖国王",
				NameEN:          "King of Wands",
				Arcana:          domain.MinorArcana,
				Suit:            domain.SuitWands,
				Number:          14,
				Element:         "water",
				UprightKeywords: []string{"远见"},
				Imagery:         "一位国王举起开花的权杖，身后是火焰与沙漠。",
				Numerology:      &domain.NumerologyInfo{Number: 14, Meaning: "沉淀"},
				Astrology:       &domain.AstrologyInfo{Planet: "太阳", SubInfluence: "狮子座后段"},
				CourtRole: &domain.CourtRole{
					Archetype:   "领导者",
					Personality: "果断",
					AsPerson:    "一位敢拍板的人",
					AsMessage:   "先做决定",
				},
			},
			Orientation: domain.Upright,
			Position:    domain.Position{NameCN: "未来"},
		},
	}
}

func TestCtrlOOpensLocalCardDetail(t *testing.T) {
	for _, st := range []State{&ReadingState{}, &FollowUpState{}, &ChatState{}} {
		m := newTestModelSize(80, 24)
		m.mode = "深入版"
		m.state = st
		m.drawResult = &domain.DrawResult{Cards: sampleDetailCards()}
		m.layoutHeights()

		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
		mm := updated.(*Model)
		detail, ok := mm.state.(*CardDetailState)
		if !ok || detail.prev != st {
			t.Fatalf("%T: state %T", st, mm.state)
		}
		if _, reading := st.(*ReadingState); reading && !mm.spinnerOn() {
			t.Fatal("detail hid the reading spinner")
		}
		view := mm.View()
		for _, want := range []string{"女皇", "依赖", "火", "创造的完成", "金星", "麦田", "火与水同时出现", "1/3", "esc 返回"} {
			if !strings.Contains(view, want) {
				t.Errorf("%T: missing %q\n%s", st, want, view)
			}
		}
		assertFitsScreen(t, "detail-open", view, 24)
		assertLineWidths(t, "detail-open", view, 80)
	}
}

func TestCardDetailPagesAndClamps(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.state = &ReadingState{}
	m.drawResult = &domain.DrawResult{Cards: sampleDetailCards()}
	m.layoutHeights()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	mm := updated.(*Model)

	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyLeft})
	mm = updated.(*Model)
	if !strings.Contains(mm.View(), "依赖") {
		t.Fatal("left from the first card wrapped")
	}

	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRight})
	mm = updated.(*Model)
	if !strings.Contains(mm.View(), "意志") || strings.Contains(mm.View(), "依赖") {
		t.Fatal("right did not move to the second card")
	}

	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	mm = updated.(*Model)
	view := mm.View()
	for _, want := range []string{"权杖国王", "远见", "太阳", "狮子座后段", "领导者", "先做决定", "3/3"} {
		if !strings.Contains(view, want) {
			t.Errorf("third card missing %q\n%s", want, view)
		}
	}
	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	mm = updated.(*Model)
	if !strings.Contains(mm.View(), "3/3") {
		t.Fatal("right past the end wrapped")
	}
	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	mm = updated.(*Model)
	if !strings.Contains(mm.View(), "战车") {
		t.Fatal("h did not move back")
	}
}

func TestCardDetailEscRestoresScrollWhileStreamContinues(t *testing.T) {
	m := newTestModelSize(80, 24)
	m.store.Readings = store.NewReadingStore(filepath.Join(t.TempDir(), "readings.jsonl"))
	m.state = &ReadingState{}
	m.userInput = "接下来怎么走？"
	m.spreadType = "three_card"
	m.drawResult = &domain.DrawResult{SpreadID: "three_card", Cards: sampleDetailCards()}
	m.layoutHeights()
	m.readingBuf.WriteString(strings.Repeat("一行解读。\n", 40))
	m.flushReadingPlain()
	m.readingVP.SetYOffset(3)
	if m.readingVP.YOffset != 3 {
		t.Fatalf("setup offset %d", m.readingVP.YOffset)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	mm := updated.(*Model)
	if _, ok := mm.state.(*CardDetailState); !ok {
		t.Fatalf("state %T", mm.state)
	}

	updated, _ = mm.Update(agentDeltaMsg{text: "新增一句"})
	mm = updated.(*Model)
	updated, _ = mm.Update(streamFlushMsg{gen: mm.streamGen})
	mm = updated.(*Model)
	if _, ok := mm.state.(*CardDetailState); !ok {
		t.Fatalf("stream left the detail view: %T", mm.state)
	}
	if !strings.Contains(mm.readingBuf.String(), "新增一句") {
		t.Fatal("delta was not applied under the detail view")
	}
	if mm.readingVP.YOffset == 3 {
		t.Fatal("flush did not move the live offset")
	}

	updated, _ = mm.Update(agentEndMsg{})
	mm = updated.(*Model)
	if _, ok := mm.state.(*CardDetailState); !ok {
		t.Fatalf("reading end closed the detail view: %T", mm.state)
	}
	if !strings.Contains(mm.readingContent, "新增一句") {
		t.Fatal("finished reading lost the streamed text")
	}

	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm = updated.(*Model)
	if _, ok := mm.state.(*FollowUpState); !ok {
		t.Fatalf("esc state %T", mm.state)
	}
	if mm.readingVP.YOffset != 3 {
		t.Fatalf("scroll offset %d, want 3", mm.readingVP.YOffset)
	}
	if !strings.Contains(mm.readingContent, "新增一句") {
		t.Fatal("leaving detail dropped the reading")
	}
}

func TestCtrlOWithoutCardsStaysPut(t *testing.T) {
	m := newTestModel()
	m.state = &ReadingState{}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	if _, ok := updated.(*Model).state.(*ReadingState); !ok {
		t.Fatalf("state %T", updated.(*Model).state)
	}
}

func TestReadingPromptIncludesSignalsOnlyWithEvidence(t *testing.T) {
	one := readingPrompt("今日指引", "single", "愚者 正位", nil)
	if strings.Contains(one, "【牌阵信号】") {
		t.Fatalf("single card prompt:\n%s", one)
	}
	signals := domain.SpreadSignals(sampleDetailCards())
	got := readingPrompt("钱和未来", "three_card", "三张牌", signals)
	for _, want := range []string{"【牌阵信号】", "火与水同时出现", "牌阵信号只引用上面的列表", "不要新开章节"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q\n%s", want, got)
		}
	}
}
