package main

import (
	"errors"
	"testing"
)

func TestParseCLI(t *testing.T) {
	mode, version, checkUpdate, dailyCmd, err := parseCLI([]string{"--version"})
	if err != nil || !version || checkUpdate || dailyCmd || mode != "" {
		t.Fatalf("version: mode=%q version=%v check=%v daily=%v err=%v", mode, version, checkUpdate, dailyCmd, err)
	}

	mode, version, checkUpdate, dailyCmd, err = parseCLI([]string{"--mode=pro"})
	if err != nil || version || checkUpdate || dailyCmd || mode != "pro" {
		t.Fatalf("mode only: mode=%q version=%v check=%v daily=%v err=%v", mode, version, checkUpdate, dailyCmd, err)
	}

	mode, version, checkUpdate, dailyCmd, err = parseCLI([]string{"daily", "--mode=simple"})
	if err != nil || version || checkUpdate || !dailyCmd || mode != "simple" {
		t.Fatalf("daily after: mode=%q version=%v check=%v daily=%v err=%v", mode, version, checkUpdate, dailyCmd, err)
	}

	mode, version, checkUpdate, dailyCmd, err = parseCLI([]string{"--mode=deep", "daily"})
	if err != nil || version || checkUpdate || !dailyCmd || mode != "deep" {
		t.Fatalf("daily before: mode=%q version=%v check=%v daily=%v err=%v", mode, version, checkUpdate, dailyCmd, err)
	}

	_, version, checkUpdate, dailyCmd, err = parseCLI([]string{"foo"})
	if err != nil || version || checkUpdate || dailyCmd {
		t.Fatalf("unknown positional should still launch: version=%v check=%v daily=%v err=%v", version, checkUpdate, dailyCmd, err)
	}

	_, version, checkUpdate, dailyCmd, err = parseCLI([]string{"--check-update"})
	if err != nil || version || !checkUpdate || dailyCmd {
		t.Fatalf("check-update: version=%v check=%v daily=%v err=%v", version, checkUpdate, dailyCmd, err)
	}
	_, version, checkUpdate, dailyCmd, err = parseCLI([]string{"daily", "--check-update"})
	if err != nil || version || !checkUpdate || !dailyCmd {
		t.Fatalf("both: version=%v check=%v daily=%v err=%v", version, checkUpdate, dailyCmd, err)
	}

	_, _, _, _, err = parseCLI([]string{"-h"})
	if !errors.Is(err, errFlagHelp) {
		t.Fatalf("help err %v", err)
	}
	_, _, _, _, err = parseCLI([]string{"--not-a-flag"})
	if !errors.Is(err, errFlagParse) {
		t.Fatalf("bad flag err %v", err)
	}
}
