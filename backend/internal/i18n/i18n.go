// Package i18n holds the user-visible text the server writes by itself:
// generated expense descriptions, notification bodies, month names.
//
// The rule it exists to enforce: anything the *user* typed is data and is
// stored and returned verbatim, in whatever language they wrote it. Anything
// the *server* composes is app text and must be composed in the reader's
// language — never in a hardcoded Italian literal buried in a handler.
//
// The messages live in locales/<lang>.json, not in Go source, so adding a
// language is dropping a file in — no code change, no growing switch. Message
// ids are stable; English is the canonical text and is used whenever a
// language has no entry for an id.
//
// These files are separate from frontend/src/i18n/locales because the strings
// are server-only: shipping them in the client bundle would be dead weight.
// A Weblate component should point at this directory.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

//go:embed locales/*.json
var localeFiles embed.FS

// messages is language -> message id -> text.
var messages = loadMessages()

func loadMessages() map[string]map[string]string {
	entries, err := fs.Glob(localeFiles, "locales/*.json")
	if err != nil {
		panic(fmt.Sprintf("i18n: %v", err))
	}
	out := make(map[string]map[string]string, len(entries))
	for _, path := range entries {
		lang := strings.TrimSuffix(strings.TrimPrefix(path, "locales/"), ".json")
		raw, err := localeFiles.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("i18n %s: %v", path, err))
		}
		var byID map[string]string
		if err := json.Unmarshal(raw, &byID); err != nil {
			panic(fmt.Sprintf("i18n %s: %v", path, err))
		}
		out[lang] = byID
	}
	if len(out["en"]) == 0 {
		panic("i18n: no English messages embedded — there is nothing to fall back to")
	}
	return out
}

// Languages lists the languages the embedded catalogue covers.
func Languages() []string {
	langs := make([]string, 0, len(messages))
	for lang := range messages {
		langs = append(langs, lang)
	}
	return langs
}

// T returns the message for id in lang, with {placeholder} occurrences filled
// from args given as alternating key/value pairs. An unknown id returns the id
// itself, which surfaces the mistake instead of hiding it behind empty text.
func T(lang, id string, args ...string) string {
	text, ok := messages[models.NormalizeLanguage(lang)][id]
	if !ok {
		text, ok = messages["en"][id]
	}
	if !ok {
		return id
	}
	for i := 0; i+1 < len(args); i += 2 {
		text = strings.ReplaceAll(text, "{"+args[i]+"}", args[i+1])
	}
	return text
}

// UtilityType returns the localized label of a service type, falling back to
// the raw type for anything not in the catalogue.
func UtilityType(lang, utilityType string) string {
	id := "utility.type." + utilityType
	if label := T(lang, id); label != id {
		return label
	}
	return utilityType
}

// MonthShort returns the abbreviated month name (month is 1-based). Out of
// range returns "".
func MonthShort(lang string, month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return T(lang, "month.short."+strconv.Itoa(month))
}

// FormatAmount renders a monetary amount with the decimal separator of the
// language, which is itself a catalogue entry so a new language brings its own.
// No currency symbol: the caller's message decides whether one is wanted, and
// the user's currency is a separate setting.
func FormatAmount(lang string, v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if sep := T(lang, "format.decimal_separator"); sep != "" && sep != "." {
		s = strings.Replace(s, ".", sep, 1)
	}
	return s
}

// UserLanguage returns the language a user reads the app in, normalized. Any
// lookup failure falls back to models.DefaultLanguage rather than to a
// hardcoded literal at the call site.
func UserLanguage(db *gorm.DB, userID uint) string {
	var settings models.UserSettings
	if err := db.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return models.DefaultLanguage
	}
	return models.NormalizeLanguage(settings.Language)
}
