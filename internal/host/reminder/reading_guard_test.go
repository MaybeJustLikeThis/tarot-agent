package reminder

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
)

func meaningOK(id string) agentcore.Event {
	return agentcore.Event{
		Type:    agentcore.EventToolExecEnd,
		Tool:    "get_card_meaning",
		IsError: false,
		Result:  json.RawMessage(fmt.Sprintf(`{"id":%q,"name_cn":"x"}`, id)),
	}
}

func saveOK() agentcore.Event {
	return agentcore.Event{
		Type:    agentcore.EventToolExecEnd,
		Tool:    "save_reading",
		IsError: false,
		Result:  json.RawMessage(`{"status":"saved"}`),
	}
}

func TestReadingGuard_AllowFollowUp(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.AllowFollowUp()
	decision := guard(nil, agentcore.StopInfo{})
	if !decision.Allow {
		t.Fatal("resumed follow-up should be allowed to finish")
	}
	if decision.InjectMessage != "" {
		t.Fatalf("inject = %q", decision.InjectMessage)
	}
}

func TestReadingGuard_AllConditionsMet(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.TrackEvent(meaningOK("major_00"))
	g.TrackEvent(meaningOK("major_01"))
	g.TrackEvent(saveOK())

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
	g.TrackEvent(meaningOK("major_00"))

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false when not all meanings looked up")
	}
	if !strings.Contains(decision.InjectMessage, "含义") {
		t.Errorf("expected meaning inject message, got %q", decision.InjectMessage)
	}
}

func TestReadingGuard_ReadingNotSaved(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)
	g.TrackEvent(meaningOK("major_00"))

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false when reading not saved")
	}
	if !strings.Contains(decision.InjectMessage, "保存") {
		t.Errorf("expected save inject message, got %q", decision.InjectMessage)
	}
}

func TestReadingGuard_MaxConsecutiveBlocks_Meanings(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(3)
	g.TrackEvent(meaningOK("major_00"))

	d1 := guard(nil, agentcore.StopInfo{})
	if d1.Allow || !strings.Contains(d1.InjectMessage, "含义") {
		t.Fatalf("expected meaning block 1, got allow=%v msg=%q", d1.Allow, d1.InjectMessage)
	}
	d2 := guard(nil, agentcore.StopInfo{})
	if d2.Allow || !strings.Contains(d2.InjectMessage, "含义") {
		t.Fatalf("expected meaning block 2, got allow=%v msg=%q", d2.Allow, d2.InjectMessage)
	}

	// The third consecutive meaning miss waives the meaning check, but the
	// save check still runs on its own counter.
	d3 := guard(nil, agentcore.StopInfo{})
	if d3.Allow || !strings.Contains(d3.InjectMessage, "保存") {
		t.Fatalf("expected save block after meaning waiver, got allow=%v msg=%q", d3.Allow, d3.InjectMessage)
	}
	d4 := guard(nil, agentcore.StopInfo{})
	if d4.Allow {
		t.Fatal("expected a second save block")
	}

	d5 := guard(nil, agentcore.StopInfo{})
	if !d5.Allow {
		t.Fatal("expected Allow=true after the save check hits its own limit")
	}
	if !strings.Contains(d5.InjectMessage, "保存") {
		t.Errorf("expected degraded save message, got %q", d5.InjectMessage)
	}
	if !strings.Contains(d5.InjectMessage, "含义") {
		t.Errorf("expected degraded message to mention missing meanings, got %q", d5.InjectMessage)
	}
}

func TestReadingGuard_MaxConsecutiveBlocks_Saving(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)
	g.TrackEvent(meaningOK("major_00"))

	for i := 0; i < 2; i++ {
		d := guard(nil, agentcore.StopInfo{})
		if d.Allow {
			t.Errorf("expected save block %d", i+1)
		}
	}

	d := guard(nil, agentcore.StopInfo{})
	if !d.Allow {
		t.Error("expected Allow=true after max consecutive save blocks")
	}
	if !strings.Contains(d.InjectMessage, "保存") {
		t.Errorf("expected degraded save message, got %q", d.InjectMessage)
	}
}

func TestReadingGuard_Reset(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.TrackEvent(meaningOK("major_00"))
	g.TrackEvent(meaningOK("major_01"))
	g.TrackEvent(saveOK())

	d := guard(nil, agentcore.StopInfo{})
	if !d.Allow {
		t.Fatal("expected Allow=true before reset")
	}

	g.Reset()

	d = guard(nil, agentcore.StopInfo{})
	if d.Allow {
		t.Error("expected Allow=false after reset")
	}
}

