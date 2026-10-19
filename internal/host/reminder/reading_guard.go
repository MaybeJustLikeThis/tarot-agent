package reminder

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/voocel/agentcore"
)

const maxConsecutiveBlocks = 3

// ReadingGuard ensures the agent completes all required steps before ending.
// Card drawing is handled by the TUI, so the guard only checks:
//  1. get_card_meaning succeeded once per distinct card id
//  2. save_reading succeeded
//
// Tool starts and failed executions do not count. Meaning misses and missing
// saves keep separate consecutive-block counters. Each counter gives up after
// maxConsecutiveBlocks so a model that never makes progress cannot loop.
type ReadingGuard struct {
	mu sync.Mutex

	seen           map[string]struct{}
	expected       int
	meaningBlocks  int
	saveBlocks     int
	meaningsWaived bool
	readingSaved   bool
}

// NewReadingGuard creates a new ReadingGuard.
func NewReadingGuard() (*ReadingGuard, agentcore.StopGuard) {
	g := &ReadingGuard{seen: make(map[string]struct{})}

	guard := func(_ context.Context, _ agentcore.StopInfo) agentcore.StopDecision {
		return g.decide()
	}
	return g, guard
}

func (g *ReadingGuard) decide() agentcore.StopDecision {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.expected > 0 && len(g.seen) < g.expected && !g.meaningsWaived {
		g.meaningBlocks++
		if g.meaningBlocks >= maxConsecutiveBlocks {
			g.meaningsWaived = true
			slog.Debug("reading_guard: waiving meaning check after consecutive blocks",
				"blocks", g.meaningBlocks, "expected", g.expected, "actual", len(g.seen))
		} else {
			slog.Debug("reading_guard: blocking — not all meanings looked up",
				"expected", g.expected, "actual", len(g.seen), "blocks", g.meaningBlocks)
			return agentcore.StopDecision{
				Allow:         false,
				InjectMessage: "你还没有查询所有牌的含义。请对每一张抽到的牌调用 get_card_meaning 查询含义，然后再生成解读。",
			}
		}
	}

	if !g.readingSaved {
		g.saveBlocks++
		if g.saveBlocks >= maxConsecutiveBlocks {
			slog.Debug("reading_guard: allowing unsaved reading after consecutive blocks",
				"blocks", g.saveBlocks)
			return agentcore.StopDecision{
				Allow:         true,
				InjectMessage: g.degradedSaveMessage(),
			}
		}
		slog.Debug("reading_guard: blocking — reading not saved", "blocks", g.saveBlocks)
		return agentcore.StopDecision{
			Allow:         false,
			InjectMessage: "你还没有保存本次占卜记录。请调用 save_reading 保存记录。",
		}
	}

	if g.meaningsWaived && g.expected > 0 && len(g.seen) < g.expected {
		return agentcore.StopDecision{
			Allow:         true,
			InjectMessage: "注意：部分牌的含义可能未被完整查询，解读中如有不完整之处请理解。",
		}
	}
	return agentcore.StopDecision{Allow: true}
}

func (g *ReadingGuard) degradedSaveMessage() string {
	if g.meaningsWaived && g.expected > 0 && len(g.seen) < g.expected {
		return "注意：部分牌的含义可能未被完整查询，且本次占卜记录可能未被保存。"
	}
	return "注意：本次占卜记录可能未被保存。"
}

// TrackEvent updates internal state from a successful tool completion.
// get_card_meaning counts only when the result JSON contains a new card id.
func (g *ReadingGuard) TrackEvent(ev agentcore.Event) {
	if ev.Type != agentcore.EventToolExecEnd || ev.IsError {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	switch ev.Tool {
	case "get_card_meaning":
		id, ok := cardIDFromMeaningResult(ev.Result)
		if !ok {
			return
		}
		if _, exists := g.seen[id]; exists {
			return
		}
		g.seen[id] = struct{}{}
		// A new card breaks the consecutive-miss streak.
		g.meaningBlocks = 0
	case "save_reading":
		g.readingSaved = true
		g.saveBlocks = 0
	}
}

func cardIDFromMeaningResult(result json.RawMessage) (string, bool) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", false
	}
	id := strings.TrimSpace(payload.ID)
	if id == "" {
		return "", false
	}
	return id, true
}

// SetExpectedCards sets the number of distinct cards the agent should look up.
// Call this from the TUI after drawing cards.
func (g *ReadingGuard) SetExpectedCards(n int32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expected = int(n)
}

// AllowFollowUp marks the new-reading checklist done.
// A chat resumed from history should not be sent back to look up every card
// and call save_reading before it can answer.
func (g *ReadingGuard) AllowFollowUp() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seen = make(map[string]struct{})
	g.expected = 0
	g.meaningBlocks = 0
	g.saveBlocks = 0
	g.meaningsWaived = false
	g.readingSaved = true
}

// Reset resets all state for a new reading session.
func (g *ReadingGuard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seen = make(map[string]struct{})
	g.expected = 0
	g.meaningBlocks = 0
	g.saveBlocks = 0
	g.meaningsWaived = false
	g.readingSaved = false
}
