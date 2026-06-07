package host

import (
	"fmt"
	"io"
	"strings"
)

// Display handles formatted output to the user.
type Display struct {
	writer io.Writer
}

// NewDisplay creates a Display that writes to the given writer.
func NewDisplay(w io.Writer) *Display {
	return &Display{writer: w}
}

// ShowWelcome displays the welcome banner.
func (d *Display) ShowWelcome() {
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  ╔═══════════════════════════════════════╗")
	fmt.Fprintln(d.writer, "  ║          星语 Tarot Agent            ║")
	fmt.Fprintln(d.writer, "  ║     你的温暖塔罗解读伙伴              ║")
	fmt.Fprintln(d.writer, "  ╚═══════════════════════════════════════╝")
	fmt.Fprintln(d.writer)
}

// ShowDisclaimer displays the disclaimer text.
func (d *Display) ShowDisclaimer(text string) {
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  ───────────────────────────────────────")
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(d.writer, "  %s\n", line)
	}
	fmt.Fprintln(d.writer, "  ───────────────────────────────────────")
	fmt.Fprintln(d.writer)
}

// ShowSpreadOptions displays available spread choices.
func (d *Display) ShowSpreadOptions() {
	fmt.Fprintln(d.writer, "  请选择牌阵：")
	fmt.Fprintln(d.writer, "    1. 单张牌   — 快速指引")
	fmt.Fprintln(d.writer, "    2. 三张牌   — 过去/现在/未来")
	fmt.Fprintln(d.writer, "    3. 凯尔特十字 — 深度全面分析")
	fmt.Fprintln(d.writer)
}

// ShowThinking displays a thinking indicator.
func (d *Display) ShowThinking() {
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  正在为你抽牌和解读...")
	fmt.Fprintln(d.writer)
}

// ShowReading displays the reading result.
func (d *Display) ShowReading(text string) {
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  ═══════════════════════════════════════")
	fmt.Fprintln(d.writer)
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(d.writer, "  %s\n", line)
	}
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  ═══════════════════════════════════════")
	fmt.Fprintln(d.writer)
}

// ShowError displays an error message.
func (d *Display) ShowError(err error) {
	fmt.Fprintf(d.writer, "  [错误] %v\n", err)
}

// ShowGoodbye displays the farewell message.
func (d *Display) ShowGoodbye() {
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  感谢你的参与，愿你找到内心的指引。再见！")
	fmt.Fprintln(d.writer)
}

// PromptInput displays a prompt and returns what the user typed.
func (d *Display) PromptInput(prompt string) {
	fmt.Fprintf(d.writer, "  %s", prompt)
}
