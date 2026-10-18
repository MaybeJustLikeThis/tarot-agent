package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultAgentTimeout is the wall clock for single-card and three-card readings.
	defaultAgentTimeout = 120 * time.Second
	// celticCrossTimeout is the default for a 10-card reading, which does not fit in 120s.
	celticCrossTimeout = 5 * time.Minute
	minAgentTimeout    = time.Second
	maxAgentTimeout    = 30 * time.Minute
)

// parseAgentTimeoutEnv reads TAROT_AGENT_TIMEOUT.
// An empty value leaves the spread default in place.
// Accepted forms: a Go duration (5m, 90s) or an integer number of seconds.
func parseAgentTimeoutEnv(raw string) (time.Duration, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		secs, nerr := strconv.Atoi(raw)
		if nerr != nil {
			return 0, false, fmt.Errorf("TAROT_AGENT_TIMEOUT=%q 无法解析，请使用时长（如 5m、90s）或秒数", raw)
		}
		d = time.Duration(secs) * time.Second
	}
	if err := validateAgentTimeout(d); err != nil {
		return 0, false, err
	}
	return d, true, nil
}

func validateAgentTimeout(d time.Duration) error {
	if d < minAgentTimeout {
		return fmt.Errorf("TAROT_AGENT_TIMEOUT 必须至少为 %s", formatTimeout(minAgentTimeout))
	}
	if d > maxAgentTimeout {
		return fmt.Errorf("TAROT_AGENT_TIMEOUT 不能超过 %s", formatTimeout(maxAgentTimeout))
	}
	return nil
}

// agentTimeoutForSpread returns the wall-clock budget for one agent run.
// A parsed override applies to every spread; otherwise only Celtic cross is longer than 120s.
func agentTimeoutForSpread(spread string, override time.Duration, overrideSet bool) time.Duration {
	if overrideSet {
		return override
	}
	if spread == "celtic_cross" {
		return celticCrossTimeout
	}
	return defaultAgentTimeout
}

func formatTimeout(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%d分钟", int(d/time.Minute))
	}
	secs := int(d.Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("%d秒", secs)
}
