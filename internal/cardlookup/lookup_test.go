package cardlookup

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

func deck(t *testing.T) []domain.Card {
	t.Helper()
	s, err := store.New()
	if err != nil {
		t.Fatal(err)
	}
	cards := s.Cards.GetAll()
	if len(cards) != 78 {
		t.Fatalf("deck size %d", len(cards))
	}
	return cards
}

func TestLookupNamesAliasesEnglishAndNumber(t *testing.T) {
	cards := deck(t)
	cases := []struct {
		query string
		id    string
		name  string
	}{
		{"愚者", "major_00", "愚者"},
		{"权杖三", "minor_wands_03", "权杖三"},
		{"权杖3", "minor_wands_03", "权杖三"},
		{"星币骑士", "minor_pentacles_12", "星币骑士"},
		{"钱币骑士", "minor_pentacles_12", "星币骑士"},
		{"金币骑士", "minor_pentacles_12", "星币骑士"},
		{"宝剑三", "minor_swords_03", "宝剑三"},
		{"剑三", "minor_swords_03", "宝剑三"},
		{"杯二", "minor_cups_02", "圣杯二"},
		{"圣杯二", "minor_cups_02", "圣杯二"},
		{"权杖一", "minor_wands_01", "权杖Ace"},
		{"皇后", "major_03", "女皇"},
		{"女皇", "major_03", "女皇"},
		{"The Fool", "major_00", "愚者"},
		{"the fool", "major_00", "愚者"},
		{"THE  FOOL", "major_00", "愚者"},
		{"fool", "major_00", "愚者"},
		{"Three of Wands", "minor_wands_03", "权杖三"},
		{"three of wands", "minor_wands_03", "权杖三"},
		{"3 of wands", "minor_wands_03", "权杖三"},
		{"0", "major_00", "愚者"},
		{"00", "major_00", "愚者"},
		{"21", "major_21", "世界"},
		{"大阿卡那3", "major_03", "女皇"},
	}
	for _, tc := range cases {
		got, err := Lookup(cards, tc.query)
		if err != nil {
			t.Fatalf("%q: %v", tc.query, err)
		}
		if got.ID != tc.id || got.NameCN != tc.name {
			t.Fatalf("%q: got %s %s", tc.query, got.ID, got.NameCN)
		}
	}
}

func TestLookupSuitCandidatesAndMiss(t *testing.T) {
	cards := deck(t)
	_, err := Lookup(cards, "权杖")
	var amb *CandidatesError
	if !errors.As(err, &amb) {
		t.Fatalf("expected candidates, got %v", err)
	}
	if len(amb.Cards) != 14 || amb.More != 0 {
		t.Fatalf("wands candidates %d more %d", len(amb.Cards), amb.More)
	}
	if amb.Cards[0].NameCN != "权杖Ace" || amb.Cards[len(amb.Cards)-1].NameCN != "权杖国王" {
		t.Fatalf("wands ends: %s .. %s", amb.Cards[0].NameCN, amb.Cards[len(amb.Cards)-1].NameCN)
	}
	if !strings.Contains(err.Error(), "权杖Ace") || !strings.Contains(err.Error(), "没有精确匹配") {
		t.Fatalf("message:\n%s", err)
	}

	_, err = Lookup(cards, "没有这张牌xyz")
	var miss *NotFoundError
	if !errors.As(err, &miss) {
		t.Fatalf("expected miss, got %v", err)
	}
	if !strings.Contains(err.Error(), "没有找到") || !strings.Contains(err.Error(), "--list") {
		t.Fatalf("miss message:\n%s", err)
	}
}

func TestEveryCardMatchesItsOwnNames(t *testing.T) {
	cards := deck(t)
	seen := map[string]struct{}{}
	for _, card := range cards {
		seen[card.ID] = struct{}{}
		for _, query := range []string{card.NameCN, card.NameEN, strings.ToLower(card.NameEN), strings.ReplaceAll(card.NameEN, " ", "")} {
			got, err := Lookup(cards, query)
			if err != nil {
				t.Fatalf("%s query %q: %v", card.ID, query, err)
			}
			if got.ID != card.ID {
				t.Fatalf("%s query %q resolved to %s", card.ID, query, got.ID)
			}
		}
	}
	if len(seen) != 78 {
		t.Fatalf("unique ids %d", len(seen))
	}
}

func TestFormatIncludesOrientationAndSkipsEmptyExtra(t *testing.T) {
	cards := deck(t)
	fool, err := Lookup(cards, "愚者")
	if err != nil {
		t.Fatal(err)
	}
	text := Format(*fool)
	for _, want := range []string{"愚者", "The Fool", "大阿卡那", "编号：0", "正位：", "逆位：", "正位含义", "逆位含义", "元素：风", "占星："} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
	knight, err := Lookup(cards, "星币骑士")
	if err != nil {
		t.Fatal(err)
	}
	knightText := Format(*knight)
	if !strings.Contains(knightText, "正位含义") || !strings.Contains(knightText, "逆位含义") {
		t.Fatalf("knight:\n%s", knightText)
	}
	if strings.Contains(knightText, "宫廷：") && knight.CourtRole == nil {
		t.Fatal("invented a court role")
	}

	bare := Format(domain.Card{NameCN: "空牌", NameEN: "Empty", Arcana: domain.MajorArcana, Number: 9})
	if strings.Contains(bare, "元素：") || strings.Contains(bare, "占星：") || strings.Contains(bare, "意象：") {
		t.Fatalf("empty extras were invented:\n%s", bare)
	}
	if !strings.Contains(bare, "正位：—") || !strings.Contains(bare, "逆位含义\n—") {
		t.Fatalf("missing dash placeholders:\n%s", bare)
	}
}

func TestListAndRun(t *testing.T) {
	cards := deck(t)
	var buf bytes.Buffer
	if err := Run(cards, "", true, &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 78 {
		t.Fatalf("list lines %d", len(lines))
	}
	if !strings.Contains(lines[0], "愚者") || !strings.Contains(lines[0], "The Fool") {
		t.Fatalf("first line %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "星币国王") {
		t.Fatalf("last line %q", lines[len(lines)-1])
	}

	buf.Reset()
	if err := Run(cards, "愚者", false, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "正位含义") || !strings.Contains(buf.String(), "逆位含义") {
		t.Fatalf("run:\n%s", buf.String())
	}
	if err := Run(cards, "权杖", false, &buf); err == nil {
		t.Fatal("suit query should fail")
	}
}
