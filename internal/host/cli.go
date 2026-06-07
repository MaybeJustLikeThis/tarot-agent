package host

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/host/reminder"
	"github.com/voocel/tarot-agent/internal/tools"
)

// Session manages the CLI interaction loop.
type Session struct {
	agent   *agentcore.Agent
	guard   *reminder.ReadingGuard
	display *Display
	reader  *bufio.Reader
}

// NewSession creates a new CLI session.
func NewSession(agent *agentcore.Agent, guard *reminder.ReadingGuard, out io.Writer, in io.Reader) *Session {
	return &Session{
		agent:   agent,
		guard:   guard,
		display: NewDisplay(out),
		reader:  bufio.NewReader(in),
	}
}

// spreadMap maps user input to spread type IDs.
var spreadMap = map[string]string{
	"1": "single",
	"2": "three_card",
	"3": "celtic_cross",
}

// exitCommands are recognized exit inputs.
var exitCommands = map[string]bool{
	"q": true, "quit": true, "exit": true, "退出": true,
}

// Run starts the main interaction loop.
func (s *Session) Run(ctx context.Context) error {
	s.display.ShowWelcome()
	s.display.ShowDisclaimer(disclaimerText())

	for {
		// Single prompt: situation + question combined
		fmt.Fprintln(s.display.writer)
		s.display.PromptInput("说说你的情况和想问的问题（输入 q 退出）：\n  > ")
		input, err := s.readLine()
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}
		if exitCommands[strings.ToLower(input)] {
			s.display.ShowGoodbye()
			return nil
		}

		// Ask for spread type
		s.display.ShowSpreadOptions()
		s.display.PromptInput("选择牌阵（1/2/3，q 退出）：")
		spreadInput, err := s.readLine()
		if err != nil {
			return fmt.Errorf("read spread choice: %w", err)
		}
		spreadInput = strings.TrimSpace(spreadInput)

		if exitCommands[strings.ToLower(spreadInput)] {
			s.display.ShowGoodbye()
			return nil
		}

		spreadType, ok := spreadMap[spreadInput]
		if !ok {
			s.display.ShowError(fmt.Errorf("无效选择，请输入 1、2 或 3"))
			continue
		}

		// Reset guard for new reading
		s.guard.Reset()

		// Run the reading
		s.display.ShowThinking()

		prompt := buildPrompt(input, spreadType)

		if err := s.runAgent(ctx, prompt); err != nil {
			s.display.ShowError(err)
			slog.Error("agent run failed", "error", err)
		}

		// Follow-up loop: keep asking until user presses Enter or q
		for {
			fmt.Fprintln(s.display.writer)
			s.display.PromptInput("还有什么想问的？（回车开始新占卜，q 退出）\n  > ")
			followUp, err := s.readLine()
			if err != nil {
				return fmt.Errorf("read follow-up: %w", err)
			}
			followUp = strings.TrimSpace(followUp)

			if followUp == "" {
				break // back to new reading
			}
			if exitCommands[strings.ToLower(followUp)] {
				s.display.ShowGoodbye()
				return nil
			}

			if err := s.runAgent(ctx, followUp); err != nil {
				s.display.ShowError(err)
				slog.Error("follow-up failed", "error", err)
			}
		}

		// Clear messages for next reading
		s.agent.ClearMessages()
		fmt.Fprintln(s.display.writer)
		fmt.Fprintln(s.display.writer, "  ───────────────────────────────────────")
	}
}

// buildPrompt constructs the user message for the agent.
func buildPrompt(input, spreadType string) string {
	return fmt.Sprintf(
		"【用户描述】\n%s\n\n【牌阵】\n%s",
		input, spreadType,
	)
}

// runAgent sends a prompt to the agent and streams the response.
func (s *Session) runAgent(ctx context.Context, prompt string) error {
	unsub := s.agent.Subscribe(func(ev agentcore.Event) {
		s.guard.TrackEvent(ev)

		switch ev.Type {
		case agentcore.EventMessageUpdate:
			if ev.DeltaKind == "" {
				fmt.Print(ev.Delta)
			}
		case agentcore.EventToolExecStart:
			fmt.Fprintf(s.display.writer, "\n  [工具调用] %s\n", ev.Tool)
		case agentcore.EventError:
			if ev.Err != nil {
				s.display.ShowError(ev.Err)
			}
		case agentcore.EventAgentEnd:
			fmt.Fprintln(s.display.writer)
		}
	})
	defer unsub()

	if err := s.agent.Prompt(prompt); err != nil {
		return fmt.Errorf("agent prompt: %w", err)
	}
	s.agent.WaitForIdle()

	return nil
}

func (s *Session) readLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func disclaimerText() string {
	return tools.DisclaimerText()
}
