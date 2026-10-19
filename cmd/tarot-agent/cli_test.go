package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseCLI(t *testing.T) {
	opt, err := parseCLI([]string{"--version"})
	if err != nil || !opt.showVersion || opt.checkUpdate || opt.daily || opt.update || opt.mode != "" {
		t.Fatalf("version: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"--mode=pro"})
	if err != nil || opt.showVersion || opt.update || opt.daily || opt.mode != "pro" {
		t.Fatalf("mode only: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"daily", "--mode=simple"})
	if err != nil || opt.showVersion || opt.update || !opt.daily || opt.mode != "simple" {
		t.Fatalf("daily after: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"--mode=deep", "daily"})
	if err != nil || opt.update || !opt.daily || opt.mode != "deep" {
		t.Fatalf("daily before: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"foo"})
	if err != nil || opt.showVersion || opt.checkUpdate || opt.daily || opt.update {
		t.Fatalf("unknown positional should still launch: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"--check-update"})
	if err != nil || opt.showVersion || !opt.checkUpdate || opt.daily || opt.update {
		t.Fatalf("check-update: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"daily", "--check-update"})
	if err != nil || !opt.checkUpdate || !opt.daily || opt.update {
		t.Fatalf("both: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"update"})
	if err != nil || !opt.update || opt.checkOnly || opt.yes || opt.daily {
		t.Fatalf("update: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"update", "--check"})
	if err != nil || !opt.update || !opt.checkOnly || opt.yes {
		t.Fatalf("update --check: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"--yes", "update"})
	if err != nil || !opt.update || !opt.yes || opt.checkOnly {
		t.Fatalf("--yes update: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"update", "-y"})
	if err != nil || !opt.update || !opt.yes {
		t.Fatalf("update -y: %+v err %v", opt, err)
	}
	opt, err = parseCLI([]string{"--check", "update", "--mode=casual"})
	if err != nil || !opt.update || !opt.checkOnly || opt.mode != "casual" {
		t.Fatalf("check before update: %+v err %v", opt, err)
	}

	_, err = parseCLI([]string{"-h"})
	if !errors.Is(err, errFlagHelp) {
		t.Fatalf("help err %v", err)
	}
	_, err = parseCLI([]string{"--not-a-flag"})
	if !errors.Is(err, errFlagParse) {
		t.Fatalf("bad flag err %v", err)
	}
}

func TestVersionAndCheckDoNotHitTheNetwork(t *testing.T) {
	out, err := captureStdout(t, func() error { return run([]string{"--version", "update"}) })
	if err != nil || !strings.Contains(out, "tarot-agent dev") || strings.Contains(out, "最新版本") {
		t.Fatalf("version wins: %q err %v", out, err)
	}
	out, err = captureStdout(t, func() error { return run([]string{"--check-update"}) })
	if err != nil || !strings.Contains(out, "不做更新提示") {
		t.Fatalf("--check-update dev: %q err %v", out, err)
	}
	out, err = captureStdout(t, func() error { return run([]string{"update", "--check"}) })
	if err != nil || !strings.Contains(out, "不做更新提示") {
		t.Fatalf("update --check dev: %q err %v", out, err)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = old
	buf, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(buf), runErr
}
