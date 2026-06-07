# Tarot Agent — AI 塔罗占卜 CLI

AI 驱动的塔罗占卜工具。双轨策略：B2B 从业者工具（¥29-99/月）+ C 端自助占卜（¥3-10/次）。

## 快速开始

```bash
# 编译
make build

# 运行（首次会引导输入 API Key）
make run

# 或直接
go run ./cmd/tarot-agent/
```

## 项目结构

```
tarot-agent/
├── cmd/tarot-agent/        ← 程序入口
├── internal/
│   ├── agents/             ← Agent 构建 + Prompt 模板
│   │   └── prompts/        ← System prompt（go:embed）
│   ├── bootstrap/          ← 配置加载、模型初始化、首次引导
│   ├── domain/             ← 核心实体（Card, Spread, Reading）
│   ├── host/               ← CLI 交互层
│   ├── store/              ← 数据层（牌义 JSON、牌阵、ReadingStore）
│   │   └── assets/         ← go:embed 数据源
│   └── tools/              ← Agent Tools（抽牌、查牌义、牌阵、免责）
├── assets/                 ← 顶层静态资源副本
├── docs/
│   ├── business/           ← 商业文档（市场调研、战略、品牌、产品、财务、验证）
│   ├── architecture.md     ← 系统架构设计
│   ├── tech-spec.md        ← 技术规格
│   ├── prd.md              ← 产品需求文档
│   ├── requirements.md     ← 用户故事
│   └── project-plan.md     ← 项目计划（Sprint 划分）
├── go.mod
├── Makefile
└── README.md
```

## 技术栈

- **语言**: Go
- **Agent 框架**: [agentcore](https://github.com/voocel/agentcore)（Coordinator + SubAgent + Tool）
- **LLM**: DeepSeek-V3（主力） + Qwen（备用）
- **存储**: JSON/JSONL 文件（go:embed 嵌入牌义库）
- **Prompt 管理**: go:embed 嵌入 Markdown，改 prompt 不改代码

## 功能

- 78 张韦特塔罗牌完整牌义（22 大阿卡纳 + 56 小阿卡纳）
- 3 种牌阵：单张牌、三张牌、凯尔特十字
- AI 个性化解读（温暖语调，非模板化）
- 多轮追问对话
- 占卜记录持久化（JSONL）
- 首次使用引导（Ctrl+V 粘贴 API Key）

## 配置

首次运行会自动引导设置。之后配置存储在 `~/.tarot-agent/config.json`。

环境变量可覆盖：
- `DEEPSEEK_API_KEY` / `TAROT_API_KEY` — API Key
- `TAROT_BASE_URL` — 自定义 API 地址
- `TAROT_MODEL` — 自定义模型名
- `TAROT_LOG_LEVEL` — 日志级别（debug/info/warn/error）

## 免责声明

塔罗是一种自我探索的工具，所有解读仅供娱乐和自我反思参考。
