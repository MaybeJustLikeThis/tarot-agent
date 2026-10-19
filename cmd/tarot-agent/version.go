package main

import (
	"runtime/debug"
	"strings"
)

// currentVersion prefers the ldflags value, then the module version from
// build info. go install @v0.6.0 leaves ldflags at dev but records v0.6.0.
func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return resolveVersion(version, nil)
	}
	return resolveVersion(version, info)
}

// resolveVersion returns the string printed by --version and used for updates.
// Release binaries print without a leading v, so v0.6.0 becomes 0.6.0.
func resolveVersion(ldflags string, bi *debug.BuildInfo) string {
	if v := normalizeVersion(ldflags); v != "" {
		return v
	}
	if bi != nil {
		if v := normalizeVersion(bi.Main.Version); v != "" {
			return v
		}
	}
	return "dev"
}

func normalizeVersion(raw string) string {
	v := strings.TrimSpace(raw)
	switch v {
	case "", "dev", "devel", "(devel)":
		return ""
	}
	v = strings.TrimPrefix(v, "v")
	switch v {
	case "", "dev", "devel", "(devel)":
		return ""
	}
	return v
}
