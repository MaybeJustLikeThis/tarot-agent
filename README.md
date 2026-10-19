# Tarot Agent

[![CI](https://github.com/MaybeJustLikeThis/tarot-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/MaybeJustLikeThis/tarot-agent/actions/workflows/ci.yml)

AI 塔罗占卜 CLI 工具。通过牌面的象征语言，帮助你重新审视自己的处境。

> 塔罗是镜子，不是预言机。

## 功能

- 🃏 78 张韦特塔罗牌完整正/逆位牌义（22 大阿卡纳 + 56 小阿卡纳）
- 🔮 3 种牌阵：单张牌（快速指引）、三张牌（过去/现在/未来）、凯尔特十字（深度分析）
- ✨ 逐张翻牌动画，还原真实占卜仪式感
- 🤖 AI 深度解读 — 结合你的具体情境，不是模板复述
- 🔎 解读时按 `ctrl+o` 查看本地牌库里的牌义、正逆位关键词和整体信号，不另调模型
- 💬 解读后可继续追问，深入探讨
- 📖 历史记录浏览 — 保存并回顾完整解读（问题、牌阵、牌面、模型、正文）
- 🌅 每日一牌 — 一键获取今日指引
- 💾 占卜记录在解读结束时写入本地，追问对话不另存
- 🔑 首次运行引导设置 API Key，自动验证连通性

## 安装

### 预编译二进制

[Releases](https://github.com/MaybeJustLikeThis/tarot-agent/releases) 提供 linux、macOS、Windows 的 amd64 和 arm64 包。Linux / macOS 是 tar.gz，Windows 是 zip。

```bash
VERSION=v0.5.0
curl -sSL -o tarot-agent.tar.gz \
  "https://github.com/MaybeJustLikeThis/tarot-agent/releases/download/${VERSION}/tarot-agent_${VERSION#v}_linux_amd64.tar.gz"
tar -xzf tarot-agent.tar.gz
./tarot-agent --version
```

macOS 把文件名里的 `linux` 换成 `darwin`。架构是 `amd64` 或 `arm64`。

用 Go 安装时，`v0.5.0` 及以后才带现在的 module 路径：

```bash
go install github.com/MaybeJustLikeThis/tarot-agent/cmd/tarot-agent@latest
```

### 从源码

```bash
# 前置条件：Go 1.25.5（与 go.mod 一致，https://go.dev/doc/install）

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

按键行为随界面状态变化。下面的是实际会生效的操作，不是全局热键。

| 状态 | 按键 | 功能 |
|------|------|------|
| 输入问题 | `Enter` | 提交问题。整段输入是 `q` / `quit` / `exit` / `退出` 时退出 |
| 输入问题 | `tab` | 输入框为空时打开历史 |
| 输入问题 | `ctrl+d` | 每日一牌 |
| 问题建议 | `Enter` | 采纳开放式改写，进入选牌阵。只在是非题、宿命式或时间预测类问题出现 |
| 问题建议 | `n` | 保留原问题，进入选牌阵 |
| 问题建议 | `esc` | 返回修改 |
| 选择牌阵 | `1` / `2` / `3` | 单张 / 三张 / 凯尔特十字。推荐项会标出来，按键不变 |
| 选择牌阵 | `q` | 立刻退出 |
| 选择牌阵 | `esc` | 返回输入问题 |
| 解读中 | `↑↓` / `jk`、`PgUp` / `PgDn` | 滚动解读 |
| 解读中 | `ctrl+c` | 中止本次解读并退出 |
| 解读中 / 追问 / 对话生成中 | `ctrl+o` | 打开当前牌阵各张牌的详情。牌名、关键词、元素、数字、占星、意象、宫廷牌角色和整体信号都来自本地牌库 |
| 牌义 | `←` `→` / `h` `l` | 上一张 / 下一张 |
| 牌义 | `esc` | 返回解读。滚动位置和正在生成的内容都还在 |
| 追问 | `Enter` | 发送追问；输入为空则开始新的占卜 |
| 追问 | `tab` | 在解读区和对话区之间切换滚动焦点 |
| 追问 | 输入 `q` 后回车 | 退出 |
| 对话生成中 | `tab` | 切换解读 / 对话焦点 |
| 对话生成中 | `ctrl+c` | 中止当前回答并退出 |
| 历史 | `↑↓` / `jk` | 选择一条记录 |
| 历史 | `PgUp` / `PgDn` | 滚动详情 |
| 历史 | `esc` | 返回输入问题 |

在输入问题或追问里，单独按 `q` 只会打进输入框，不会退出。

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

# 还没有配置文件时，只导出 DeepSeek key 也可以启动
export DEEPSEEK_API_KEY="sk-xxx"
```

## 配置

API Key 写在 `~/.tarot-agent/config.json`，文件权限是 `0600`，目录是 `0700`。日志写到 `~/.tarot-agent/tarot.log`。打不开日志文件时直接丢弃，不再写到 stderr，避免打乱界面。文件大约超过 1MB 时轮转，只保留上一份 `tarot.log.1`。

合并顺序：内置默认值 < 配置文件 < 环境变量。`--mode` 再覆盖模式。

| 来源 | 说明 |
|------|------|
| `TAROT_API_KEY` | 只要设置了，就覆盖配置文件和 `DEEPSEEK_API_KEY` |
| `DEEPSEEK_API_KEY` | 没有配置文件时可以单独用来启动。配置里如果已经保存了其他服务商的 key，它不会覆盖那把 key；当前服务商是 `deepseek` 时，会替换已保存的 DeepSeek key |
| `--mode` | `pro`、`professional`、`deep` 是深入版，`casual`、`simple` 是简洁版。配置文件里的旧值照常读取。非法值会警告并回退到深入版，不会因此启动失败 |

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `TAROT_PROVIDER` | AI 服务商：`deepseek` / `openai` / `anthropic` | `deepseek` |
| `TAROT_API_KEY` | API Key，优先级最高 | 无（首次引导设置，或只用 `DEEPSEEK_API_KEY`） |
| `DEEPSEEK_API_KEY` | 见上一节。不能替换其他服务商已保存的 key | 无 |
| `TAROT_BASE_URL` | API 地址 | 按 provider 自动选择 |
| `TAROT_MODEL` | 模型名 | 按 provider 自动选择 |
| `TAROT_MODE` | 解读模式，与 `--mode` 互通：`professional`、`pro`、`deep`、`casual`、`simple` | `professional` |
| `TAROT_LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `TAROT_AGENT_TIMEOUT` | 单次解读的墙钟上限。可用 `5m`、`90s` 或纯秒数。未设置时单张牌和三张牌是 120 秒，凯尔特十字是 5 分钟。至少 1 秒，最多 30 分钟；非法值会在进入界面前报错 | 按牌阵 |

历史记录在解读结束时写入一条完整记录：问题、牌阵、每张牌的位置和正逆位、模型名，以及你看到的整段解读。追问不会追加进这条记录。

## 命令行

深入版会多讲元素和数字；简洁版用大白话，更短。界面上显示这两个名字。配置文件仍保存 `professional` 或 `casual`，已经写过的 `pro` / `casual` 不用改。历史记录不保存模式，旧记录照常打开。

```bash
./tarot-agent --version       # 打印版本后退出，不读取配置
./tarot-agent --mode=pro      # 深入版，也可写 professional 或 deep
./tarot-agent --mode=casual   # 简洁版，也可写 simple
```

## License

MIT
