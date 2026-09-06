import { createI18n } from 'vue-i18n'

// Eagerly load every locale file under ./locales/{lang}/{component}.json.
// File-per-feature layout pairs 1:1 with Weblate components (one file = one
// translation component, easier context for translators).
const modules = import.meta.glob('./locales/**/*.json', {
  eager: true,
  import: 'default',
})

const messages = {}
for (const path in modules) {
  const match = path.match(/\.\/locales\/([^/]+)\/([^/]+)\.json$/)
  if (!match) continue
  const [, locale, component] = match
  if (!messages[locale]) messages[locale] = {}
  messages[locale][component] = modules[path]
}

// The set of locale directories IS the list of supported languages: dropping
// frontend/src/i18n/locales/<lang>/ in place is all it takes to add one. No
// hand-maintained array to forget — that array is why a translation could be
// merged and never appear in the app. A Go test checks models.SupportedLanguages
// still agrees with this list.
export const SUPPORTED_LOCALES = Object.keys(messages).sort()

// English is the canonical source language of every message file and the
// vue-i18n fallback, so it is also the default for a visitor whose browser
// asks for a language we do not have.
export const DEFAULT_LOCALE = 'en'

// Endonyms for the language picker: a language is named in itself, never
// translated. Falls back to the bare tag for a locale added without an entry.
const LANGUAGE_ENDONYMS = {
  it: 'Italiano',
  en: 'English',
  de: 'Deutsch',
  fr: 'Français',
  es: 'Español',
}

// Options for the settings language selector, in alphabetical endonym order.
export function localeOptions() {
  return SUPPORTED_LOCALES.map((code) => ({
    code,
    label: LANGUAGE_ENDONYMS[code] || code.toUpperCase(),
  })).sort((a, b) => a.label.localeCompare(b.label))
}

// Per-browser locale memory. Not a server setting: it is what lets pre-login
// screens and the shared demo account render in the visitor's own language.
const LOCALE_STORAGE_KEY = 'locale'

// Reads the locale the visitor last picked in this browser, if any.
export function getStoredLocale() {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY)
    return SUPPORTED_LOCALES.includes(stored) ? stored : null
  } catch {
    // Private mode / storage disabled — fall through to detection.
    return null
  }
}

// Remembers an explicit locale choice for this browser.
export function persistLocale(locale) {
  if (!SUPPORTED_LOCALES.includes(locale)) return
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // Nothing to do: the choice simply won't survive a reload.
  }
}

// Best supported match for the browser/OS language list, e.g. "en-GB" → "en".
export function detectBrowserLocale() {
  const tags = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const tag of tags) {
    if (!tag) continue
    const base = String(tag).toLowerCase().split('-')[0]
    if (SUPPORTED_LOCALES.includes(base)) return base
  }
  return DEFAULT_LOCALE
}

// Locale to use before (or instead of) a server-stored preference: an explicit
// choice made in this browser, else the system language.
export function resolveInitialLocale() {
  return getStoredLocale() ?? detectBrowserLocale()
}

export const i18n = createI18n({
  legacy: false,
  locale: resolveInitialLocale(),
  // EN is the canonical source language — fall back to it so missing keys
  // in any partial translation render in English, never as raw key paths.
  fallbackLocale: 'en',
  messages,
  globalInjection: true,
  missingWarn: import.meta.env.DEV,
  fallbackWarn: import.meta.env.DEV,
})

export function setI18nLocale(locale) {
  const resolved = SUPPORTED_LOCALES.includes(locale) ? locale : DEFAULT_LOCALE
  i18n.global.locale.value = resolved
  document.documentElement.setAttribute('lang', resolved)
}

export default i18n
