package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voocel/tarot-agent/internal/store"
)

// cardMeaningTool implements agentcore.Tool for looking up card meanings.
type cardMeaningTool struct {
	store *store.Store
}

// GetCardMeaningTool creates a tool to look up a card's meaning.
func GetCardMeaningTool(s *store.Store) *cardMeaningTool {
	return &cardMeaningTool{store: s}
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
		card, err := t.store.Cards.GetByID(params.CardID)
		if err != nil {
			return nil, err
		}
		return formatCardMeaning(card)
	}

	if params.CardName != "" {
		for _, card := range t.store.Cards.GetAll() {
			if card.NameCN == params.CardName || card.NameEN == params.CardName {
				return formatCardMeaning(card)
			}
		}
		return nil, fmt.Errorf("card %q not found", params.CardName)
	}

	return nil, fmt.Errorf("must provide card_id or card_name")
}

func formatCardMeaning(card any) (json.RawMessage, error) {
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal card: %w", err)
	}
	return json.RawMessage(data), nil
}
