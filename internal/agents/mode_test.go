package agents

import (
	"strings"
	"testing"
)

func TestNormalizeMode(t *testing.T) {
	cases := []struct {
		raw  string
		want AgentMode
		ok   bool
	}{
		{"", ModeProfessional, true},
		{"pro", ModeProfessional, true},
		{"PRO", ModeProfessional, true},
		{" professional ", ModeProfessional, true},
		{"casual", ModeCasual, true},
		{"Casual", ModeCasual, true},
		{"deep", ModeProfessional, true},
		{" simple ", ModeCasual, true},
		{"nope", ModeProfessional, false},
		{"easy", ModeProfessional, false},
	}
	for _, tc := range cases {
		got, ok := NormalizeMode(tc.raw)
		if got != tc.want || ok != tc.ok {
			t.Errorf("NormalizeMode(%q) = %q, %v; want %q, %v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveMode_FlagOverridesAndWarns(t *testing.T) {
	mode, warn := ResolveMode("casual", "pro")
	if mode != ModeProfessional || warn != "" {
		t.Fatalf("flag pro: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("professional", "casual")
	if mode != ModeCasual || warn != "" {
		t.Fatalf("flag casual: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("casual", "")
	if mode != ModeCasual || warn != "" {
		t.Fatalf("env casual: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("", "")
	if mode != ModeProfessional || warn != "" {
		t.Fatalf("empty: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("nope", "")
	if mode != ModeProfessional || !strings.Contains(warn, "nope") || !strings.Contains(warn, "深入版") {
		t.Fatalf("invalid configured: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("casual", "later")
	if mode != ModeProfessional || !strings.Contains(warn, "later") {
		t.Fatalf("invalid flag: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("professional", "simple")
	if mode != ModeCasual || warn != "" {
		t.Fatalf("flag simple: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("casual", "deep")
	if mode != ModeProfessional || warn != "" {
		t.Fatalf("flag deep: mode=%s warn=%q", mode, warn)
	}
}

func TestModeLabel(t *testing.T) {
	if ModeProfessional.Label() != "深入版" || ModeCasual.Label() != "简洁版" {
		t.Fatalf("labels %q %q", ModeProfessional.Label(), ModeCasual.Label())
	}
}

func TestPromptsMatchToolResults(t *testing.T) {
	for name, text := range map[string]string{
		"pro":    systemPromptPro,
		"casual": systemPromptCasual,
	} {
		if strings.Contains(text, "包含 `interactions`") {
			t.Errorf("%s prompt still says the tool result contains interactions", name)
		}
		if !strings.Contains(text, `"status":"accepted"`) {
			t.Errorf("%s prompt does not describe save_reading's accepted status", name)
		}
		if !strings.Contains(text, "不要把完整解读贴进参数") {
			t.Errorf("%s prompt still tells the model to store the full reading via the tool", name)
		}
	}
	if !strings.Contains(systemPromptPro, "`element` 是字符串") {
		t.Fatal("pro prompt does not describe element as a string field")
	}
	if !strings.Contains(systemPromptPro, "不会返回 `interactions`") {
		t.Fatal("pro prompt does not say interactions are absent")
	}
}

func TestPromptsUseThreePartStructure(t *testing.T) {
	for name, text := range map[string]string{
		"pro":    systemPromptPro,
		"casual": systemPromptCasual,
	} {
		for _, heading := range []string{"### 一句话结论", "### 逐张", "### 可执行建议"} {
			if !strings.Contains(text, heading) {
				t.Errorf("%s prompt missing %s", name, heading)
			}
		}
		for _, old := range []string{"### 第一层", "### 第六层", "1500-2500"} {
			if strings.Contains(text, old) {
				t.Errorf("%s prompt still has %s", name, old)
			}
		}
		if !strings.Contains(text, "get_card_meaning") || !strings.Contains(text, "save_reading") {
			t.Errorf("%s prompt dropped the required tools", name)
		}
		if !strings.Contains(text, "1–2") {
			t.Errorf("%s prompt does not limit advice to 1-2 items", name)
		}
		if !strings.Contains(text, "牌阵信号") || !strings.Contains(text, "不要新开章节") || !strings.Contains(text, "不要硬凑") {
			t.Errorf("%s prompt missing the cite-only signal rule", name)
		}
	}
	if !strings.Contains(systemPromptPro, "元素和数字") {
		t.Fatal("pro prompt should let the model expand elements and numbers")
	}
	if !strings.Contains(systemPromptCasual, "大白话") || !strings.Contains(systemPromptCasual, "一两句") {
		t.Fatal("casual prompt should keep signal talk to a sentence or two of plain language")
	}
	if strings.Contains(systemPromptCasual, "如果有明显呼应") {
		t.Fatal("casual prompt still invites invented card links")
	}
}
