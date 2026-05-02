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

export const SUPPORTED_LOCALES = ['it', 'en']
export const DEFAULT_LOCALE = 'it'

export const i18n = createI18n({
  legacy: false,
  locale: DEFAULT_LOCALE,
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
