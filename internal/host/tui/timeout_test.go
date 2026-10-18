package tui

import (
	"strings"
	"testing"
	"time"
)

func TestAgentTimeoutDefaults(t *testing.T) {
	if got := agentTimeoutForSpread("single", 0, false); got != defaultAgentTimeout {
		t.Fatalf("single: got %s", got)
	}
	if got := agentTimeoutForSpread("three_card", 0, false); got != 120*time.Second {
		t.Fatalf("three_card: got %s", got)
	}
	if got := agentTimeoutForSpread("celtic_cross", 0, false); got != 5*time.Minute {
		t.Fatalf("celtic_cross: got %s", got)
	}
	if got := agentTimeoutForSpread("celtic_cross", 45*time.Second, true); got != 45*time.Second {
		t.Fatalf("override: got %s", got)
	}
}

func TestParseAgentTimeoutEnv(t *testing.T) {
	d, set, err := parseAgentTimeoutEnv("")
	if err != nil || set || d != 0 {
		t.Fatalf("empty: %s set=%v err=%v", d, set, err)
	}
	d, set, err = parseAgentTimeoutEnv("  ")
	if err != nil || set {
		t.Fatalf("blank: set=%v err=%v", set, err)
	}
	d, set, err = parseAgentTimeoutEnv("90")
	if err != nil || !set || d != 90*time.Second {
		t.Fatalf("seconds: %s set=%v err=%v", d, set, err)
	}
	d, set, err = parseAgentTimeoutEnv("5m")
	if err != nil || !set || d != 5*time.Minute {
		t.Fatalf("duration: %s set=%v err=%v", d, set, err)
	}
	d, set, err = parseAgentTimeoutEnv("30m")
	if err != nil || d != 30*time.Minute {
		t.Fatalf("cap edge: %s err=%v", d, err)
	}

	for _, raw := range []string{"0", "-1s", "nope", "31m", "500ms"} {
		if _, _, err := parseAgentTimeoutEnv(raw); err == nil {
			t.Errorf("%q: expected error", raw)
		}
	}
}

func TestTimeoutAndMaxTurnsMessages(t *testing.T) {
	err := agentTimeoutError(5 * time.Minute)
	if !strings.Contains(err.Error(), "5分钟") || !strings.Contains(err.Error(), "TAROT_AGENT_TIMEOUT") {
		t.Fatalf("timeout message: %s", err)
	}
	err = agentTimeoutError(90 * time.Second)
	if !strings.Contains(err.Error(), "90秒") {
		t.Fatalf("timeout seconds: %s", err)
	}
}

func TestRunRejectsBadTimeout(t *testing.T) {
	t.Setenv("TAROT_AGENT_TIMEOUT", "nope")
	err := Run(nil, nil, nil, "", "")
	if err == nil || !strings.Contains(err.Error(), "TAROT_AGENT_TIMEOUT") {
		t.Fatalf("got %v", err)
	}
}
