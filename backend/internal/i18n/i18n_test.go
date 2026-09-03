package i18n

import (
	"strings"
	"testing"

	"github.com/sgiraz/homelog/internal/models"
)

func TestT_FillsPlaceholdersPerLanguage(t *testing.T) {
	it := T("it", "bill.expense.description", "type", "Luce", "month", "Mar", "year", "2025")
	if it != "Bolletta Luce - Mar 2025" {
		t.Errorf("IT = %q", it)
	}
	en := T("en", "bill.expense.description", "type", "Electricity", "month", "Mar", "year", "2025")
	if en != "Electricity bill - Mar 2025" {
		t.Errorf("EN = %q", en)
	}
}

func TestT_UnsupportedLanguageFallsBackToDefault(t *testing.T) {
	// A regional tag resolves to its base language; anything unsupported goes
	// through models.NormalizeLanguage and lands on the default.
	if got := T("en-GB", "utility.type.water"); got != "Water" {
		t.Errorf("en-GB = %q, want %q", got, "Water")
	}
	want := T(models.DefaultLanguage, "utility.type.water")
	if got := T("de", "utility.type.water"); got != want {
		t.Errorf("de = %q, want %q", got, want)
	}
}

func TestT_UnknownIDReturnsTheID(t *testing.T) {
	// Surfacing the id beats rendering empty text: a missing message is a bug
	// and must be visible.
	if got := T("it", "nope.not.here"); got != "nope.not.here" {
		t.Errorf("unknown id = %q", got)
	}
}

func TestUtilityType_FallsBackToRawType(t *testing.T) {
	if got := UtilityType("it", "electricity"); got != "Luce" {
		t.Errorf("electricity IT = %q", got)
	}
	if got := UtilityType("it", "something_new"); got != "something_new" {
		t.Errorf("unknown type = %q, want the raw type", got)
	}
}

func TestMonthShort_RangeAndLanguage(t *testing.T) {
	if got := MonthShort("it", 5); got != "Mag" {
		t.Errorf("May IT = %q", got)
	}
	if got := MonthShort("en", 5); got != "May" {
		t.Errorf("May EN = %q", got)
	}
	for _, m := range []int{0, 13} {
		if got := MonthShort("it", m); got != "" {
			t.Errorf("month %d = %q, want \"\"", m, got)
		}
	}
}

func TestFormatAmount_DecimalSeparatorFollowsLanguage(t *testing.T) {
	if got := FormatAmount("it", 12.5); got != "12,50" {
		t.Errorf("IT = %q", got)
	}
	if got := FormatAmount("en", 12.5); got != "12.50" {
		t.Errorf("EN = %q", got)
	}
}

// Every message must exist in every supported language: a gap would silently
// serve English to a user who chose Italian.
func TestEveryMessageCoversEverySupportedLanguage(t *testing.T) {
	for id, byLang := range messages {
		for lang := range models.SupportedLanguages {
			text, ok := byLang[lang]
			if !ok {
				t.Errorf("message %q has no %q text", id, lang)
				continue
			}
			if strings.TrimSpace(text) == "" {
				t.Errorf("message %q is empty in %q", id, lang)
			}
		}
	}
}

// A translation that drops a placeholder would render a broken sentence.
func TestPlaceholdersMatchAcrossLanguages(t *testing.T) {
	for id, byLang := range messages {
		reference := placeholders(byLang["en"])
		for lang, text := range byLang {
			if got := placeholders(text); !equalSets(got, reference) {
				t.Errorf("message %q: %q has placeholders %v, EN has %v", id, lang, got, reference)
			}
		}
	}
}

func placeholders(text string) map[string]bool {
	found := map[string]bool{}
	for {
		open := strings.Index(text, "{")
		if open < 0 {
			return found
		}
		close := strings.Index(text[open:], "}")
		if close < 0 {
			return found
		}
		found[text[open+1:open+close]] = true
		text = text[open+close:]
	}
}

func equalSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
