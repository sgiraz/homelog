package models

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The list of supported languages exists in two runtimes: this map, and
// SUPPORTED_LOCALES in frontend/src/i18n/index.js — which derives itself from
// the locale directories, so those directories are the real source of truth.
//
// This test is what keeps the Go copy honest. Without it a translation could be
// merged, appear in the picker, and still be rejected by NormalizeLanguage on
// registration — the user would sign up and land in English.
func TestSupportedLanguagesMatchTheLocaleDirectories(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("frontend locales not available (%v)", err)
	}

	var onDisk []string
	for _, e := range entries {
		if e.IsDir() {
			onDisk = append(onDisk, e.Name())
		}
	}
	sort.Strings(onDisk)

	var declared []string
	for lang := range SupportedLanguages {
		declared = append(declared, lang)
	}
	sort.Strings(declared)

	if len(onDisk) == 0 {
		t.Fatal("no locale directories found — the path is probably wrong")
	}

	inGo := map[string]bool{}
	for _, l := range declared {
		inGo[l] = true
	}
	for _, l := range onDisk {
		if !inGo[l] {
			t.Errorf("locale %q is translated but missing from models.SupportedLanguages: "+
				"the UI would offer it and the server would reject it", l)
		}
	}
	onDiskSet := map[string]bool{}
	for _, l := range onDisk {
		onDiskSet[l] = true
	}
	for _, l := range declared {
		if !onDiskSet[l] {
			t.Errorf("models.SupportedLanguages declares %q but there is no locales/%s directory", l, l)
		}
	}
}

// DefaultLanguage is the fallback of last resort, so it has to be a language
// the server actually accepts.
func TestDefaultLanguageIsSupported(t *testing.T) {
	if !SupportedLanguages[DefaultLanguage] {
		t.Fatalf("DefaultLanguage %q is not in SupportedLanguages", DefaultLanguage)
	}
}

func TestNormalizeLanguage(t *testing.T) {
	cases := map[string]string{
		"en":       "en",
		"EN":       "en",
		"en-GB":    "en",
		"en_US":    "en",
		"  it  ":   "it",
		"de":       "de",
		"ja":       DefaultLanguage,
		"":         DefaultLanguage,
		"nonsense": DefaultLanguage,
	}
	for in, want := range cases {
		if got := NormalizeLanguage(in); got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}
