package handlers

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Two rules the repo states but nothing enforced, which is how Italian strings
// and hardcoded "it"/"EUR" defaults survived the i18n migration:
//
//  1. Source text is English. Comments and the strings the server composes
//     read in English; what the user should see in their own language goes
//     through internal/i18n or comes back as an error code. This is not an
//     anti-Italian rule — German or French prose in a comment is as wrong.
//  2. A starting language or currency comes from models.DefaultLanguage /
//     models.DefaultCurrency. A literal in a fallback silently wins over the
//     constant — an account with no settings row got an Italian UI.
//
// Rule 1 is checked against the locale catalogues rather than a hand-written
// word list, because the languages the app supports are the directories under
// i18n/locales: a language added later is covered without editing this file.

// scanDirs are the Go packages that answer requests or seed rows for them.
var scanDirs = []string{
	".",
	filepath.Join("..", "middleware"),
	filepath.Join("..", "database"),
}

// dataFiles hold string literals that are data, not text the server composes:
// the words printed on an Italian bill that the parser matches on, the seed
// names written into categories.name, and the demo's authored sample content.
// Rule 1 exempts their strings — never their comments, and never rule 2.
var dataFiles = map[string]bool{
	"demo.go":        true, // authored sample household
	"categories.go":  true, // CategorySpec.SeedName, written to categories.name
	"pdf_default.go": true, // Italian bill vocabulary the extractor matches on
}

// codeValues are foreign words that are real values in the code, so a comment
// naming one is accurate rather than untranslated. Each is a migration, not a
// cleanup — see issue #59. Delete entries here as they are resolved.
var codeValues = map[string]bool{
	"affitto":  true, // models.Utility.Type, stored in the database
	"mutuo":    true, // models.Utility.Type, stored in the database
	"bilancio": true, // expenses tab id, also /expenses?tab=bilancio
	"lista":    true, // expenses tab id
	"debiti":   true, // expenses tab id
}

// technicalTerms look foreign only because a locale file happens to use them;
// they are the ordinary English spelling in a technical sentence.
var technicalTerms = map[string]bool{
	"data": true, "file": true, "import": true, "export": true,
	"error": true, "info": true, "test": true, "demo": true,
	"token": true, "server": true, "client": true, "cache": true,
	"reset": true, "link": true, "online": true, "offline": true,
	"backup": true, "label": true, "normale": true, "menu": true,
	// Collide with another language: "lang" is also German for "long",
	// "quote" is also Italian for "shares".
	"lang": true, "quote": true,
}

var wordRe = regexp.MustCompile(`\p{L}{4,}`)

// localeWords returns every word of four letters or more in the message
// catalogues of lang, lowercased.
func localeWords(t *testing.T, lang string) map[string]bool {
	t.Helper()
	words := map[string]bool{}

	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			for _, w := range wordRe.FindAllString(x, -1) {
				words[strings.ToLower(w)] = true
			}
		case map[string]any:
			for _, child := range x {
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}

	paths, _ := filepath.Glob(filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", lang, "*.json"))
	paths = append(paths,
		filepath.Join("..", "i18n", "locales", lang+".json"),
		filepath.Join("..", "database", "locales", "categories."+lang+".json"),
	)

	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue // not every catalogue exists for every language
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		walk(doc)
	}
	return words
}

// foreignWords is every word the non-English catalogues use and the English
// ones do not.
func foreignWords(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("frontend locales not available (%v)", err)
	}

	english := localeWords(t, "en")
	if len(english) < 500 {
		t.Fatalf("English catalogue yielded only %d words — the scan is probably broken", len(english))
	}

	foreign := map[string]bool{}
	langs := 0
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "en" {
			continue
		}
		langs++
		for w := range localeWords(t, e.Name()) {
			if isEnglish(w, english) || technicalTerms[w] || codeValues[w] {
				continue
			}
			foreign[w] = true
		}
	}
	if langs == 0 {
		t.Skip("no non-English locale to derive foreign vocabulary from")
	}
	return foreign
}

// isEnglish folds a trailing plural "s" both ways, so a catalogue that uses
// "token" does not make "tokens" look foreign in a comment.
func isEnglish(w string, english map[string]bool) bool {
	return english[w] || english[strings.TrimSuffix(w, "s")] || english[w+"s"]
}

// minForeignWords is how many distinct foreign words make a string a foreign
// sentence rather than a coincidence. Two keeps the test quiet on a single
// ambiguous word — one shared with English, a proper noun, a product name —
// at the cost of missing a lone stray word, which review catches. A test that
// cries wolf gets deleted instead of obeyed.
const minForeignWords = 2

