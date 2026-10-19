// Package cardlookup resolves a tarot card from the local deck.
// It does not call a model.
package cardlookup

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
)

const candidateCap = 8

// Lookup finds one card. A suit name or a partial name returns *CandidatesError.
// A complete miss returns *NotFoundError.
func Lookup(cards []domain.Card, query string) (*domain.Card, error) {
	ix := buildIndex(cards)
	q := fold(query)
	if q == "" {
		return nil, fmt.Errorf("请写出牌名，或用 --list 列出全部")
	}
	if card, ok := ix.exact[q]; ok {
		found := card
		return &found, nil
	}
	if group := ix.group(q); len(group) > 0 {
		return nil, &CandidatesError{Query: strings.TrimSpace(query), Cards: group}
	}
	matches := ix.partial(q)
	if len(matches) == 0 {
		return nil, &NotFoundError{Query: strings.TrimSpace(query)}
	}
	err := &CandidatesError{Query: strings.TrimSpace(query), Cards: matches}
	if len(matches) > candidateCap {
		err.Cards = append([]domain.Card(nil), matches[:candidateCap]...)
		err.More = len(matches) - candidateCap
	}
	return nil, err
}

// CandidatesError means the query is a suit or a partial name, not one card.
type CandidatesError struct {
	Query string
	Cards []domain.Card
	More  int
}

func (e *CandidatesError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "没有精确匹配「%s」。可能是：\n", e.Query)
	for _, card := range e.Cards {
		fmt.Fprintf(&b, "%s  %s\n", card.NameCN, card.NameEN)
	}
	if e.More > 0 {
		fmt.Fprintf(&b, "还有 %d 张。写全牌名，或用 --list 查看全部。\n", e.More)
	}
	return strings.TrimRight(b.String(), "\n")
}

// NotFoundError means nothing in the deck is close to the query.
type NotFoundError struct {
	Query string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("没有找到「%s」。可以写中文名、英文名或大阿卡那编号（0–21），也可以用 --list 查看全部 78 张。", e.Query)
}

type index struct {
	exact map[string]domain.Card
	cards []domain.Card
}

func buildIndex(cards []domain.Card) *index {
	ix := &index{
		exact: make(map[string]domain.Card, len(cards)*8),
		cards: Sort(cards),
	}
	for _, card := range ix.cards {
		ix.add(card.NameCN, card)
		ix.add(card.NameEN, card)
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(card.NameEN)), "the ") {
			ix.add(strings.TrimPrefix(fold(card.NameEN), "the"), card)
		}
		for _, alias := range card.NameAliases {
			ix.add(alias, card)
		}
		for _, form := range nameAliases(card.NameCN) {
			ix.add(form, card)
		}
		ix.addRankKeys(card)
	}
	return ix
}

func (ix *index) add(key string, card domain.Card) {
	key = fold(key)
	if key == "" {
		return
	}
	if prev, ok := ix.exact[key]; ok && prev.ID != card.ID {
		return
	}
	ix.exact[key] = card
}

func (ix *index) addRankKeys(card domain.Card) {
	if card.Arcana == domain.MajorArcana {
		n := strconv.Itoa(card.Number)
		ix.add(n, card)
		if card.Number < 10 {
			ix.add(fmt.Sprintf("%02d", card.Number), card)
		}
		ix.add("大阿卡那"+n, card)
		ix.add("大阿卡纳"+n, card)
		ix.add("major"+n, card)
		return
	}
	suit := suitCN[card.Suit]
	suitEN := string(card.Suit)
	if suit == "" || suitEN == "" || card.Number < 1 || card.Number > 10 {
		return
	}
	ix.add(suit+cnRank[card.Number], card)
	ix.add(suit+strconv.Itoa(card.Number), card)
	ix.add(strconv.Itoa(card.Number)+"of"+suitEN, card)
	for _, form := range nameAliases(suit + cnRank[card.Number]) {
		ix.add(form, card)
	}
	for _, form := range nameAliases(suit + strconv.Itoa(card.Number)) {
		ix.add(form, card)
	}
	if card.Number == 1 {
		for _, rank := range []string{"A", "Ace"} {
			ix.add(suit+rank, card)
			for _, form := range nameAliases(suit + rank) {
				ix.add(form, card)
			}
		}
	}
}

func (ix *index) group(query string) []domain.Card {
	var suit domain.Suit
	var arcana domain.ArcanaType
	switch query {
	case "权杖", "wands":
		suit = domain.SuitWands
	case "圣杯", "杯", "cups":
		suit = domain.SuitCups
	case "宝剑", "剑", "swords":
		suit = domain.SuitSwords
	case "星币", "钱币", "金币", "pentacles":
		suit = domain.SuitPentacles
	case "大阿卡那", "大阿卡纳", "major", "majorarcana":
		arcana = domain.MajorArcana
	case "小阿卡那", "minor", "minorarcana":
		arcana = domain.MinorArcana
	default:
		return nil
	}
	var out []domain.Card
	for _, card := range ix.cards {
		if arcana != "" && card.Arcana == arcana {
			out = append(out, card)
			continue
		}
		if suit != "" && card.Suit == suit {
			out = append(out, card)
		}
	}
	return out
}

func (ix *index) partial(query string) []domain.Card {
	var matches []domain.Card
	for _, card := range ix.cards {
		if containsFolded(card, query) {
			matches = append(matches, card)
		}
	}
	return matches
}

func containsFolded(card domain.Card, query string) bool {
	if strings.Contains(fold(card.NameCN), query) || strings.Contains(fold(card.NameEN), query) {
		return true
	}
	for _, alias := range card.NameAliases {
		if strings.Contains(fold(alias), query) {
			return true
		}
	}
	for _, form := range nameAliases(card.NameCN) {
		if strings.Contains(fold(form), query) {
			return true
		}
	}
	return false
}

// Sort orders majors, then suits, then number.
func Sort(cards []domain.Card) []domain.Card {
	out := append([]domain.Card(nil), cards...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Arcana != b.Arcana {
			return a.Arcana == domain.MajorArcana
		}
		if suitOrder(a.Suit) != suitOrder(b.Suit) {
			return suitOrder(a.Suit) < suitOrder(b.Suit)
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})
	return out
}

func suitOrder(s domain.Suit) int {
	switch s {
	case "":
		return 0
	case domain.SuitWands:
		return 1
	case domain.SuitCups:
		return 2
	case domain.SuitSwords:
		return 3
	case domain.SuitPentacles:
		return 4
	default:
		return 9
	}
}

var suitCN = map[domain.Suit]string{
	domain.SuitWands:     "权杖",
	domain.SuitCups:      "圣杯",
	domain.SuitSwords:    "宝剑",
	domain.SuitPentacles: "星币",
}

var cnRank = []string{"", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}

func nameAliases(name string) []string {
	if name == "" {
		return nil
	}
	var forms []string
	replacements := []struct{ from, to string }{
		{"星币", "钱币"},
		{"星币", "金币"},
		{"宝剑", "剑"},
		{"圣杯", "杯"},
	}
	for _, r := range replacements {
		if strings.Contains(name, r.from) {
			forms = append(forms, strings.ReplaceAll(name, r.from, r.to))
		}
	}
	if name == "女皇" {
		forms = append(forms, "皇后")
	}
	return forms
}

func fold(s string) string {
	s = normalizeDigits(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '　', '-', '_', '\'', '’', '.', ',', ':', '·':
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func normalizeDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '０' && r <= '９' {
			b.WriteRune('0' + (r - '０'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
