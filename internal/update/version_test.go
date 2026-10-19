package update

import "testing"

func TestParseSemver(t *testing.T) {
	ok := []string{"v1.2.3", "1.2.3", "V0.6.0", "  v10.0.1  "}
	for _, raw := range ok {
		if _, parsed := ParseSemver(raw); !parsed {
			t.Errorf("expected to parse %q", raw)
		}
	}
	bad := []string{"", "dev", "v1.2", "1.2.3.4", "v1.2.3-rc1", "nightly", "vv1.2.3"}
	for _, raw := range bad {
		if _, parsed := ParseSemver(raw); parsed {
			t.Errorf("expected to reject %q", raw)
		}
	}
	got, _ := ParseSemver("v1.2.3")
	if got.Major != 1 || got.Minor != 2 || got.Patch != 3 {
		t.Fatalf("parsed %+v", got)
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.3", "1.2.3", false},
		{"1.2.3", "v1.2.3", false},
		{"v1.2.4", "1.2.3", true},
		{"v1.3.0", "1.2.9", true},
		{"v2.0.0", "1.9.9", true},
		{"v1.2.3", "v1.2.4", false},
		{"v0.6.0", "0.5.0", true},
		{"nightly", "0.5.0", false},
		{"v0.6.0", "dev", false},
		{"v0.6.0", "", false},
		{"", "0.5.0", false},
		{"v1.2.3-rc1", "1.2.0", false},
		{"v1.2", "1.0.0", false},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestStartupEnabled(t *testing.T) {
	getenv := func(vals map[string]string) func(string) string {
		return func(k string) string { return vals[k] }
	}
	if !StartupEnabled("0.5.0", getenv(nil)) {
		t.Fatal("release build should check")
	}
	if StartupEnabled("dev", getenv(nil)) || StartupEnabled("", getenv(nil)) {
		t.Fatal("dev and empty should not check")
	}
	if StartupEnabled("v0.5.0", getenv(map[string]string{"TAROT_NO_UPDATE": "1"})) {
		t.Fatal("TAROT_NO_UPDATE=1 should disable")
	}
	if StartupEnabled("v0.5.0", getenv(map[string]string{"TAROT_AGENT_NO_UPDATE": "1"})) {
		t.Fatal("TAROT_AGENT_NO_UPDATE=1 should disable")
	}
	if !StartupEnabled("v0.5.0", getenv(map[string]string{"TAROT_NO_UPDATE": "0"})) {
		t.Fatal("only the value 1 disables the check")
	}
}
