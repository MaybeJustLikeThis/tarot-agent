package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voocel/tarot-agent/internal/store"
)

// spreadLayoutTool implements agentcore.Tool for querying spread layouts.
type spreadLayoutTool struct {
	store *store.Store
}

// GetSpreadLayoutTool creates a tool that returns spread layout details.
func GetSpreadLayoutTool(s *store.Store) *spreadLayoutTool {
	return &spreadLayoutTool{store: s}
}

func (t *spreadLayoutTool) Name() string { return "get_spread_layout" }
func (t *spreadLayoutTool) Description() string {
	return "查询牌阵的布局信息，包括每个位置的名称和含义。可查询所有牌阵或指定牌阵。"
}

func (t *spreadLayoutTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spread_id": map[string]any{
				"type":        "string",
				"description": "牌阵 ID：single、three_card、celtic_cross。不传则返回所有牌阵列表。",
			},
		},
	}
}

func (t *spreadLayoutTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		SpreadID string `json:"spread_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	if params.SpreadID == "" {
		spreads := t.store.Spreads.GetAll()
		data, err := json.MarshalIndent(spreads, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal spreads: %w", err)
		}
		return json.RawMessage(data), nil
	}

	spread, err := t.store.Spreads.GetByID(params.SpreadID)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(spread, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal spread: %w", err)
	}
	return json.RawMessage(data), nil
}
