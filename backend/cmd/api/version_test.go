package main

import "testing"

// The update check has already failed silently once: a build with no version
// reached parseCalVer, which rejected it, and every instance then reported
// "you are up to date" forever. These are pure functions, so the cost of
// pinning that behaviour down is close to nothing.

func TestNormalizeVersionTag(t *testing.T) {
	cases := map[string]string{
		"2026.10.1":            "2026.10.1",
		"v2026.10.1":           "2026.10.1",
		"V2026.10.1":           "2026.10.1",
		"refs/tags/v2026.10.1": "2026.10.1",
		"  v2026.10.1\n":       "2026.10.1",
		"":                     "",
	}
	for in, want := range cases {
		if got := normalizeVersionTag(in); got != want {
			t.Errorf("normalizeVersionTag(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseCalVer(t *testing.T) {
	ok := []struct {
		in   string
		want [4]int
	}{
		{"2026.10.1", [4]int{2026, 10, 1, 0}},
		{"2026.10.1.2", [4]int{2026, 10, 1, 2}},
		{"v2026.1.0", [4]int{2026, 1, 0, 0}},
	}
	for _, tc := range ok {
		got, valid := parseCalVer(tc.in)
		if !valid {
			t.Errorf("parseCalVer(%q) rejected a valid version", tc.in)
			continue
		}
		if got != tc.want {
			t.Errorf("parseCalVer(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}

	// An empty version is the case that broke: the binary built without
	// -X main.appVersion must not parse, so the check can report that it
	// could not conclude instead of claiming the instance is current.
	bad := []string{"", "dev", "2026.10", "2026.10.1.2.3", "2026.ten.1", "2026..1", "latest"}
	for _, in := range bad {
		if _, valid := parseCalVer(in); valid {
			t.Errorf("parseCalVer(%q) accepted an unusable version", in)
		}
	}
}

func TestIsVersionNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"2026.10.2", "2026.10.1", true},
		{"2026.11.0", "2026.10.9", true},
		{"2027.1.0", "2026.12.31", true},
		{"2026.10.1.1", "2026.10.1", true},
		{"2026.10.1", "2026.10.1", false},
		{"2026.10.1", "2026.10.2", false},
		{"v2026.10.2", "2026.10.1", true},
		// Neither side parses: never claim an update exists on a guess.
		{"", "2026.10.1", false},
		{"2026.10.1", "", false},
		{"2026.10.1", "dev", false},
		{"garbage", "garbage", false},
	}
	for _, tc := range cases {
		if got := isVersionNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("isVersionNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}
