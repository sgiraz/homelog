package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The labels under internal/database/locales are copies of
// frontend/src/i18n/locales/<lang>/categories.json, produced by
// scripts/sync-category-names.mjs. Go cannot embed files outside its module, so
// the duplication is unavoidable — but it must not be silent. These tests turn
// "kept in sync by a comment" into "kept in sync by the build".

func frontendLocalesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("frontend locales not available (%v)", err)
	}
	return dir
}

func readJSONMap(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

// A stale copy would silently export the old label — or, after a rename, the
// wrong one. Re-run scripts/sync-category-names.mjs when this fails.
//
// It iterates the FRONTEND directories, not the embedded ones: a language
// translated but never synced would otherwise be invisible here, and its users
// would silently get English category names in the CSV export.
func TestEmbeddedNamesMatchTheFrontendSource(t *testing.T) {
	dir := frontendLocalesDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lang := e.Name()
		sourcePath := filepath.Join(dir, lang, "categories.json")
		if _, err := os.Stat(sourcePath); err != nil {
			// A partially translated language is fine: DefaultCategoryName
			// falls back to English, matching the UI's fallbackLocale.
			continue
		}

		source := readJSONMap(t, sourcePath)
		embedded, ok := categoryNames[lang]
		if !ok {
			t.Errorf("%s has categories.json but no embedded copy — run scripts/sync-category-names.mjs", lang)
			continue
		}

		if len(source) != len(embedded) {
			t.Errorf("%s: embedded copy has %d labels, source has %d — run scripts/sync-category-names.mjs",
				lang, len(embedded), len(source))
		}
		for slug, want := range source {
			if got := embedded[slug]; got != want {
				t.Errorf("%s/%s: embedded %q, source %q — run scripts/sync-category-names.mjs", lang, slug, got, want)
			}
		}
	}
}

// A slug with no label renders as an empty category in the CSV export.
func TestEverySlugHasALabelInEveryLanguage(t *testing.T) {
	var slugs []string
	for _, cat := range DefaultCategories {
		slugs = append(slugs, cat.Slug)
		for _, sub := range cat.Subs {
			slugs = append(slugs, sub.Slug)
		}
	}

	for _, lang := range LocalizedCategoryLanguages() {
		var missing []string
		for _, slug := range slugs {
			if categoryNames[lang][slug] == "" {
				missing = append(missing, slug)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s is missing %d labels: %v", lang, len(missing), missing)
		}
	}
}

// A label with no slug is dead weight that a translator still has to translate.
func TestNoOrphanLabels(t *testing.T) {
	known := map[string]bool{}
	for _, cat := range DefaultCategories {
		known[cat.Slug] = true
		for _, sub := range cat.Subs {
			known[sub.Slug] = true
		}
	}
	for _, lang := range LocalizedCategoryLanguages() {
		var orphans []string
		for slug := range categoryNames[lang] {
			if !known[slug] {
				orphans = append(orphans, slug)
			}
		}
		sort.Strings(orphans)
		if len(orphans) > 0 {
			t.Errorf("%s has %d labels no category uses: %v", lang, len(orphans), orphans)
		}
	}
}

// English is the canonical language and the fallback DefaultCategoryName uses,
// so it must always be complete.
func TestEnglishIsAlwaysPresent(t *testing.T) {
	if len(categoryNames["en"]) == 0 {
		t.Fatal("no English labels embedded: DefaultCategoryName has nothing to fall back to")
	}
}
