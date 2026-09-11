import { TokenType } from '@/utils/tokenizer'

/**
 * Colours of the PDF template wizard: one per group of token types.
 *
 * This is a categorical legend — green means "an amount", not "good" — so it
 * does not use the status tokens. It does not use the chart series palette
 * either: the highlights sit on top of the rendered PDF page, which is white
 * paper in both themes, while the series palette switches to its dark-surface
 * variant in dark mode.
 *
 * For the same reason the text on a highlight is always the dark shade, in
 * both themes: it sits on paper. Theme-aware classes (text-ink, dark:text-*)
 * turned it near-white on a light box in dark mode and made it unreadable.
 *
 * The legend is generated from this list, so a group can no longer have a
 * highlight colour without a legend entry (POD/PDR used to).
 *
 * Class names are written out in full: Tailwind finds them by scanning source.
 */
export const TOKEN_GROUPS = [
  {
    key: 'currency',
    labelKey: 'legendCurrency',
    types: [TokenType.CURRENCY],
    swatch: 'bg-green-400/60 border-green-500',
    overlay: 'bg-green-400/60 border-green-500 text-green-900',
    tooltip: 'text-green-400'
  },
  {
    key: 'date',
    labelKey: 'legendDate',
    types: [TokenType.DATE, TokenType.MONTH],
    swatch: 'bg-purple-400/60 border-purple-500',
    overlay: 'bg-purple-400/60 border-purple-500 text-purple-900',
    tooltip: 'text-purple-400'
  },
  {
    key: 'number',
    labelKey: 'legendNumber',
    types: [TokenType.NUMBER],
    swatch: 'bg-blue-400/60 border-blue-500',
    overlay: 'bg-blue-400/60 border-blue-500 text-blue-900',
    tooltip: 'text-blue-400'
  },
  {
    key: 'symbol',
    labelKey: 'legendSymbol',
    types: [TokenType.SYMBOL],
    swatch: 'bg-yellow-400/60 border-yellow-500',
    overlay: 'bg-yellow-400/60 border-yellow-500 text-yellow-900',
    tooltip: 'text-yellow-400'
  },
  {
    key: 'code',
    labelKey: 'legendCode',
    types: [TokenType.POD, TokenType.PDR],
    swatch: 'bg-orange-400/60 border-orange-500',
    overlay: 'bg-orange-400/60 border-orange-500 text-orange-900',
    tooltip: 'text-orange-400'
  }
]

// Plain text, punctuation and noise are not highlighted as a category.
const OTHER = { overlay: 'bg-gray-300/60 border-gray-400 text-gray-900', tooltip: 'text-gray-200' }

const GROUP_BY_TYPE = new Map(TOKEN_GROUPS.flatMap((group) => group.types.map((type) => [type, group])))

/** Highlight classes for a word on the PDF page. */
export function tokenOverlayClass(type) {
  return (GROUP_BY_TYPE.get(type) || OTHER).overlay
}

/** Text colour for the token type inside the dark tooltip. */
export function tokenTooltipClass(type) {
  return (GROUP_BY_TYPE.get(type) || OTHER).tooltip
}
