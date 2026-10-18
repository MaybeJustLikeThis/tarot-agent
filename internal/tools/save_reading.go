package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

// saveReadingTool acknowledges that the reading should be kept.
// The TUI writes the full record once when the interpretation finishes.
// This tool must not append, or the same reading is stored twice.
type saveReadingTool struct {
	store *store.Store
}

// SaveReadingTool creates a tool that confirms a reading should be saved.
func SaveReadingTool(s *store.Store) *saveReadingTool {
	return &saveReadingTool{store: s}
}

func (t *saveReadingTool) Name() string { return "save_reading" }

func (t *saveReadingTool) Description() string {
	return "确认本次占卜应当保存。宿主会在解读结束时把完整解读、抽到的牌、问题和牌阵写入本地。不要在参数里粘贴全文；重复调用也不会多写一条记录。"
}

func (t *saveReadingTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "用户的问题",
			},
			"spread_id": map[string]any{
				"type":        "string",
				"description": "使用的牌阵类型（single/three_card/celtic_cross）",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "可选的一句话摘要。完整正文由程序保存，这里不需要粘贴全文。",
			},
		},
		"required": []string{"question", "spread_id", "summary"},
	}
}

func (t *saveReadingTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t.store == nil {
		return nil, fmt.Errorf("save reading: store is not configured")
	}
	if len(args) > 0 {
		var params struct {
			Question string `json:"question"`
			SpreadID string `json:"spread_id"`
			Summary  string `json:"summary"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("parse save_reading args: %w", err)
		}
	}
	data, err := json.Marshal(map[string]string{
		"status": "accepted",
		"detail": "完整占卜记录会在解读结束时写入，本次调用不会另存一条。",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal save_reading result: %w", err)
	}
	return json.RawMessage(data), nil
}
