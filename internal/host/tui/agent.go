package tui

import (
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
)

// agentBridge manages the agent lifecycle: subscribe, start, cleanup.
type agentBridge struct {
	agent  *agentcore.Agent
	guard  *reminder.ReadingGuard
	unsub  func()
	eventCh chan tea.Msg
}

func newAgentBridge(agent *agentcore.Agent, guard *reminder.ReadingGuard) *agentBridge {
	return &agentBridge{
		agent:   agent,
		guard:   guard,
		eventCh: make(chan tea.Msg, 128),
	}
}

// setup subscribes to agent events and starts the agent in background.
func (b *agentBridge) setup(prompt string) {
	b.cleanup()

	b.unsub = b.agent.Subscribe(func(ev agentcore.Event) {
		b.guard.TrackEvent(ev)
		switch ev.Type {
		case agentcore.EventMessageUpdate:
			if ev.DeltaKind == "" {
				b.eventCh <- agentDeltaMsg{text: ev.Delta}
			}
		case agentcore.EventToolExecStart:
			b.eventCh <- agentToolMsg{name: ev.Tool}
		case agentcore.EventError:
			if ev.Err != nil {
				b.eventCh <- agentErrMsg{err: ev.Err}
			}
		case agentcore.EventAgentEnd:
			b.eventCh <- agentEndMsg{}
		}
	})

	go func() {
		if err := b.agent.Prompt(prompt); err != nil {
			b.eventCh <- agentErrMsg{err: err}
			return
		}
		b.agent.WaitForIdle()
	}()
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
