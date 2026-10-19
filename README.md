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
- 🌅 每日一牌 — 界面里 `ctrl+d`，或命令行 `tarot-agent daily`
- 🗂 选牌阵时，如果最近问过类似的问题，会提示上次的日期和牌面
- 💾 占卜记录在解读结束时写入本地，追问对话不另存
- 🔑 首次运行引导设置 API Key，自动验证连通性
- ⬆️ 发布版启动时提示有新版本；`tarot-agent update` 负责安装，界面里不自动替换

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
| 健康 / 法律确认 | `Enter` | 先说明塔罗不能代替医生或律师，回车后继续选牌阵。是非题仍可以采纳改写或按 `n` 保留原问题 |
| 关怀 | `esc` | 自伤或轻生时先停下，不抽牌。紧急情况拨打 120 或 110，也可以联系当地心理援助热线或信任的人 |
| 选择牌阵 | `1` / `2` / `3` | 单张 / 三张 / 凯尔特十字。推荐项会标出来，按键不变。最近有相似问题时，这里多一行上次的日期和牌面 |
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
| `TAROT_AGENT_NO_UPDATE` | 设为 `1` 时，启动不检查新版本。`TAROT_NO_UPDATE=1` 效果相同 | 不设置 |

历史记录在解读结束时写入一条完整记录：问题、牌阵、每张牌的位置和正逆位、模型名，以及你看到的整段解读。追问不会追加进这条记录。确认问题之后，会在最近 20 条里用中文二字重合度找一条相似问题。旧记录没有牌面，或者问题是「今日指引」，就跳过。对上了的话，选牌阵时提示上次的日期和牌，解读提示词里附上上次的牌面和可执行建议摘录。

「身体健康运势」这类泛泛的说法只在选牌阵时轻轻提一句，不额外拦截。具体的病情、用药，或起诉、合同、抚养权这类问题，会先说明塔罗不能代替医生或律师，回车后仍可占卜。自伤或轻生会停在关怀页，不抽牌。

## 命令行

深入版会多讲元素和数字；简洁版用大白话，更短。界面上显示这两个名字。配置文件仍保存 `professional` 或 `casual`，已经写过的 `pro` / `casual` 不用改。历史记录不保存模式，旧记录照常打开。

```bash
./tarot-agent --version       # 打印版本后退出，不读取配置，也不检查更新
./tarot-agent --mode=pro      # 深入版，也可写 professional 或 deep
./tarot-agent --mode=casual   # 简洁版，也可写 simple
./tarot-agent daily           # 抽一张牌，打印牌名、正逆位和本地关键词，再要一句今日指引
./tarot-agent daily --mode=simple
./tarot-agent --mode=deep daily
./tarot-agent update          # 有新版本时安装；交互终端会先确认
./tarot-agent update -y       # 不询问，直接安装。也可以写 --yes
./tarot-agent update --check  # 只比较，不安装
./tarot-agent --check-update  # 与 update --check 相同
```

`daily` 不进入界面，`ctrl+d` 仍是界面里的每日一牌。没有 API Key，或者模型请求失败时，只打印本地牌义并说明原因，退出码仍是 0。结果写入历史，问题记为「今日指引」，不参与上面的相似问题匹配。不写子命令时，启动方式和以前一样。`--version` 和 `daily` 不会在启动时检查更新。`--version` 优先于 `update`。

## 更新

发布版启动后会在后台请求 `https://api.github.com/repos/MaybeJustLikeThis/tarot-agent/releases/latest`，最多等 3 秒，不带 token。只比较 `tag_name` 和本机 `main.version` 的 major.minor.patch（可以带或不带 `v`）。当前更旧时，输入页显示两行：当前版本 → 最新 tag，以及「运行 `tarot-agent update` 更新」。按任意键，或离开输入页，这行就消失。界面里不会下载或替换程序。网络失败、限流、或 tag 无法解析时，启动过程不报错。

`dev` 或空版本是本地构建，启动时不检查。要关掉发布版的启动检查：

```bash
export TAROT_AGENT_NO_UPDATE=1
# 或
export TAROT_NO_UPDATE=1
```

这两个变量不影响 `update` 子命令。

`tarot-agent update` 会查询最新 Release。已经是最新版时打印后退出，退出码 0。有新版本时打印当前版本 → 最新 tag 和发布页链接，然后安装：

- 交互终端会先问「要现在更新到 vX.Y.Z 吗？[y/N]」。直接回车是不更新，退出码 0。
- `-y` / `--yes` 跳过这个问题。
- 没有终端而且没有 `-y` 时，只打印安装指引，不安装，退出码 3（表示有更新可用）。
- 只检查用 `update --check` 或 `--check-update`。成功退出码 0；网络或接口错误退出码 1，并打印 `检查更新失败：…`。远端 tag 无法解析时退出码 0，并写明已跳过。

安装时二选一，失败不会静默过去：

1. PATH 里有 `go` 时，执行 `go install github.com/MaybeJustLikeThis/tarot-agent/cmd/tarot-agent@<tag>`，然后打印 `which` 和 `tarot-agent --version`。`go install` 没有打进发布版本号时，`--version` 仍可能显示 `dev`，但代码已是该 tag。`go install` 失败（包括拉不到模块）时，改用下面的预编译包。
2. 没有 `go`，或 `go install` 失败时，下载当前平台的 Release 包（linux / macOS / Windows，amd64 / arm64；Windows 是 zip，其他是 tar.gz），有 `checksums.txt` 就核对 sha256，再写入 `os.Executable()` 指向的文件。目录不可写时打印 `curl`、解压和移动命令，退出码 1。Windows 不能覆盖正在运行的 exe，会把新文件写到旁边的 `.new`，并提示退出后替换。

也可以自己从 [Releases](https://github.com/MaybeJustLikeThis/tarot-agent/releases) 下载对应系统的包，对照 `checksums.txt` 后替换原来的二进制。或者：

```bash
go install github.com/MaybeJustLikeThis/tarot-agent/cmd/tarot-agent@latest
```

## License

MIT
