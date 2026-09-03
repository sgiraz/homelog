import { i18n } from '@/i18n'

/**
 * Resolves an API failure into a message in the user's language.
 *
 * The server answers an error with a stable `error_code` plus an English
 * `error` string. The code is what gets translated here; the English text is
 * developer-facing (logs, curl) and is deliberately NOT shown to the user —
 * surfacing it would put English in an Italian UI, which is the very thing the
 * codes exist to prevent.
 *
 * `error_params` fills placeholders, e.g. `errors.currency_not_supported`
 * carries `{currency}`.
 *
 * @param {unknown} err - the rejected axios error
 * @param {string} [fallback] - message to show when the server sent no usable
 *   code; defaults to the generic server-error text
 * @returns {string}
 */
export function apiErrorMessage(err, fallback = '') {
  const data = err?.response?.data
  const code = data?.error_code

  if (code) {
    const key = `errors.${code}`
    const translated = i18n.global.t(key, data.error_params || {})
    // An unknown code (older server, newer client) renders the key path; treat
    // that as "no usable code" rather than showing it to the user.
    if (translated !== key) return translated
  }

  return fallback || i18n.global.t('errors.server_error')
}
