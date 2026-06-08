# 塔罗专业性增强设计

> 日期：2026-06-08
> 状态：待审核
> 范围：牌面数据充实 + 双 Prompt 架构 + 全局知识文件

## 背景

当前 CLI MVP 的牌面数据只有"大众科普"深度——每张牌仅含 5 个关键词 + 1 句正位/逆位含义。系统 prompt 要求 agent 做元素分析、牌间关系解读，但数据层完全没有这些信息，导致 agent 只能依赖 LLM 自身知识"编"，质量不可控。

本设计的目标：将牌面数据提升到专业塔罗师水平，同时支持 B2B（专业）和 C2C（通俗）两种解读模式。

## 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 方案类型 | JSON 数据充实 + 双 Prompt | 改动集中，不引入新代码包 |
| 数据来源 | 开源数据参考 + AI 生成 + 人工审核 | 平衡速度和准确度 |
| 深度切换 | 两套独立 system prompt | 各自优化，避免互相妥协 |
| 术语边界 | 保留专业体系，去掉迷信表述 | 专业性不能丢，但不搞玄学 |
| 画面象征 | 纯文字描述 | 当前无图片文件，文字足够 |
| 牌间关系 | 靠 LLM 利用丰富数据自行综合 | 真正的塔罗师也是靠理解而非背表 |

## 第一节：牌面数据 Schema 扩展

### 大牌（Major Arcana）新增字段

在现有字段基础上新增：

```json
{
  "element": "air",
  "astrology": {
    "planet": "天王星",
    "zodiac": "",
    "note": "传统对应天王星，部分体系对应风元素"
  },
  "numerology": {
    "number": 0,
    "meaning": "无限可能、起点之前、虚无与圆满",
    "note": "0 是唯一不在愚者之旅中的数字，代表纯粹潜能"
  },
  "imagery": "一个年轻人站在悬崖边，背着小包袱，仰望天空，身旁有一只小白狗。他似乎毫不在意脚下的深渊，象征着对未知的天真信任。",
  "keywords_context": {
    "love": ["新恋情的开始", "无条件的信任", "关系中的冒险"],
    "career": ["新项目启动", "创业精神", "不走寻常路"],
    "growth": ["拥抱未知", "放下恐惧", "回归初心"]
  }
}
```

字段说明：
- `element`：四元素之一（fire/water/air/earth）
- `astrology.planet`：守护行星（中文）
- `astrology.zodiac`：对应星座（如有）
- `astrology.note`：补充说明
- `numerology.number`：牌的序号
- `numerology.meaning`：数字在愚者之旅中的象征含义
- `numerology.note`：特殊情况说明
- `imagery`：RWS 牌面关键视觉元素描述（2-3 句）
- `keywords_context`：按 love/career/growth 三个场景维度的关键词

### 小牌（Minor Arcana）新增字段

```json
{
  "element": "fire",
  "numerology": {
    "number": 1,
    "meaning": "起源、纯粹能量、种子"
  },
  "astrology": {
    "sub_influence": "白羊座/狮子座/射手座（火象星座共性）"
  },
  "imagery": "一只手从云中伸出，握着一根发芽的权杖，背景是远方的城堡和山脉。象征创造能量的迸发和新机会的到来。",
  "keywords_context": {
    "love": ["热烈的新恋情", "激情重燃"],
    "career": ["新项目", "创业", "灵感爆发"],
    "growth": ["自我发现", "勇气的觉醒"]
  }
}
```

### 宫庭牌特殊字段

在小牌字段基础上额外添加：

```json
{
  "court_role": {
    "archetype": "热情的学习者",
    "personality": "充满好奇心和冒险精神，但缺乏经验。像一个刚踏入新领域的年轻人。",
    "as_person": "可能代表一个年轻的、充满活力的人，或者你内心好奇、勇于尝试的那一面",
    "as_message": "带来新消息或新机会的信号"
  }
}
```

字段说明：
- `court_role.archetype`：人格原型概括
- `court_role.personality`：性格描述
- `court_role.as_person`：作为人物解读时的含义
- `court_role.as_message`：作为消息/信号解读时的含义

### 元素固定对应（不可更改）

| 花色 | 元素 | 领域 |
|------|------|------|
| 权杖 (Wands) | fire | 行动、激情、创造力、意志力 |
| 圣杯 (Cups) | water | 情感、关系、直觉、潜意识 |
| 宝剑 (Swords) | air | 思维、沟通、冲突、真相 |
| 星币 (Pentacles) | earth | 物质、健康、工作、安全感 |

### 大牌元素/占星对应表（RWS 传统）

