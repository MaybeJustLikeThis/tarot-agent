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
	if mode != ModeProfessional || !strings.Contains(warn, "nope") || !strings.Contains(warn, "专业模式") {
		t.Fatalf("invalid configured: mode=%s warn=%q", mode, warn)
	}
	mode, warn = ResolveMode("casual", "later")
	if mode != ModeProfessional || !strings.Contains(warn, "later") {
		t.Fatalf("invalid flag: mode=%s warn=%q", mode, warn)
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
