package agents

import "testing"

func TestMaxAgentTurnsCoversCelticCross(t *testing.T) {
	// 10 card meanings + spread + disclaimer + save + final reading ≈ 14 turns,
	// plus guard retries. The old cap of 15 stopped a normal Celtic cross.
	if maxAgentTurns < 30 {
		t.Fatalf("maxAgentTurns = %d, want at least 30", maxAgentTurns)
	}
}
