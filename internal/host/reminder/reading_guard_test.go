package reminder

import (
	"testing"

	"github.com/voocel/agentcore"
)

func TestReadingGuard_AllConditionsMet(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "save_reading"})

	decision := guard(nil, agentcore.StopInfo{})
	if !decision.Allow {
		t.Error("expected Allow=true when all meanings looked up and reading saved")
	}
	if decision.InjectMessage != "" {
		t.Errorf("expected no inject message, got %q", decision.InjectMessage)
	}
}

func TestReadingGuard_MeaningsNotComplete(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(3)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	// Only 1 of 3 meanings looked up

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false when not all meanings looked up")
	}
	if decision.InjectMessage == "" {
		t.Error("expected inject message for missing meanings")
	}
}

func TestReadingGuard_ReadingNotSaved(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	// All meanings done but reading not saved

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false when reading not saved")
	}
	if decision.InjectMessage == "" {
		t.Error("expected inject message for unsaved reading")
	}
}

func TestReadingGuard_MaxConsecutiveBlocks_Meanings(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(3)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	// Only 1 of 3 — will block

	// Block 1
	d1 := guard(nil, agentcore.StopInfo{})
	if d1.Allow {
		t.Error("expected block 1")
	}

	// Block 2
	d2 := guard(nil, agentcore.StopInfo{})
	if d2.Allow {
		t.Error("expected block 2")
	}

	// Block 3 — should allow with degraded message
	d3 := guard(nil, agentcore.StopInfo{})
	if !d3.Allow {
		t.Error("expected Allow=true after max consecutive blocks")
	}
	if d3.InjectMessage == "" {
		t.Error("expected degraded inject message")
	}
}

func TestReadingGuard_MaxConsecutiveBlocks_Saving(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	// Meanings done, but not saved — will block

	for i := 0; i < 2; i++ {
		d := guard(nil, agentcore.StopInfo{})
		if d.Allow {
			t.Errorf("expected block %d", i+1)
		}
	}

	d := guard(nil, agentcore.StopInfo{})
	if !d.Allow {
		t.Error("expected Allow=true after max consecutive blocks for save")
	}
}

func TestReadingGuard_Reset(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "save_reading"})

	// Verify it passes
	d := guard(nil, agentcore.StopInfo{})
	if !d.Allow {
		t.Fatal("expected Allow=true before reset")
	}

	// Reset and verify state is cleared
	g.Reset()

	d = guard(nil, agentcore.StopInfo{})
	if d.Allow {
		t.Error("expected Allow=false after reset (no expected cards means no check)")
	}
}

func TestReadingGuard_NoExpectedCards(t *testing.T) {
	_, guard := NewReadingGuard()
	// No expected cards set — meanings check is skipped

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false — reading not saved yet")
	}
}

func TestReadingGuard_NonToolEventsIgnored(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)

	// Send non-tool events — should not affect counts
	g.TrackEvent(agentcore.Event{Type: agentcore.EventMessageUpdate})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventAgentEnd})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventError})

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false — no tool calls tracked")
	}
}

func TestReadingGuard_BlockCountResets(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)

	// Block twice
	guard(nil, agentcore.StopInfo{})
	guard(nil, agentcore.StopInfo{})
	if g.blockCount.Load() != 2 {
		t.Errorf("expected blockCount=2, got %d", g.blockCount.Load())
	}

	// Reset should clear blockCount
	g.Reset()
	if g.blockCount.Load() != 0 {
		t.Errorf("expected blockCount=0 after reset, got %d", g.blockCount.Load())
	}
}