func reportForeign(t *testing.T, where, text string, foreign map[string]bool) {
	t.Helper()
	seen := map[string]bool{}
	var hits []string
	for _, w := range wordRe.FindAllString(text, -1) {
		lw := strings.ToLower(w)
		if foreign[lw] && !seen[lw] {
			seen[lw] = true
			hits = append(hits, lw)
		}
	}
	if len(hits) < minForeignWords {
		return
	}
	flat := strings.Join(strings.Fields(text), " ")
	if len(flat) > 120 {
		flat = flat[:120] + "…"
	}
	t.Errorf("%s: source text is not English (%s): %q\n\t"+
		"comments are English; user-facing text goes through i18n or an error code",
		where, strings.Join(hits, ", "), flat)
}

func TestSourceTextIsEnglish(t *testing.T) {
	foreign := foreignWords(t)
	fset, files := parseDirs(t)

	for path, f := range files {
		base := filepath.Base(path)

		for _, group := range f.Comments {
			for _, c := range group.List {
				reportForeign(t, where(base, fset, c.Pos()), c.Text, foreign)
			}
		}

		if dataFiles[base] {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			reportForeign(t, where(base, fset, lit.Pos()), val, foreign)
			return true
		})
	}
}

// camelWords splits an identifier into its words: readingsSection → readings,
// section. Runs of capitals (PDFURL) stay together rather than becoming
// one-letter fragments.
var camelWords = regexp.MustCompile(`\p{Lu}+(?:\p{Ll}+)?|\p{Ll}+`)

// Identifiers are checked one word at a time, so a single foreign word is
// enough: lettureSection has no second word to corroborate it.
func TestIdentifiersAreEnglish(t *testing.T) {
	foreign := foreignWords(t)
	fset, files := parseDirs(t)

	for path, f := range files {
		base := filepath.Base(path)
		seen := map[string]bool{}

		var visit func(n ast.Node) bool
		visit = func(n ast.Node) bool {
			// In x.Foo, Foo was named by whoever declared it, not by us.
			if sel, ok := n.(*ast.SelectorExpr); ok {
				ast.Inspect(sel.X, visit)
				return false
			}
			id, ok := n.(*ast.Ident)
			if !ok || seen[id.Name] {
				return true
			}
			seen[id.Name] = true
			for _, w := range camelWords.FindAllString(id.Name, -1) {
				lw := strings.ToLower(w)
				if len(lw) < 4 || !foreign[lw] {
					continue
				}
				t.Errorf("%s: identifier %q contains the non-English word %q",
					where(base, fset, id.Pos()), id.Name, lw)
			}
			return true
		}
		ast.Inspect(f, visit)
	}
}

func where(base string, fset *token.FileSet, pos token.Pos) string {
	return base + ":" + strconv.Itoa(fset.Position(pos).Line)
}

var frontendComment = regexp.MustCompile(`(?s)<!--.*?-->|/\*.*?\*/`)
var lineComment = regexp.MustCompile(`(?m)^\s*//[^\n]*$`)

// The frontend has no AST here, so only comments are checked: a string in a
// .vue file is as likely to be an i18n key or a class list as it is prose.
func TestFrontendCommentsAreEnglish(t *testing.T) {
	foreign := foreignWords(t)

	root := filepath.Join("..", "..", "..", "frontend", "src")
	var paths []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil //nolint:nilerr // a missing frontend is handled below
		}
		if ext := filepath.Ext(p); ext != ".vue" && ext != ".js" {
			return nil
		}
		if strings.Contains(filepath.ToSlash(p), "/i18n/locales/") {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil || len(paths) == 0 {
		t.Skipf("frontend sources not available (%v)", err)
	}
	if len(paths) < 50 {
		t.Fatalf("found only %d frontend files — the scan is probably broken", len(paths))
	}

	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		src := string(raw)
		for _, re := range []*regexp.Regexp{frontendComment, lineComment} {
			for _, m := range re.FindAllStringIndex(src, -1) {
				line := strings.Count(src[:m[0]], "\n") + 1
				reportForeign(t, filepath.Base(p)+":"+strconv.Itoa(line), src[m[0]:m[1]], foreign)
			}
		}
	}
}

// Language and Currency struct fields must come from the constants, so one
// place decides what a new account starts with.
func TestNoHardcodedLanguageOrCurrencyDefault(t *testing.T) {
	fset, files := parseDirs(t)
	for path, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || (key.Name != "Language" && key.Name != "Currency") {
				return true
			}
			lit, ok := kv.Value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			t.Errorf("%s:%d: %s is assigned the literal %s — use models.DefaultLanguage / models.DefaultCurrency",
				filepath.Base(path), fset.Position(lit.Pos()).Line, key.Name, lit.Value)
			return true
		})
	}
}

// parseDirs parses every non-test Go file under the scanned directories,
// keeping comments.
func parseDirs(t *testing.T) (*token.FileSet, map[string]*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for _, dir := range scanDirs {
		paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		for _, p := range paths {
			if strings.HasSuffix(p, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", p, err)
			}
			files[p] = f
		}
	}
	if len(files) < 10 {
		t.Fatalf("parsed only %d files — the scan is probably broken", len(files))
	}
	return fset, files
}
