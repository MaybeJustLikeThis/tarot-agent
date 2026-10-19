package update

import (
	"strconv"
	"strings"
)

// Semver is a major.minor.patch triple. Pre-release and build suffixes are rejected.
type Semver struct {
	Major int
	Minor int
	Patch int
}

// ParseSemver accepts "v1.2.3" and "1.2.3". Anything else, including dev and
// empty strings, is not a release version.
func ParseSemver(raw string) (Semver, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Semver{}, false
	}
	nums := make([]int, 3)
	for i, part := range parts {
		if part == "" || strings.ContainsAny(part, "+-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return Semver{}, false
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return Semver{}, false
		}
		nums[i] = n
	}
	return Semver{Major: nums[0], Minor: nums[1], Patch: nums[2]}, true
}

// IsNewer reports whether latest is a newer release than current.
// Unparseable values are not newer, so a bad tag never nags.
func IsNewer(latest, current string) bool {
	next, okL := ParseSemver(latest)
	have, okC := ParseSemver(current)
	if !okL || !okC {
		return false
	}
	if next.Major != have.Major {
		return next.Major > have.Major
	}
	if next.Minor != have.Minor {
		return next.Minor > have.Minor
	}
	return next.Patch > have.Patch
}

// Disabled reports whether startup checks are turned off.
// TAROT_AGENT_NO_UPDATE=1 or TAROT_NO_UPDATE=1 both count. Other values do not.
func Disabled(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	return strings.TrimSpace(getenv("TAROT_AGENT_NO_UPDATE")) == "1" ||
		strings.TrimSpace(getenv("TAROT_NO_UPDATE")) == "1"
}

// StartupEnabled is false for dev/empty builds and when the env switch is on.
func StartupEnabled(version string, getenv func(string) string) bool {
	if Disabled(getenv) {
		return false
	}
	_, ok := ParseSemver(version)
	return ok
}
