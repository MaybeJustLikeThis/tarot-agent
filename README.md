# Tarot Agent — AI 塔罗占卜 CLI

AI 驱动的塔罗占卜命令行工具。通过牌面的象征语言，帮助你重新审视自己的处境，发现被忽略的角度。

> 塔罗是镜子，不是预言机。

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 功能

- 🃏 **78 张韦特塔罗牌** — 完整正/逆位牌义（22 大阿卡纳 + 56 小阿卡纳）
- 🔮 **3 种牌阵** — 单张牌（快速指引）、三张牌（过去/现在/未来）、凯尔特十字（深度分析）
- ✨ **翻牌动画** — 逐张揭示，仪式感拉满
- 🤖 **AI 深度解读** — 不是模板复述，是结合你具体情况的个性化分析
- 💬 **多轮对话** — 解读后可以继续追问，深入探讨
- 💾 **记录持久化** — 每次占卜自动保存到本地

## 快速开始

### 1. 环境要求

- **Go 1.21+**（[安装指南](https://go.dev/doc/install)）
- **DeepSeek API Key**（[免费获取](https://platform.deepseek.com/api_keys)）

验证 Go 安装：
```bash
go version
# 应输出 go version go1.21.x 或更高
```

### 2. 克隆 & 编译

```bash
# 克隆仓库
git clone https://github.com/MaybeJustLikeThis/tarot-agent.git
cd tarot-agent

# 编译（Windows/macOS/Linux 通用）
go build -o tarot-agent ./cmd/tarot-agent
```

Windows 用户会生成 `tarot-agent.exe`，macOS/Linux 生成 `tarot-agent`。

### 3. 运行

```bash
# macOS / Linux
./tarot-agent

# Windows (PowerShell)
.\tarot-agent.exe

# 或者直接运行源码（不编译）
go run ./cmd/tarot-agent
```

**首次运行**会引导你粘贴 API Key：

```
  ╔═══════════════════════════════════════╗
  ║       星语 Tarot Agent — 首次设置      ║
  ╚═══════════════════════════════════════╝

  本工具需要 DeepSeek API Key 来驱动 AI 解读。
  获取方式：https://platform.deepseek.com/api_keys

  请粘贴你的 API Key（sk-xxx）：>
```

粘贴你的 Key 后回车即可。配置会保存到 `~/.tarot-agent/config.json`，之后无需重复输入。

### 4. 开始占卜

```
  ✦ 星语 Tarot Agent                                        等待输入
  ▍牌面 ────────────────────     │▍解读 ────────────────────
                                 │
  等待抽牌...                    │
                                 │
                                 │
──────────────────────────────────────────────────────────────
  说说你的情况和想问的问题：
  > 我最近在纠结要不要换工作
```

1. 输入你的情况和问题
2. 选择牌阵（1/2/3）
3. 看翻牌动画
4. 阅读 AI 解读
5. 可以继续追问

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DEEPSEEK_API_KEY` | DeepSeek API Key | 无（必填，或通过首次引导设置） |
| `TAROT_API_KEY` | 同上（优先级更高） | 无 |
| `TAROT_BASE_URL` | 自定义 API 地址 | `https://api.deepseek.com` |
| `TAROT_MODEL` | 自定义模型名 | `deepseek-chat` |
| `TAROT_LOG_LEVEL` | 日志级别 | `info` |

示例（使用环境变量跳过首次引导）：
```bash
# macOS / Linux
export DEEPSEEK_API_KEY=sk-xxxxxxxxxxxx
./tarot-agent

# Windows PowerShell
$env:DEEPSEEK_API_KEY="sk-xxxxxxxxxxxx"
.\tarot-agent.exe
```

## 使用其他 LLM

tarot-agent 兼容所有 OpenAI API 格式的 LLM：

```bash
# 使用 OpenAI
export TAROT_BASE_URL="https://api.openai.com/v1"
export TAROT_MODEL="gpt-4o"
export TAROT_API_KEY="sk-xxxxxxxxxxxx"

# 使用本地 Ollama
export TAROT_BASE_URL="http://localhost:11434/v1"
export TAROT_MODEL="qwen2.5"
export TAROT_API_KEY="ollama"
```

## 键盘快捷键

| 按键 | 功能 |
|------|------|
| `Enter` | 提交输入 / 追问 |
| `q` | 退出 |
| `↑` / `k` | 解读区向上滚动 |
| `↓` / `j` | 解读区向下滚动 |
| `PgUp` / `PgDn` | 解读区翻页 |
| `Ctrl+C` | 强制退出 |

## 终端要求

- **最小尺寸**：80×20（字符），小于此会提示调整窗口
- **推荐尺寸**：120×40 或更大，体验最佳
- **支持终端**：Windows Terminal、iTerm2、GNOME Terminal、Alacritty 等现代终端

## 项目结构

```
tarot-agent/
├── cmd/tarot-agent/        ← 程序入口
├── internal/
│   ├── agents/             ← Agent 构建 + Prompt 模板（go:embed）
│   ├── bootstrap/          ← 配置加载、模型初始化、首次引导
│   ├── domain/             ← 核心实体（Card, Spread, Reading）
│   ├── host/
│   │   ├── tui/            ← Bubble Tea TUI（状态机 + 并排布局）
│   │   ├── reminder/       ← StopGuard 防偷懒机制
│   │   └── cli.go          ← 旧版 CLI（已弃用）
│   ├── store/              ← 数据层（78 张牌 JSON + 牌阵 + ReadingStore）
│   └── tools/              ← Agent Tools（抽牌、查牌义、牌阵、免责）
├── assets/                 ← 静态资源副本
├── go.mod
├── Makefile
└── README.md
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| Agent 框架 | [agentcore](https://github.com/voocel/agentcore) |
| LLM | DeepSeek-V3（兼容 OpenAI API） |
| TUI | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) |
| 存储 | JSON/JSONL（go:embed 嵌入牌义库） |
| Prompt | go:embed 嵌入 Markdown，改 prompt 不改代码 |

## 常见问题

**Q: `go: command not found`**
A: Go 未安装或未加入 PATH。参考 https://go.dev/doc/install

**Q: `API key is required`**
A: 未设置 API Key。运行程序后按引导粘贴 Key，或设置环境变量 `DEEPSEEK_API_KEY`。

**Q: 终端显示乱码**
A: 确保终端支持 UTF-8 和 Unicode 字符。Windows 用户推荐使用 Windows Terminal。

**Q: 解读太慢**
A: DeepSeek-V3 响应通常需要 5-15 秒。如果超过 30 秒，检查网络连接或 API 状态。

**Q: 可以用其他模型吗？**
A: 可以，任何兼容 OpenAI API 的模型都能用。设置 `TAROT_BASE_URL` 和 `TAROT_MODEL` 即可。

## 免责声明

塔罗是一种自我探索的工具，所有解读仅供娱乐和自我反思参考。本工具不提供任何形式的命运预测、医疗建议、法律建议或财务建议。

## License

MIT
