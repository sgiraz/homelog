import { i18n } from '@/i18n'

/**
 * Resolves the display name of a category or subcategory.
 *
 * Built-in categories carry a `slug` and are translated through
 * `categories.<slug>`; user-created ones (and default ones an admin renamed,
 * which the backend strips the slug from) have no slug and render the raw
 * `name` they were saved with, in every language — free-form user data is
 * never translated.
 *
 * Reads `i18n.global.t`, so calling it inside a computed keeps it reactive to
 * locale changes.
 *
 * @param {{slug?: string, name?: string}|null|undefined} category
 * @param {string} [fallback] label to use when there is no category at all
 * @returns {string}
 */
export function categoryLabel(category, fallback = '') {
  if (!category) return fallback
  const raw = category.name || fallback
  if (!category.slug) return raw
  const key = `categories.${category.slug}`
  const translated = i18n.global.t(key)
  // A slug with no message file entry falls back to the stored name rather
  // than rendering the raw key path.
  return translated === key ? raw : translated
}

/**
 * Same resolution for a `{ category_slug, category_name }` row as returned by
 * the /expenses/stats endpoint.
 *
 * @param {{category_slug?: string, category_name?: string}} row
 * @param {string} [fallback] label for the empty bucket (e.g. "No subcategory")
 * @returns {string}
 */
export function statsCategoryLabel(row, fallback = '') {
  return categoryLabel({ slug: row?.category_slug, name: row?.category_name }, fallback)
}
