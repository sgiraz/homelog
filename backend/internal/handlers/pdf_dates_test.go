package handlers

import "testing"

func TestParseDate(t *testing.T) {
	h := &PDFHandler{}

	cases := []struct {
		name  string
		value string
		want  string
	}{
		// Extended Italian form (newer Eon-style bills)
		{"extended lowercase", "01 aprile 2026", "2026-04-01"},
		{"extended end of month", "30 aprile 2026", "2026-04-30"},
		{"extended capitalized month", "01 Aprile 2026", "2026-04-01"},
		{"extended single-digit day", "1 dicembre 2025", "2025-12-01"},
		{"extended extra spaces", "5  marzo  2026", "2026-03-05"},
		// Numeric forms (legacy bills) must keep working
		{"numeric slash", "01/04/2026", "2026-04-01"},
		{"numeric single digits", "1/4/2026", "2026-04-01"},
		{"numeric dash", "01-04-2026", "2026-04-01"},
		{"numeric dot", "01.04.2026", "2026-04-01"},
		{"numeric two-digit year", "01/04/26", "2026-04-01"},
		// Unparseable input is returned unchanged
		{"unknown", "not a date", "not a date"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.parseDate(tc.value, ""); got != tc.want {
				t.Errorf("parseDate(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
