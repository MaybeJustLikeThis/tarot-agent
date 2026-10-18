package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/agentcore"
	"github.com/voocel/tarot-agent/internal/host/reminder"
)

// Agent event messages sent to the TUI event loop.
type (
	agentDeltaMsg struct{ text string }
	agentToolMsg  struct{ name string }
	agentEndMsg   struct{}
	agentErrMsg   struct{ err error }
	agentRetryMsg struct{ note string }
)

// agentBridge manages the agent lifecycle: subscribe, start, cleanup.
type agentBridge struct {
	agent   *agentcore.Agent
	guard   *reminder.ReadingGuard
	unsub   func()
	eventCh chan tea.Msg
	aborted bool
}

func newAgentBridge(agent *agentcore.Agent, guard *reminder.ReadingGuard) *agentBridge {
	return &agentBridge{
		agent:   agent,
		guard:   guard,
		eventCh: make(chan tea.Msg, 128),
	}
}

// setup subscribes to agent events and starts the agent in background.
// timeout is the wall-clock budget for this run; zero falls back to the small-spread default.
func (b *agentBridge) setup(prompt string, timeout time.Duration) {
	b.cleanup()
	b.aborted = false
	if timeout <= 0 {
		timeout = defaultAgentTimeout
	}

	b.unsub = b.agent.Subscribe(func(ev agentcore.Event) {
		b.guard.TrackEvent(ev)
		switch ev.Type {
		case agentcore.EventMessageUpdate:
			if ev.DeltaKind == "" {
				b.eventCh <- agentDeltaMsg{text: ev.Delta}
			}
		case agentcore.EventToolExecStart:
			b.eventCh <- agentToolMsg{name: ev.Tool}
		case agentcore.EventRetry:
			b.eventCh <- agentRetryMsg{note: formatRetryNote(ev.RetryInfo)}
		case agentcore.EventError:
			if ev.Err != nil {
				b.eventCh <- agentErrMsg{err: presentAgentError(ev.Err)}
			}
		case agentcore.EventAgentEnd:
			b.eventCh <- agentEndMsg{}
		}
	})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := b.agent.Prompt(prompt); err != nil {
				b.eventCh <- agentErrMsg{err: presentAgentError(err)}
				return
			}
			b.agent.WaitForIdle()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			b.agent.Abort()
			b.eventCh <- agentErrMsg{err: agentTimeoutError(timeout)}
		}
	}()
}

// abort cancels an in-flight run. Safe when nothing is running.
func (b *agentBridge) abort() {
	if b == nil {
		return
	}
	b.aborted = true
	if b.agent != nil {
		b.agent.Abort()
	}
}

func presentAgentError(err error) error {
	if err == nil {
		return nil
	}
	var maxTurns *agentcore.MaxTurnsError
	if errors.As(err, &maxTurns) {
		if maxTurns.Limit <= 0 {
			return fmt.Errorf("已达到轮数上限，本次解读已停止")
		}
		return fmt.Errorf("已达到轮数上限（%d 轮），本次解读已停止", maxTurns.Limit)
	}
	return err
}

func formatRetryNote(info *agentcore.RetryInfo) string {
	if info == nil || info.MaxRetries <= 0 {
		return "模型请求失败，正在重试…"
	}
	return fmt.Sprintf("模型请求失败，正在重试（%d/%d）…", info.Attempt, info.MaxRetries)
}

func agentTimeoutError(d time.Duration) error {
	return fmt.Errorf("解读超时（%s），请重试。可设置 TAROT_AGENT_TIMEOUT 放宽上限", formatTimeout(d))
}

// nextEvent returns a tea.Cmd that reads the next event from the channel.
func (b *agentBridge) nextEvent() tea.Cmd {
	return func() tea.Msg {
		return <-b.eventCh
	}
}

// cleanup unsubscribes and drains the event channel.
func (b *agentBridge) cleanup() {
	if b.unsub != nil {
		b.unsub()
		b.unsub = nil
	}
	for {
		select {
		case <-b.eventCh:
		default:
			return
		}
	}
}

// clearMessages clears the agent's conversation history.
func (b *agentBridge) clearMessages() {
	b.agent.ClearMessages()
}
