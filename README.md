# Tarot Agent

AI 塔罗占卜 CLI 工具。通过牌面的象征语言，帮助你重新审视自己的处境。

> 塔罗是镜子，不是预言机。

## 功能

- 🃏 78 张韦特塔罗牌完整正/逆位牌义（22 大阿卡纳 + 56 小阿卡纳）
- 🔮 3 种牌阵：单张牌（快速指引）、三张牌（过去/现在/未来）、凯尔特十字（深度分析）
- ✨ 逐张翻牌动画，还原真实占卜仪式感
- 🤖 AI 深度解读 — 结合你的具体情境，不是模板复述
- 💬 解读后可继续追问，深入探讨
- 📖 历史记录浏览 — 随时回顾过去的占卜
- 🌅 每日一牌 — 一键获取今日指引
- 💾 占卜记录自动保存到本地
- 🔑 首次运行引导设置 API Key，自动验证连通性

## 安装

```bash
# 前置条件：Go 1.21+ (https://go.dev/doc/install)

git clone https://github.com/MaybeJustLikeThis/tarot-agent.git
cd tarot-agent
go build -o tarot-agent ./cmd/tarot-agent
```

## 运行

```bash
./tarot-agent          # macOS / Linux
.\tarot-agent.exe      # Windows PowerShell
go run ./cmd/tarot-agent  # 或直接运行源码
```

首次运行会引导粘贴 DeepSeek API Key（[免费获取](https://platform.deepseek.com/api_keys)），自动验证连通性。

## 快捷键

| 按键 | 功能 |
|------|------|
| `Enter` | 提交问题 / 追问 |
| `tab` | 查看历史记录（输入框为空时） |
| `ctrl+d` | 每日一牌 |
| `↑↓` / `jk` | 解读区滚动 |
| `PgUp/PgDn` | 翻页 |
| `esc` | 返回上一步 |
| `q` | 退出 |

## 自定义 LLM

支持 DeepSeek、OpenAI、Anthropic 三种 AI 服务商。首次运行会引导选择，之后可通过环境变量切换：

```bash
# 使用 OpenAI
export TAROT_PROVIDER="openai"
export TAROT_API_KEY="sk-xxx"
export TAROT_MODEL="gpt-4o"

# 使用 Anthropic
export TAROT_PROVIDER="anthropic"
export TAROT_API_KEY="sk-ant-xxx"
export TAROT_MODEL="claude-sonnet-4-20250514"

# 使用自建代理
export TAROT_BASE_URL="https://your-proxy.com/v1"
export TAROT_API_KEY="sk-xxx"
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `TAROT_PROVIDER` | AI 服务商：`deepseek` / `openai` / `anthropic` | `deepseek` |
| `TAROT_API_KEY` | API Key | 无（首次引导设置） |
| `TAROT_BASE_URL` | API 地址 | 按 provider 自动选择 |
| `TAROT_MODEL` | 模型名 | 按 provider 自动选择 |
| `TAROT_MODE` | 解读模式：`professional` / `casual` | `professional` |

## License

MIT
