package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/store"
)

// cardMeaningTool implements agentcore.Tool for looking up card meanings.
type cardMeaningTool struct {
	store      *store.Store
	idAliasMap map[string]domain.Card // normalized ID → card
}

// GetCardMeaningTool creates a tool to look up a card's meaning.
func GetCardMeaningTool(s *store.Store) *cardMeaningTool {
	aliasMap := buildIDAliasMap(s)
	return &cardMeaningTool{store: s, idAliasMap: aliasMap}
}

// buildIDAliasMap builds a map of normalized IDs to cards for fuzzy lookup.
// Handles LLM variations like "major_0" → "major_00", "wands_01" → "minor_wands_01".
func buildIDAliasMap(s *store.Store) map[string]domain.Card {
	m := make(map[string]domain.Card)
	for _, card := range s.Cards.GetAll() {
		normalized := normalizeCardID(card.ID)
		if normalized != "" {
			m[normalized] = card
		}
		// Map common short forms (without "minor_" prefix)
		if strings.HasPrefix(card.ID, "minor_") {
			short := strings.TrimPrefix(card.ID, "minor_")
			m[short] = card
		}
	}
	return m
}

// normalizeCardID normalizes common LLM ID variations:
// - Pads single-digit numbers: "major_0" → "major_00", "wands_1" → "wands_01"
// - Adds missing "minor_" prefix: "wands_01" → "minor_wands_01"
func normalizeCardID(id string) string {
	idx := strings.LastIndex(id, "_")
	if idx == -1 || idx == len(id)-1 {
		return ""
	}
	prefix := id[:idx]
	suffix := id[idx+1:]

	// Pad single-digit number with leading zero
	if len(suffix) == 1 && suffix[0] >= '0' && suffix[0] <= '9' {
		suffix = "0" + suffix
	}

	// Add missing "minor_" prefix for suit names
	suits := map[string]bool{"wands": true, "cups": true, "swords": true, "pentacles": true}
	if suits[prefix] {
		prefix = "minor_" + prefix
	}

	return prefix + "_" + suffix
}

func (t *cardMeaningTool) Name() string { return "get_card_meaning" }
func (t *cardMeaningTool) Description() string {
	return "查询某张塔罗牌的详细含义，包括正位和逆位的解读、关键词。按 ID 或中英文名查询。"
}

func (t *cardMeaningTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"card_id": map[string]any{
				"type":        "string",
				"description": "牌的 ID，如 'major_00'、'major_01'",
			},
			"card_name": map[string]any{
				"type":        "string",
				"description": "牌的中文名或英文名，如'愚者'或'The Fool'",
			},
		},
	}
}

func (t *cardMeaningTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		CardID   string `json:"card_id"`
		CardName string `json:"card_name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	if params.CardID != "" {
		// Exact match first
		card, err := t.store.Cards.GetByID(params.CardID)
		if err == nil {
			return formatCardMeaning(card)
		}
		// Fuzzy ID match: "major_0" → "major_00", "wands_01" → "minor_wands_01"
		if c, ok := t.idAliasMap[params.CardID]; ok {
			return formatCardMeaning(c)
		}
		normalized := normalizeCardID(params.CardID)
		if normalized != "" {
			if c, ok := t.idAliasMap[normalized]; ok {
				return formatCardMeaning(c)
			}
		}
		return nil, fmt.Errorf("card id %q not found", params.CardID)
	}

	if params.CardName != "" {
		for _, card := range t.store.Cards.GetAll() {
			if card.NameCN == params.CardName || card.NameEN == params.CardName {
				return formatCardMeaning(card)
			}
			// Check aliases
			for _, alias := range card.NameAliases {
				if alias == params.CardName {
					return formatCardMeaning(card)
				}
			}
		}
		return nil, fmt.Errorf("card %q not found", params.CardName)
	}

	return nil, fmt.Errorf("must provide card_id or card_name")
}

func formatCardMeaning(card domain.Card) (json.RawMessage, error) {
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal card: %w", err)
	}
	return json.RawMessage(data), nil
}
