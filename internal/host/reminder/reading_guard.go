package reminder

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/voocel/agentcore"
)

const maxConsecutiveBlocks = 3

// ReadingGuard ensures the agent completes all required steps before ending.
// Card drawing is handled by the TUI, so the guard only checks:
// 1. get_card_meaning called for all drawn cards
// 2. save_reading called
type ReadingGuard struct {
	meaningsCount atomic.Int32
	expectedCards atomic.Int32
	readingSaved  atomic.Bool
}

// NewReadingGuard creates a new ReadingGuard.
func NewReadingGuard() (*ReadingGuard, agentcore.StopGuard) {
	g := &ReadingGuard{}

	guard := func(_ context.Context, info agentcore.StopInfo) agentcore.StopDecision {
		expected := g.expectedCards.Load()

		// Check: all card meanings looked up?
		if expected > 0 {
			actual := g.meaningsCount.Load()
			if actual < expected {
				slog.Debug("reading_guard: blocking — not all meanings looked up",
					"expected", expected, "actual", actual)
				return agentcore.StopDecision{
					Allow:         false,
					InjectMessage: "你还没有查询所有牌的含义。请对每一张抽到的牌调用 get_card_meaning 查询含义，然后再生成解读。",
				}
			}
		}

		// Check: reading saved?
		if !g.readingSaved.Load() {
			slog.Debug("reading_guard: blocking — reading not saved")
			return agentcore.StopDecision{
				Allow:         false,
				InjectMessage: "你还没有保存本次占卜记录。请调用 save_reading 保存记录。",
			}
		}

		return agentcore.StopDecision{Allow: true}
	}

	return g, guard
}

// TrackEvent updates internal state based on agent events.
func (g *ReadingGuard) TrackEvent(ev agentcore.Event) {
	if ev.Type != agentcore.EventToolExecStart {
		return
	}
	switch ev.Tool {
	case "get_card_meaning":
		g.meaningsCount.Add(1)
	case "save_reading":
		g.readingSaved.Store(true)
	}
}

// SetExpectedCards sets the number of cards the agent should look up.
// Call this from the TUI after drawing cards.
func (g *ReadingGuard) SetExpectedCards(n int32) {
	g.expectedCards.Store(n)
}

// Reset resets all state for a new reading session.
func (g *ReadingGuard) Reset() {
	g.meaningsCount.Store(0)
	g.expectedCards.Store(0)
	g.readingSaved.Store(false)
}
