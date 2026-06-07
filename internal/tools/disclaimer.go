package tools

import (
	"context"
	"encoding/json"
)

const disclaimerText = `塔罗牌解读免责声明

塔罗牌是一种自我探索和心灵成长的工具，其解读仅供参考和启发，不应被视为：
- 医学诊断或治疗建议
- 法律或财务决策的依据
- 对未来事件的确定性预测

塔罗牌揭示的是当前能量状态下的可能性，而非既定命运。
你始终拥有自由意志，能够创造自己的未来。

如有健康、法律或财务问题，请咨询相关专业人士。`

// DisclaimerText returns the disclaimer text for external use.
func DisclaimerText() string { return disclaimerText }

// disclaimerTool implements agentcore.Tool for returning the disclaimer.
type disclaimerTool struct{}

// GetDisclaimerTool creates a tool that returns the disclaimer text.
func GetDisclaimerTool() *disclaimerTool {
	return &disclaimerTool{}
}

func (t *disclaimerTool) Name() string { return "get_disclaimer" }
func (t *disclaimerTool) Description() string {
	return "获取塔罗牌解读的免责声明。在用户首次使用或需要时调用。"
}

func (t *disclaimerTool) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *disclaimerTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	data, _ := json.Marshal(disclaimerText)
	return json.RawMessage(data), nil
}