func TestReadingGuard_NoExpectedCards(t *testing.T) {
	_, guard := NewReadingGuard()

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false — reading not saved yet")
	}
	if !strings.Contains(decision.InjectMessage, "保存") {
		t.Errorf("expected save inject message, got %q", decision.InjectMessage)
	}
}

func TestReadingGuard_NonToolEventsIgnored(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)

	g.TrackEvent(agentcore.Event{Type: agentcore.EventMessageUpdate})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventAgentEnd})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventError})
	g.TrackEvent(agentcore.Event{Type: agentcore.EventToolExecStart, Tool: "get_card_meaning"})

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Error("expected Allow=false — no successful tool calls tracked")
	}
	if !strings.Contains(decision.InjectMessage, "含义") {
		t.Errorf("expected meaning block, got %q", decision.InjectMessage)
	}
}

func TestReadingGuard_BlockCountResets(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)

	guard(nil, agentcore.StopInfo{})
	guard(nil, agentcore.StopInfo{})
	if g.meaningBlocks != 2 {
		t.Errorf("expected meaningBlocks=2, got %d", g.meaningBlocks)
	}
	if g.saveBlocks != 0 {
		t.Errorf("expected saveBlocks=0 while meanings are incomplete, got %d", g.saveBlocks)
	}

	g.Reset()
	if g.meaningBlocks != 0 || g.saveBlocks != 0 || g.meaningsWaived || g.readingSaved || len(g.seen) != 0 {
		t.Errorf("expected counters cleared after reset, got meaning=%d save=%d waived=%v saved=%v seen=%d",
			g.meaningBlocks, g.saveBlocks, g.meaningsWaived, g.readingSaved, len(g.seen))
	}
}

func TestReadingGuard_FailedToolDoesNotCount(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(1)

	g.TrackEvent(agentcore.Event{
		Type: agentcore.EventToolExecStart,
		Tool: "get_card_meaning",
		Args: json.RawMessage(`{"card_id":"major_00"}`),
	})
	g.TrackEvent(agentcore.Event{
		Type:    agentcore.EventToolExecEnd,
		Tool:    "get_card_meaning",
		IsError: true,
		Result:  json.RawMessage(`"card id \"major_00\" not found"`),
	})
	g.TrackEvent(agentcore.Event{
		Type:    agentcore.EventToolExecEnd,
		Tool:    "save_reading",
		IsError: true,
		Result:  json.RawMessage(`"save reading: disk full"`),
	})

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Fatal("failed tools must not satisfy the guard")
	}
	if !strings.Contains(decision.InjectMessage, "含义") {
		t.Fatalf("expected meaning block, got %q", decision.InjectMessage)
	}
	if g.readingSaved {
		t.Fatal("failed save_reading must not mark the reading saved")
	}
	if len(g.seen) != 0 {
		t.Fatalf("failed get_card_meaning must not be counted, seen=%d", len(g.seen))
	}
}

func TestReadingGuard_DuplicateCardDoesNotCount(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	// Two successful results for the same canonical id (for example a fuzzy
	// id and the exact id both resolve to major_00 in the tool result).
	g.TrackEvent(meaningOK("major_00"))
	g.TrackEvent(meaningOK("major_00"))

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Fatal("duplicate lookups of the same card must not fill the quota")
	}
	if !strings.Contains(decision.InjectMessage, "含义") {
		t.Fatalf("expected meaning block, got %q", decision.InjectMessage)
	}
	if len(g.seen) != 1 {
		t.Fatalf("expected 1 unique card, got %d", len(g.seen))
	}
}

func TestReadingGuard_MeaningBlocksDoNotConsumeSaveBudget(t *testing.T) {
	g, guard := NewReadingGuard()
	g.SetExpectedCards(2)
	g.TrackEvent(meaningOK("major_00"))

	for i := 0; i < 2; i++ {
		d := guard(nil, agentcore.StopInfo{})
		if d.Allow || !strings.Contains(d.InjectMessage, "含义") {
			t.Fatalf("meaning block %d: allow=%v msg=%q", i+1, d.Allow, d.InjectMessage)
		}
	}

	g.TrackEvent(meaningOK("minor_wands_01"))

	decision := guard(nil, agentcore.StopInfo{})
	if decision.Allow {
		t.Fatal("after two meaning blocks, the first save check must still block")
	}
	if !strings.Contains(decision.InjectMessage, "保存") {
		t.Fatalf("expected save block, got %q", decision.InjectMessage)
	}
	if g.saveBlocks != 1 {
		t.Fatalf("expected saveBlocks=1, got %d", g.saveBlocks)
	}
}
