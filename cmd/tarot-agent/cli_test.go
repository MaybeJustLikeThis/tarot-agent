package main

import (
	"errors"
	"testing"
)

func TestParseCLI(t *testing.T) {
	mode, version, dailyCmd, err := parseCLI([]string{"--version"})
	if err != nil || !version || dailyCmd || mode != "" {
		t.Fatalf("version: mode=%q version=%v daily=%v err=%v", mode, version, dailyCmd, err)
	}

	mode, version, dailyCmd, err = parseCLI([]string{"--mode=pro"})
	if err != nil || version || dailyCmd || mode != "pro" {
		t.Fatalf("mode only: mode=%q version=%v daily=%v err=%v", mode, version, dailyCmd, err)
	}

	mode, version, dailyCmd, err = parseCLI([]string{"daily", "--mode=simple"})
	if err != nil || version || !dailyCmd || mode != "simple" {
		t.Fatalf("daily after: mode=%q version=%v daily=%v err=%v", mode, version, dailyCmd, err)
	}

	mode, version, dailyCmd, err = parseCLI([]string{"--mode=deep", "daily"})
	if err != nil || version || !dailyCmd || mode != "deep" {
		t.Fatalf("daily before: mode=%q version=%v daily=%v err=%v", mode, version, dailyCmd, err)
	}

	_, version, dailyCmd, err = parseCLI([]string{"foo"})
	if err != nil || version || dailyCmd {
		t.Fatalf("unknown positional should still launch: version=%v daily=%v err=%v", version, dailyCmd, err)
	}

	_, _, _, err = parseCLI([]string{"-h"})
	if !errors.Is(err, errFlagHelp) {
		t.Fatalf("help err %v", err)
	}
	_, _, _, err = parseCLI([]string{"--not-a-flag"})
	if !errors.Is(err, errFlagParse) {
		t.Fatalf("bad flag err %v", err)
	}
}
