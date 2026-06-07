package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/store"
)

// saveReadingTool implements agentcore.Tool for saving a reading record.
type saveReadingTool struct {
	store *store.Store
}

// SaveReadingTool creates a tool for saving reading records.
func SaveReadingTool(s *store.Store) *saveReadingTool {
	return &saveReadingTool{store: s}
}

func (t *saveReadingTool) Name() string { return "save_reading" }

func (t *saveReadingTool) Description() string {
	return "保存本次占卜记录。在完成解读后调用此工具，将占卜记录持久化到本地。"
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
				"description": "解读的核心洞察摘要（1-2 句话）",
			},
		},
		"required": []string{"question", "spread_id", "summary"},
	}
}

func (t *saveReadingTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Question string `json:"question"`
		SpreadID string `json:"spread_id"`
		Summary  string `json:"summary"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse save_reading args: %w", err)
	}

	reading := &domain.Reading{
		ID:             generateReadingID(),
		Question:       params.Question,
		SpreadID:       params.SpreadID,
		Interpretation: params.Summary,
		CreatedAt:      time.Now(),
	}

	if err := t.store.Readings.Append(reading); err != nil {
		return nil, fmt.Errorf("save reading: %w", err)
	}

	result := map[string]string{
		"status":  "saved",
		"reading_id": reading.ID,
	}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func generateReadingID() string {
	return fmt.Sprintf("r_%d", time.Now().UnixNano())
}
