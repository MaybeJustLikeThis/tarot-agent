package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCard(t *testing.T) {
	opt, err := parseCLI([]string{"card", "愚者"})
	if err != nil || !opt.card || opt.daily || opt.export || opt.mode != "" {
		t.Fatalf("card: %+v err %v", opt, err)
	}
	if strings.Join(opt.cardArgs, " ") != "愚者" {
		t.Fatalf("args %q", opt.cardArgs)
	}

	opt, err = parseCLI([]string{"--mode=pro", "card", "The", "Fool"})
	if err != nil || !opt.card || opt.mode != "pro" || strings.Join(opt.cardArgs, " ") != "The Fool" {
		t.Fatalf("mode before card: %+v args %q err %v", opt, opt.cardArgs, err)
	}

	opt, err = parseCLI([]string{"--mode", "casual", "card", "0"})
	if err != nil || !opt.card || opt.mode != "casual" || strings.Join(opt.cardArgs, " ") != "0" {
		t.Fatalf("mode value then card command: %+v args %q err %v", opt, opt.cardArgs, err)
	}

	opt, err = parseCLI([]string{"--mode", "card"})
	if err != nil || opt.card || opt.daily || opt.mode != "card" {
		t.Fatalf("--mode card: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"--mode", "daily"})
	if err != nil || opt.daily || opt.mode != "daily" {
		t.Fatalf("--mode daily: %+v err %v", opt, err)
	}

	opt, err = parseCLI([]string{"card", "--list"})
	if err != nil || !opt.card || strings.Join(opt.cardArgs, " ") != "--list" {
		t.Fatalf("list: %+v args %q err %v", opt, opt.cardArgs, err)
	}

	if _, err = parseCLI([]string{"--not-a-flag"}); !errors.Is(err, errFlagParse) {
		t.Fatalf("bad flag still rejected: %v", err)
	}
	if err = run([]string{"card", "-h"}); !errors.Is(err, errFlagHelp) {
		t.Fatalf("card help: %v", err)
	}
}

func TestCardCommand(t *testing.T) {
	out, err := captureStdout(t, func() error { return run([]string{"card", "愚者"}) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"愚者", "The Fool", "正位含义", "逆位含义", "元素：风"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}

	out, err = captureStdout(t, func() error { return run([]string{"card", "Three", "of", "Wands"}) })
	if err != nil || !strings.Contains(out, "权杖三") || !strings.Contains(out, "逆位含义") {
		t.Fatalf("english: %q err %v", out, err)
	}

	out, err = captureStdout(t, func() error { return run([]string{"--mode=simple", "card", "钱币骑士"}) })
	if err != nil || !strings.Contains(out, "星币骑士") {
		t.Fatalf("alias: %q err %v", out, err)
	}

	err = run([]string{"card", "权杖"})
	if err == nil || !strings.Contains(err.Error(), "权杖Ace") || !strings.Contains(err.Error(), "没有精确匹配") {
		t.Fatalf("suit: %v", err)
	}
	err = run([]string{"card", "没有这张牌xyz"})
	if err == nil || !strings.Contains(err.Error(), "没有找到") {
		t.Fatalf("miss: %v", err)
	}

	out, err = captureStdout(t, func() error { return run([]string{"card", "--list"}) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "\n") != 78 {
		t.Fatalf("list lines %d", strings.Count(out, "\n"))
	}
	if !strings.Contains(out, "愚者") || !strings.Contains(out, "世界") || !strings.Contains(out, "星币国王") {
		t.Fatal("list missing ends of the deck")
	}
}