| 牌 | 元素 | 守护行星/星座 |
|----|------|--------------|
| 0 愚者 | air | 天王星 |
| I 魔术师 | air | 水星 |
| II 女祭司 | water | 月亮 |
| III 女皇 | earth | 金星 |
| IV 皇帝 | fire | 白羊座 |
| V 教皇 | earth | 金牛座 |
| VI 恋人 | air | 双子座 |
| VII 战车 | water | 巨蟹座 |
| VIII 力量 | fire | 狮子座 |
| IX 隐士 | earth | 处女座 |
| X 命运之轮 | fire | 木星 |
| XI 正义 | air | 天秤座 |
| XII 倒吊人 | water | 海王星 |
| XIII 死神 | water | 天蝎座 |
| XIV 节制 | fire | 射手座 |
| XV 恶魔 | earth | 摩羯座 |
| XVI 塔 | fire | 火星 |
| XVII 星星 | air | 水瓶座 |
| XVIII 月亮 | water | 双鱼座 |
| XIX 太阳 | fire | 太阳 |
| XX 审判 | fire | 冥王星 |
| XXI 世界 | earth | 土星 |

## 第二节：全局知识文件

### `internal/store/assets/knowledge/elements.json`

```json
{
  "interactions": {
    "fire_fire": "能量叠加，行动力极强但可能急躁冲动",
    "fire_water": "对立元素，产生蒸汽——情感与行动的张力",
    "fire_earth": "火生土——激情转化为实际成果",
    "fire_air": "风助火势——想法激发行动，但也可能失控",
    "water_water": "情感深度叠加，直觉强烈但可能过度情绪化",
    "water_earth": "水生土——情感滋养现实，带来稳定成长",
    "water_air": "对立元素——理性与感性的拉扯",
    "earth_earth": "极度务实，稳定但可能固执停滞",
    "earth_air": "对立元素——理想与现实的冲突",
    "air_air": "思维活跃，分析力强但可能过度思虑"
  },
  "suit_meanings": {
    "wands": {"element": "fire", "domain": "行动、激情、创造力、意志力"},
    "cups": {"element": "water", "domain": "情感、关系、直觉、潜意识"},
    "swords": {"element": "air", "domain": "思维、沟通、冲突、真相"},
    "pentacles": {"element": "earth", "domain": "物质、健康、工作、安全感"}
  },
  "numerology": {
    "1": "起源、新开始、纯粹能量",
    "2": "选择、平衡、二元性",
    "3": "创造力、成长、表达",
    "4": "稳定、结构、基础",
    "5": "冲突、变化、考验",
    "6": "和谐、责任、调整",
    "7": "内在探索、挑战、灵性",
    "8": "力量、成就、掌控",
    "9": "完成、智慧、接近终点",
    "10": "圆满、结束、新循环前的终章"
  }
}
```

## 第三节：双 Prompt 架构

### Prompt 文件

| 文件 | 用途 |
|------|------|
| `internal/agents/prompts/system_pro.md` | 专业模式，面向塔罗师和深度用户 |
| `internal/agents/prompts/system_casual.md` | 通俗模式，面向普通用户 |

### 两个版本的对比

| 维度 | 专业版 | 通俗版 |
|------|--------|--------|
| 解读框架 | 6 层完整框架 + 元素/数字/占星深度分析 | 3 层简化框架：牌面概述 → 逐张解读 → 核心建议 |
| 元素分析 | 使用"火元素""水元素"等术语，分析元素互动 | 用"行动力""情感力""思考力""稳定感"代替，做同样分析但不出现术语 |
| 术语密度 | 允许专业术语，括号注释含义 | 不使用专业术语，全部转译为日常语言 |
| 牌间关系 | 分析元素互动、数字序列、经典牌对 | 讲"这些牌在一起讲了什么故事" |
| 逆位解读 | 四种逆位理解方式，根据牌的类型选择最合适的方式 | 简化为"这张牌的能量被限制或内化" |
| 宫庭牌 | 分析人格原型、可能代表的人物、性格面向 | 只描述这股能量的感觉 |
| 输出长度 | 1500-2500 字 | 800-1200 字 |
| 反思问题 | 2-3 个深入问题 | 1 个轻松问题 |

### 共通规则（两个版本都遵守）

- 解读是对话，不是宣判
- 语言克制，用"可能""暗示"不用"注定""肯定"
- 逆位不是坏牌
- 不做第三人称解读
- 不制造焦虑
- 先共情再分析
- 不使用：预测、算命、注定、转运、大师、灵媒、命运
- 不宣称：准确性、科学性、确定性
- 不给：具体时间预测、生死判断、医疗建议、法律或财务建议

## 第四节：Agent 集成

### 代码变更

#### `internal/store/store.go`

新增 `ElementKnowledge` 字段，通过 `go:embed` 加载 `assets/knowledge/elements.json`：

```go
type ElementKnowledge struct {
    Interactions  map[string]string     `json:"interactions"`
    SuitMeanings  map[string]SuitInfo   `json:"suit_meanings"`
    Numerology    map[string]string     `json:"numerology"`
}

type Store struct {
    Cards    *CardStore
    Spreads  *SpreadStore
    Elements *ElementKnowledge
}
```

