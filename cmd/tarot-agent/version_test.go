package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	with := &debug.BuildInfo{Main: debug.Module{Version: "v0.6.0"}}
	devel := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}

	cases := []struct {
		name    string
		ldflags string
		bi      *debug.BuildInfo
		want    string
	}{
		{name: "ldflags wins", ldflags: "0.6.0", bi: with, want: "0.6.0"},
		{name: "ldflags with v", ldflags: "v0.7.1", bi: with, want: "0.7.1"},
		{name: "dev uses module version", ldflags: "dev", bi: with, want: "0.6.0"},
		{name: "empty uses module version", ldflags: "", bi: with, want: "0.6.0"},
		{name: "devel build info", ldflags: "dev", bi: devel, want: "dev"},
		{name: "empty build info version", ldflags: "dev", bi: &debug.BuildInfo{}, want: "dev"},
		{name: "no build info", ldflags: "dev", bi: nil, want: "dev"},
		{name: "blank ldflags no info", ldflags: "  ", bi: nil, want: "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.ldflags, tc.bi); got != tc.want {
				t.Fatalf("resolveVersion(%q) = %q, want %q", tc.ldflags, got, tc.want)
			}
		})
	}
}
