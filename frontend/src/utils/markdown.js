import { Marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Markdown support for free-form text written by household members
 * (currently project descriptions).
 *
 * Two outputs, one parser:
 *   - `renderMarkdown` → sanitized HTML, for `v-html` in a detail view.
 *   - `stripMarkdown`  → plain text, for a truncated card preview.
 *
 * The preview never builds HTML at all: it reads the token stream, so a card
 * stays a plain `{{ }}` interpolation and `line-clamp` keeps working. Only the
 * detail view goes through `v-html`, and only on sanitizer output.
 */

/**
 * The supported subset. DOMPurify is the single authority on what survives:
 * `marked` emits more than this (tables, images, raw HTML the author typed),
 * and everything outside the list is dropped here rather than being disabled
 * tokenizer-side — one allowlist to audit instead of two half-filters, and no
 * reliance on what `marked` does when a tokenizer override declines a match.
 *
 * Tag content is kept when a tag is dropped (DOMPurify's default), so an
 * unsupported construct degrades to its text instead of vanishing: `#####`
 * reads as a plain line, a fenced block as inline code, a quote as its text.
 */
const ALLOWED_TAGS = [
  'h1', 'h2', 'h3', 'h4',
  'p', 'br',
  'strong', 'em',
  'ul', 'ol', 'li',
  'a', 'code',
]

/** Links are the only place an attribute is allowed through. */
const ALLOWED_ATTR = ['href']

/**
 * `javascript:` and `data:` URLs never reach the DOM. DOMPurify's own default
 * is wider than we need (it permits tel:, sms:, ftp: …); a description links
 * to a page or an address, nothing else.
 */
const ALLOWED_URI_REGEXP = /^(?:https?|mailto):/i

const SANITIZE_CONFIG = { ALLOWED_TAGS, ALLOWED_ATTR, ALLOWED_URI_REGEXP }

/**
 * A parser instance of our own rather than the shared `marked` singleton, so
 * these options cannot leak into — or be overwritten by — any other caller.
 *
 * `breaks: true` because the text is typed in a plain textarea by people who
 * are not writing Markdown on purpose: a single Enter has to behave like a
 * line break, not be swallowed into the previous paragraph.
 */
const parser = new Marked({
  gfm: true,
  breaks: true,
  async: false,
})

/**
 * A DOMPurify instance of our own, for the same reason: `addHook` is
 * instance-global, and registering it on the default export would silently
 * rewrite links for every other consumer of the library.
 */
const purifier = DOMPurify(window)

// Markdown links point outside the app, so they open in a new tab and are
// denied access to `window.opener` and to our referrer.
purifier.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A' && node.hasAttribute('href')) {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer nofollow')
  }
})

/**
 * Renders Markdown to HTML that is safe to pass to `v-html`.
 *
 * @param {string|null|undefined} source - raw text as the author typed it
 * @returns {string} sanitized HTML, or '' when there is nothing to render
 */
export function renderMarkdown(source) {
  if (typeof source !== 'string' || !source.trim()) return ''

  return purifier.sanitize(parser.parse(source), SANITIZE_CONFIG)
}

/* ────────────────────────────────────────────────────────────────────────────
   Plain-text flattening
   ──────────────────────────────────────────────────────────────────────────── */

/**
 * Token types that open a block. Inline tokens are contiguous in the source
 * and already carry their own spacing ("del " before an `em`), so they are
 * concatenated; blocks are not, and need a separator or two paragraphs run
 * into one word.
 */
const BLOCK_TOKENS = new Set([
  'paragraph', 'heading', 'list', 'list_item',
  'code', 'blockquote', 'table', 'hr', 'html', 'space',
])

/** Markup that produces no text of its own but does separate words. */
const WHITESPACE_TOKENS = new Set(['space', 'br'])

/**
 * Elements whose content is code rather than prose. DOMPurify discards them
 * wholesale (content included) instead of keeping the text, so the flattener
 * has to do the same or a preview would read `body{display:none}` where the
 * detail view correctly shows nothing.
 */
const CODE_ELEMENT = /<(script|style)\b[\s\S]*?(?:<\/\1\s*>|$)/gi

/** Any remaining tag, in raw HTML the author typed. */
const ANY_TAG = /<[^>]*>/g

/**
 * Reduces a token and its children to their text, dropping the markup.
 *
 * @param {object} token - a `marked` token
 * @returns {string}
 */
function tokenText(token) {
  if (WHITESPACE_TOKENS.has(token.type)) return ' '

  // An image is the one construct DOMPurify removes outright — a void element
  // has no content to keep — so its alt text must not surface here either.
  if (token.type === 'image') return ''

  // Raw HTML is not part of the subset: the sanitizer drops the tags and keeps
  // the prose between them, and this mirrors that. The regexes are cosmetic,
  // NOT a sanitizer — the result is interpolated as text, never as HTML.
  if (token.type === 'html') {
    return (token.text ?? '').replace(CODE_ELEMENT, '').replace(ANY_TAG, '')
  }

  // A list holds its entries under `items`, a table its cells under
  // `header`/`rows`; every other container nests under `tokens`.
  if (token.type === 'list') return tokensText(token.items)
  if (token.type === 'table') return tokensText([...token.header, ...token.rows.flat()])
  if (Array.isArray(token.tokens) && token.tokens.length) return tokensText(token.tokens)

  return token.text ?? ''
}

/**
 * @param {object[]} tokens
 * @returns {string}
 */
function tokensText(tokens) {
  let text = ''
  for (const token of tokens) {
    const part = tokenText(token)
    if (!part) continue
    text += text && BLOCK_TOKENS.has(token.type) ? ` ${part}` : part
  }
  return text
}

/**
 * Flattens Markdown to a single line of plain text, for a truncated preview.
 *
 * Headings, list bullets and emphasis markers disappear instead of being shown
 * as literal `##` / `**`, and the result is safe to interpolate — no HTML is
 * produced along the way.
 *
 * @param {string|null|undefined} source - raw text as the author typed it
 * @returns {string} plain text, or '' when there is nothing to show
 */
export function stripMarkdown(source) {
  if (typeof source !== 'string' || !source.trim()) return ''

  return tokensText(parser.lexer(source))
    .replace(/\s+/g, ' ')
    .trim()
}
