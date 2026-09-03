// Package i18n holds the user-visible text the server writes by itself:
// generated expense descriptions, notification bodies and API error messages.
//
// The rule it exists to enforce: anything the *user* typed is data and is
// stored and returned verbatim, in whatever language they wrote it. Anything
// the *server* composes is app text and must be composed in the reader's
// language — never in a hardcoded Italian literal buried in a handler.
//
// Message ids are stable; the English text is the canonical source and is used
// whenever a language has no entry for an id.
package i18n

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

// messages maps a message id to its text per language. Placeholders are
// written {name} and filled by T.
var messages = map[string]map[string]string{
	// Utility/service type labels.
	"utility.type.electricity": {"en": "Electricity", "it": "Luce"},
	"utility.type.gas":         {"en": "Gas", "it": "Gas"},
	"utility.type.water":       {"en": "Water", "it": "Acqua"},
	"utility.type.waste":       {"en": "Waste", "it": "Rifiuti"},
	"utility.type.internet":    {"en": "Internet", "it": "Internet"},
	"utility.type.insurance":   {"en": "Insurance", "it": "Assicurazione"},
	"utility.type.affitto":     {"en": "Rent", "it": "Affitto"},
	"utility.type.mutuo":       {"en": "Mortgage", "it": "Mutuo"},

	// Abbreviated month names, used in generated labels and descriptions.
	"month.short.1":  {"en": "Jan", "it": "Gen"},
	"month.short.2":  {"en": "Feb", "it": "Feb"},
	"month.short.3":  {"en": "Mar", "it": "Mar"},
	"month.short.4":  {"en": "Apr", "it": "Apr"},
	"month.short.5":  {"en": "May", "it": "Mag"},
	"month.short.6":  {"en": "Jun", "it": "Giu"},
	"month.short.7":  {"en": "Jul", "it": "Lug"},
	"month.short.8":  {"en": "Aug", "it": "Ago"},
	"month.short.9":  {"en": "Sep", "it": "Set"},
	"month.short.10": {"en": "Oct", "it": "Ott"},
	"month.short.11": {"en": "Nov", "it": "Nov"},
	"month.short.12": {"en": "Dec", "it": "Dic"},

	// Expense descriptions generated when a bill or an instalment is paid.
	"bill.expense.description": {
		"en": "{type} bill - {month} {year}",
		"it": "Bolletta {type} - {month} {year}",
	},
	"bill.expense.installment": {
		"en": "{description} (Instalment {number}/{total})",
		"it": "{description} (Rata {number}/{total})",
	},

	// Notifications.
	"notification.expense_shared.title": {
		"en": "New shared expense: {description}",
		"it": "Nuova spesa condivisa: {description}",
	},
	"notification.expense_shared.body": {
		"en": "{payer} added an expense. Your share: {amount}.",
		"it": "{payer} ha inserito una spesa. La tua quota: {amount}.",
	},

	// Errors returned to the client.
	"error.utility.service_already_active": {
		"en": "A {type} service is already active for this property. Deactivate the existing one before creating a new one.",
		"it": "Esiste già un servizio {type} attivo per questa proprietà. Disattiva quello esistente prima di crearne uno nuovo.",
	},
	"error.utility.utility_already_active": {
		"en": "A {type} service is already active for this property. Deactivate the existing one before activating this.",
		"it": "Esiste già un'utenza {type} attiva per questa proprietà. Disattiva quella esistente prima di attivare questa.",
	},
}

// T returns the message for id in lang, with {placeholder} occurrences filled
// from args given as alternating key/value pairs. An unknown id returns the id
// itself, which surfaces the mistake instead of hiding it behind empty text.
func T(lang, id string, args ...string) string {
	byLang, ok := messages[id]
	if !ok {
		return id
	}
	text, ok := byLang[models.NormalizeLanguage(lang)]
	if !ok {
		text = byLang["en"]
	}
	for i := 0; i+1 < len(args); i += 2 {
		text = strings.ReplaceAll(text, "{"+args[i]+"}", args[i+1])
	}
	return text
}

// UtilityType returns the localized label of a service type, falling back to
// the raw type for anything not in the catalogue.
func UtilityType(lang, utilityType string) string {
	label := T(lang, "utility.type."+utilityType)
	if label == "utility.type."+utilityType {
		return utilityType
	}
	return label
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
// language. No currency symbol: the caller's message decides whether one is
// wanted, and the user's currency is a separate setting.
func FormatAmount(lang string, v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if models.NormalizeLanguage(lang) == "it" {
		s = strings.Replace(s, ".", ",", 1)
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
