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

func loadLocale(t *testing.T, lang string) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", lang, "errors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("locale file not available (%v) — skipping the parity check", err)
	}
	var messages map[string]string
	if err := json.Unmarshal(raw, &messages); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return messages
}

func TestEveryCodeHasATranslation(t *testing.T) {
	codes := codesUsedInHandlers(t)
	for _, lang := range []string{"en", "it"} {
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

func TestLocalesAgreeOnKeys(t *testing.T) {
	en := loadLocale(t, "en")
	it := loadLocale(t, "it")
	for code := range en {
		if it[code] == "" {
			t.Errorf("code %q has no Italian text", code)
		}
	}
	for code := range it {
		if en[code] == "" {
			t.Errorf("code %q has no English text", code)
		}
	}
}
