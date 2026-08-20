// Currency codes and symbols only — the display name is NOT here.
// A name is user-visible text, so it lives in i18n under `common.currencies.<CODE>`
// and is resolved by the component that renders it (see `currencyLabel` below).
// This list is the single source: don't re-declare a subset inline in a modal.
export const currencies = [
  { code: 'EUR', symbol: '€' },
  { code: 'USD', symbol: '$' },
  { code: 'GBP', symbol: '£' },
  { code: 'CHF', symbol: 'Fr' },
  { code: 'JPY', symbol: '¥' },
  { code: 'CAD', symbol: 'C$' },
  { code: 'AUD', symbol: 'A$' },
  { code: 'SEK', symbol: 'kr' },
  { code: 'NOK', symbol: 'kr' },
  { code: 'DKK', symbol: 'kr' },
  { code: 'PLN', symbol: 'zł' },
  { code: 'CZK', symbol: 'Kč' },
  { code: 'HUF', symbol: 'Ft' },
  { code: 'TRY', symbol: '₺' },
  { code: 'BRL', symbol: 'R$' },
  { code: 'CNY', symbol: '¥' },
  { code: 'INR', symbol: '₹' },
  { code: 'KRW', symbol: '₩' },
  { code: 'THB', symbol: '฿' },
]

/**
 * Translated name of a currency, e.g. currencyName('USD', t) -> "US Dollar".
 * Falls back to the code itself for a currency with no message yet.
 * @param {string} code - ISO 4217 code
 * @param {Function} t - vue-i18n `t` from the calling component
 */
export function currencyName(code, t) {
  const key = `common.currencies.${code}`
  const name = t(key)
  return name === key ? code : name
}

/**
 * "USD — US Dollar", the option label used by every currency picker.
 */
export function currencyOptionLabel(code, t) {
  return `${code} — ${currencyName(code, t)}`
}
