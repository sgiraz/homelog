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
  // Fall back to IT (the source language) so partial translations during
  // migration degrade gracefully — missing EN keys render in IT, never as
  // raw key paths. Switch to 'en' after migration completes.
  fallbackLocale: 'it',
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
