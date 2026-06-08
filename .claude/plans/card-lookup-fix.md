# 修复卡牌查询名称匹配问题

## 问题

`card_meaning.go` 的 `Execute` 方法使用精确匹配（`==`）查找卡牌名称。LLM agent 经常用中文数字查询（如"权杖一"），但数据中 Ace 牌叫"权杖Ace"，导致查不到。

## 修复方案

### 1. Card struct 新增 `NameAliases` 字段

在 `domain/card.go` 的 `Card` struct 中新增：
```go
NameAliases []string `json:"name_aliases,omitempty"`
```

### 2. 所有 Ace 牌添加别名

在 4 个 minor arcana JSON 文件中，给每张 Ace 牌添加：
```json
"name_aliases": ["权杖一"]
```
即：
- 权杖Ace → aliases: ["权杖一"]
- 圣杯Ace → aliases: ["圣杯一"]
- 宝剑Ace → aliases: ["宝剑一"]
- 星币Ace → aliases: ["星币一"]

### 3. card_meaning.go 增加模糊匹配

修改 `Execute` 方法，当精确匹配失败时，检查 `NameAliases`。

### 4. system prompt 提示用 card_id

在两个 system prompt 的"查牌义"步骤中，提示 agent 优先使用 `card_id` 而非 `card_name`。

## 文件变更

| 文件 | 变更 |
|------|------|
| `internal/domain/card.go` | Card 新增 NameAliases |
| `internal/store/assets/cards/minor_*.json` (×4) | Ace 牌添加 name_aliases |
| `internal/tools/card_meaning.go` | 匹配逻辑增加 aliases 检查 |
| `internal/agents/prompts/system_pro.md` | 提示用 card_id |
| `internal/agents/prompts/system_casual.md` | 提示用 card_id |
| `internal/store/cards_test.go` | 新增别名测试 |
