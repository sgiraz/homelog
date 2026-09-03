package apierr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// An error code with no entry in a locale file renders as the raw key path in
// the UI ("errors.bill_not_found"), which is worse than the English message it
// replaced. This test is the contract between the Go call sites and the
// frontend catalogue: it fails the build instead of shipping that.

var (
	callSite = regexp.MustCompile(`apierr\.Fail(?:With)?\(\s*c\s*,\s*[\w.]+\s*,\s*"([a-z0-9_]+)"`)
	rawCode  = regexp.MustCompile(`"error_code":\s*"([a-z0-9_]+)"`)
)

func codesUsedInHandlers(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "handlers", "*.go"))
	if err != nil {
		t.Fatalf("glob handlers: %v", err)
	}
	codes := map[string]bool{
		CodeServerError:    true,
		CodeInvalidRequest: true,
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range callSite.FindAllSubmatch(src, -1) {
			codes[string(m[1])] = true
		}
		for _, m := range rawCode.FindAllSubmatch(src, -1) {
			codes[string(m[1])] = true
		}
	}
	if len(codes) < 50 {
		t.Fatalf("found only %d codes — the scan is probably broken", len(codes))
	}
	return codes
}

// translatedLanguages lists the locales that ship an errors.json. It is read
// from disk rather than hardcoded: a language added to locales/ must be checked
// too, otherwise a de/ directory with no errors.json would pass silently and
// German users would read English errors.
func translatedLanguages(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("frontend locales not available (%v)", err)
	}
	var langs []string
	for _, e := range entries {
		if e.IsDir() {
			langs = append(langs, e.Name())
		}
	}
	sort.Strings(langs)
	if len(langs) == 0 {
		t.Fatal("no locale directories found — the path is probably wrong")
	}
	return langs
}

func loadLocale(t *testing.T, lang string) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", lang, "errors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s has no errors.json: every supported language needs one, "+
			"or its users read English errors (%v)", lang, err)
		return map[string]string{}
	}
	var messages map[string]string
	if err := json.Unmarshal(raw, &messages); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return messages
}

func TestEveryCodeHasATranslation(t *testing.T) {
	codes := codesUsedInHandlers(t)
	for _, lang := range translatedLanguages(t) {
		messages := loadLocale(t, lang)
		var missing []string
		for code := range codes {
			if messages[code] == "" {
				missing = append(missing, code)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s/errors.json is missing %d codes: %v", lang, len(missing), missing)
		}
	}
}

func TestNoOrphanTranslations(t *testing.T) {
	codes := codesUsedInHandlers(t)
	messages := loadLocale(t, "en")
	var orphans []string
	for code := range messages {
		if !codes[code] {
			orphans = append(orphans, code)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("errors.json has %d codes no handler returns: %v", len(orphans), orphans)
	}
}

// English is the canonical source, so every other language must cover exactly
// the same codes: a gap renders as English inside an otherwise translated UI.
func TestLocalesAgreeOnKeys(t *testing.T) {
	en := loadLocale(t, "en")
	for _, lang := range translatedLanguages(t) {
		if lang == "en" {
			continue
		}
		other := loadLocale(t, lang)
		for code := range en {
			if other[code] == "" {
				t.Errorf("code %q has no %s text", code, lang)
			}
		}
		for code := range other {
			if en[code] == "" {
				t.Errorf("code %q exists in %s but not in English", code, lang)
			}
		}
	}
}