agent 通过 `store.Elements` 访问元素互动规则和数理含义。

#### `internal/agents/build.go`

```go
type AgentMode string

const (
    ModeProfessional AgentMode = "professional"
    ModeCasual       AgentMode = "casual"
)

// BuildAgent 新增 mode 参数
func BuildAgent(s *store.Store, mode AgentMode) *agentcore.Agent {
    promptFile := "prompts/system_pro.md"
    if mode == ModeCasual {
        promptFile = "prompts/system_casual.md"
    }
    // ... 其余不变
}
```

#### `internal/bootstrap/bootstrap.go`

```go
type Config struct {
    // 现有字段保留
    Mode string `json:"mode"` // "professional" 或 "casual"
}
```

默认值：`"professional"`

首次运行向导新增一步：
```
选择解读模式：
  1. 专业模式 — 包含元素、占星、数字等深度分析
  2. 轻松模式 — 温暖对话式解读，不使用专业术语
请选择 [1/2]：
```

#### `cmd/tarot-agent/main.go`

支持命令行参数：
- `--mode pro` → `ModeProfessional`
- `--mode casual` → `ModeCasual`
- 未指定时使用 config.json 中的值

#### `internal/host/tui/model.go`

状态栏显示当前模式标识：`[专业模式]` 或 `[轻松模式]`

### 不变的部分

- 4 个 agent 工具（get_card_meaning, get_spread_layout, get_disclaimer, save_reading）
- ReadingGuard StopGuard
- TUI 状态机流程
- 卡牌抽取逻辑
- 数据存储格式（JSON/JSONL）

## 第五节：测试策略

### 新增数据层测试

**`internal/store/cards_test.go` 新增用例**：

- `TestCardEnrichedFields`：每张牌必须有 element、astrology、numerology、imagery；大牌必须有 keywords_context；宫庭牌必须有 court_role
- `TestElementConsistency`：权杖→fire、圣杯→water、宝剑→air、星币→earth；大牌元素必须是四元素之一
- `TestNumerologyRange`：1-10 的 numerology.number 必须与牌的 number 一致

**`internal/store/knowledge_test.go`（新增）**：

- `TestElementInteractions`：10 种元素组合必须全部存在
- `TestNumerologyEntries`：1-10 每个数字必须有含义

### 新增集成测试

- `TestDualPromptLoading`：两种 mode 各自加载对应 prompt 文件不报错，且包含关键段落

### 不变的测试

现有所有测试保留不变。

### 覆盖率

新增代码覆盖率 ≥ 80%。

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/store/assets/cards/major_arcana.json` | 修改 | 新增 element/astrology/numerology/imagery/keywords_context |
| `internal/store/assets/cards/minor_wands.json` | 修改 | 新增 element/numerology/astrology/imagery/keywords_context，宫庭牌新增 court_role |
| `internal/store/assets/cards/minor_cups.json` | 同上 | |
| `internal/store/assets/cards/minor_swords.json` | 同上 | |
| `internal/store/assets/cards/minor_pentacles.json` | 同上 | |
| `internal/store/assets/knowledge/elements.json` | 新增 | 元素互动、花色含义、数理通用含义 |
| `internal/store/store.go` | 修改 | 加载 knowledge 目录 |
| `internal/agents/prompts/system_pro.md` | 新增 | 专业版 system prompt |
| `internal/agents/prompts/system_casual.md` | 新增（从现有 system.md 演化） | 通俗版 system prompt |
| `internal/agents/prompts/system.md` | 删除（拆分为上面两个） | |
| `internal/agents/build.go` | 修改 | 新增 AgentMode 参数，选择 prompt |
| `internal/bootstrap/bootstrap.go` | 修改 | Config 新增 Mode 字段 |
| `cmd/tarot-agent/main.go` | 修改 | 新增 --mode 命令行参数 |
| `internal/host/tui/model.go` | 修改 | 状态栏显示模式标识 |
| `internal/store/cards_test.go` | 修改 | 新增字段完整性测试 |
| `internal/store/knowledge_test.go` | 新增 | 全局知识文件测试 |

## 数据量估算

| 数据类别 | 条目数 | 新增数据点/条 | 总计 |
|---------|--------|-------------|------|
| 大牌 | 22 | ~8 | ~176 |
| 小牌（含宫庭牌） | 56 | ~6 | ~336 |
| 全局知识 | 1 | - | ~30 条规则 |
| **合计** | | | **~540 个数据点** |

## 执行顺序

1. 开源数据调研 + AI 生成结构化 JSON
2. 你审核关键数据（大牌占星对应、元素互动规则）
3. 写入 JSON 文件
4. 实现代码变更（AgentMode、config、TUI）
5. 编写双 Prompt
6. 测试
7. 集成验证
