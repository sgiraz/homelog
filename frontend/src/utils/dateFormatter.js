/**
 * Centralized date formatting utility.
 * All date display in the app should use these functions
 * to respect the user's locale/format preferences.
 *
 * Supported date_format values (stored in user settings DB):
 *   - 'DD/MM/YYYY'  → 31/12/2024  (default, European)
 *   - 'MM/DD/YYYY'  → 12/31/2024  (US)
 *   - 'YYYY-MM-DD'  → 2024-12-31  (ISO)
 *   - 'DD MMM YYYY' → 31 dic 2024 (short month name, locale-aware)
 */

// Map date_format setting → Intl locale + options
const FORMAT_CONFIG = {
  'DD/MM/YYYY': {
    locale: 'it-IT',
    options: { day: '2-digit', month: '2-digit', year: 'numeric' }
  },
  'MM/DD/YYYY': {
    locale: 'en-US',
    options: { day: '2-digit', month: '2-digit', year: 'numeric' }
  },
  'YYYY-MM-DD': {
    locale: 'sv-SE', // Swedish locale produces YYYY-MM-DD natively
    options: { day: '2-digit', month: '2-digit', year: 'numeric' }
  },
  'DD MMM YYYY': {
    locale: null, // will use language setting
    options: { day: '2-digit', month: 'short', year: 'numeric' }
  }
}

// Map language code → Intl locale string
const LANGUAGE_LOCALE = {
  it: 'it-IT',
  en: 'en-US'
}

/**
 * Get the Intl locale string for a language code.
 * @param {string} language - e.g. 'it', 'en'
 * @returns {string}
 */
function getLocale(language) {
  return LANGUAGE_LOCALE[language] || language || 'it-IT'
}

/**
 * Format a date string for display.
 *
 * @param {string|Date} dateStr - ISO date string or Date object
 * @param {object} settings - { date_format: 'DD/MM/YYYY', language: 'it' }
 * @returns {string} Formatted date string, or '-' if input is falsy
 */
export function formatDate(dateStr, settings = {}) {
  if (!dateStr) return '-'

  const date = dateStr instanceof Date ? dateStr : new Date(dateStr)
  if (isNaN(date.getTime())) return '-'

  const dateFormat = settings.date_format || 'DD/MM/YYYY'
  const language = settings.language || 'it'

  const config = FORMAT_CONFIG[dateFormat] || FORMAT_CONFIG['DD/MM/YYYY']
  const locale = config.locale || getLocale(language)

  return date.toLocaleDateString(locale, config.options)
}

/**
 * Format a period (start - end) for display.
 * Uses short month + year, e.g. "dic 2024 - gen 2025"
 *
 * @param {string|Date} start - Period start
 * @param {string|Date} end - Period end
 * @param {object} settings - { language: 'it' }
 * @returns {string}
 */
export function formatPeriod(start, end, settings = {}) {
  if (!start || !end) return '-'

  const language = settings.language || 'it'
  const locale = getLocale(language)
  const options = { month: 'short', year: 'numeric' }

  const startDate = start instanceof Date ? start : new Date(start)
  const endDate = end instanceof Date ? end : new Date(end)

  if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) return '-'

  return `${startDate.toLocaleDateString(locale, options)} - ${endDate.toLocaleDateString(locale, options)}`
}

/**
 * Format a compact period for tables (shorter than formatPeriod).
 * Shows "MM/YY - MM/YY" for monthly periods, full date otherwise.
 *
 * @param {string|Date} start
 * @param {string|Date} end
 * @param {object} settings - { language: 'it' }
 * @returns {string}
 */
export function formatPeriodCompact(start, end, settings = {}) {
  if (!start || !end) return '-'

  const language = settings.language || 'it'
  const locale = getLocale(language)

  const startDate = start instanceof Date ? start : new Date(start)
  const endDate = end instanceof Date ? end : new Date(end)

  if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) return '-'

  const opts = { month: '2-digit', year: '2-digit' }
  return `${startDate.toLocaleDateString(locale, opts)} – ${endDate.toLocaleDateString(locale, opts)}`
}

// ── Number formatting ──

/**
 * Map currency code → Intl currency symbol style
 */
const CURRENCY_MAP = {
  EUR: 'EUR',
  USD: 'USD',
  GBP: 'GBP'
}

/**
 * Format a number respecting user locale.
 *
 * @param {number} value
 * @param {object} settings - { language: 'it' }
 * @param {object} options - Intl.NumberFormat options override
 * @returns {string}
 */
export function formatNumber(value, settings = {}, options = {}) {
  if (value == null) return '-'
  const locale = getLocale(settings.language || 'it')
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 3,
    ...options
  }).format(value)
}

/**
 * Format a currency value respecting user locale and currency setting.
 *
 * @param {number} value
 * @param {object} settings - { language: 'it', currency: 'EUR' }
 * @param {object} options - Intl.NumberFormat options override
 * @returns {string}
 */
export function formatCurrency(value, settings = {}, options = {}) {
  const locale = getLocale(settings.language || 'it')
  const currency = CURRENCY_MAP[settings.currency] || settings.currency || 'EUR'
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    ...options
  }).format(value || 0)
}

/**
 * Format a difference value with +/- sign.
 *
 * @param {number} value
 * @param {object} settings - { language: 'it' }
 * @returns {string}
 */
export function formatDiff(value, settings = {}) {
  if (value == null) return '-'
  const locale = getLocale(settings.language || 'it')
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: 3,
    signDisplay: 'exceptZero'
  }).format(value)
  return formatted
}
