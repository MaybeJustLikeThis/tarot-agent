# Changelog

本文件记录 tarot-agent 的版本变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [0.3.0] - 2026-06-09

### 新增

- **Anthropic API 兼容** — 支持 DeepSeek、OpenAI、Anthropic 三种 AI 服务商，首次设置向导可选择
- `TAROT_PROVIDER` 环境变量，可切换 AI 服务商
- API Key 按 provider 格式验证（Anthropic `sk-ant-`、OpenAI/DeepSeek `sk-`）
- 已有配置无 provider 字段时自动从 BaseURL 推断迁移

### 改进

- 设置向导重构为 5 步流程：选择服务商 → API 地址 → 模型 → API Key → 解读模式
- 每个 provider 自动填充默认 URL 和 Model，回车即用
- 日志输出包含 provider 信息

## [0.2.0] - 2026-06-09

### 新增

- **历史记录浏览** — 输入界面按 `tab` 键查看最近 20 条占卜记录，支持 `↑↓` 浏览、`esc` 返回
- **每日一牌** — 按 `ctrl+d` 一键进入单牌快速模式，自动跳过问题输入和牌阵选择
- **API Key 验证** — 首次设置保存 Key 后自动验证连通性，失败时提示用户
- **Agent 超时保护** — AI 解读超过 120 秒自动中止并提示重试，防止无限等待
- **ReadingGuard 测试** — 9 个测试用例覆盖正常/不足/未保存/达到上限/重置等全场景
- **saveReadingTool 测试** — 7 个测试用例覆盖正常保存、无效 JSON、缺失字段、ID 唯一性等

### 改进

- **ReadingGuard 降级策略** — LLM 连续 3 次不响应工具调用后自动放行，附带降级提示，避免无限阻塞浪费 token
- **SpreadState 返回导航** — 选牌阵界面支持 `esc`/`backspace` 返回问题输入
- **视图代码去重** — 提取 `renderReadingView()` 共享函数，ReadingState/FollowUpState 复用同一渲染逻辑
- **Style 重构** — 删除 `lipgloss_muted()` 等包装函数和错误注释，改用 `styleMuted`/`styleSubtle`/`styleSuccess` 包级变量
- **文件权限统一** — 占卜记录目录和文件权限从 `0o755`/`0o644` 收紧为 `0o700`/`0o600`，与配置文件一致
- **输入提示更新** — 底部状态栏显示 `tab 历史 · ctrl+d 每日一牌` 快捷键提示

### 修复

- **类型安全** — `formatCardMeaning(card any)` 改为 `formatCardMeaning(card domain.Card)`，消除运行时类型断言风险

### 删除

- **死代码清理（-735 行）**
  - `internal/host/cli.go` — 未使用的 CLI 会话（TUI 已替代）
  - `internal/host/display.go` — 未使用的 CLI 显示层
  - `internal/tools/draw_cards.go` 中未注册的 `drawCardsTool` 结构体
  - 项目根目录 `assets/` — 过期的旧版数据（已被 `internal/store/assets/` 替代）
  - `cmd/tarot-agent/main.go` 中未使用的 `context.WithCancel` 代码

## [0.1.0] - 2026-06-07

### 新增

- **核心功能**
  - 78 张韦特塔罗牌完整正/逆位牌义（22 大阿卡纳 + 56 小阿卡纳）
  - 3 种牌阵：单张牌、三张牌（过去/现在/未来）、凯尔特十字（10 张）
  - 逐张翻牌动画，还原占卜仪式感
  - AI 深度个性化解读 + 多轮追问
  - 占卜记录自动保存（JSONL 格式）
  - 首次运行引导设置 API Key
  - TUI 双栏布局（牌面 + 解读），支持滚动浏览

- **塔罗专业性增强**
  - 卡牌数据充实：元素、占星、数理、画面象征、宫廷牌人格
  - 全局知识库：元素交互、牌组属性、数字含义
  - 双 Prompt 架构：专业模式（含元素/占星/数理分析）+ 轻松模式（温暖对话式）
  - `--mode` 命令行参数切换解读模式
  - 卡牌名称模糊匹配：支持中英文名、别名、ID 容错

- **技术栈**
  - Go + Bubbletea TUI 框架
  - agentcore Agent 框架（工具调用 + StopGuard）
  - DeepSeek API（兼容 OpenAI 格式）
  - Fisher-Yates 洗牌（crypto/rand）
