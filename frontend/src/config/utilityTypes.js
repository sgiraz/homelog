/**
 * Service types: icon and identity colour.
 *
 * These colours are identity, not status — electricity is yellow the way a
 * category keeps its slot, whatever the bill's state. They used to be copied
 * into six components and had already drifted (water was blue in three and
 * cyan in two). This is the only map; views ask it instead of keeping one.
 *
 * The emoji carries the identity on its own, so the tint is decoration and is
 * not held to the contrast bars that apply to the status tokens.
 *
 * Class names are written out in full on purpose: Tailwind finds them by
 * scanning source text, so they cannot be assembled from a hue variable.
 *
 * Which types have a meter is NOT decided here: models.MeteredByType on the
 * server is the single source of truth for that.
 */
export const UTILITY_TYPES = {
  electricity: {
    icon: '⚡',
    tile: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
    iconColor: 'text-yellow-500'
  },
  gas: {
    icon: '🔥',
    tile: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800',
    iconColor: 'text-orange-500'
  },
  water: {
    icon: '💧',
    tile: 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800',
    iconColor: 'text-cyan-500'
  },
  waste: {
    icon: '♻️',
    tile: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
    iconColor: 'text-green-500'
  },
  internet: {
    icon: '🌐',
    tile: 'bg-indigo-50 dark:bg-indigo-900/20 border-indigo-200 dark:border-indigo-800',
    iconColor: 'text-indigo-500'
  },
  insurance: {
    icon: '🛡️',
    tile: 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-200 dark:border-emerald-800',
    iconColor: 'text-emerald-500'
  },
  affitto: {
    icon: '🏠',
    tile: 'bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800',
    iconColor: 'text-purple-500'
  },
  mutuo: {
    icon: '🏦',
    tile: 'bg-sky-50 dark:bg-sky-900/20 border-sky-200 dark:border-sky-800',
    iconColor: 'text-sky-500'
  }
}

// An unknown type (added server-side before the client knows it) falls back to
// a neutral tile rather than borrowing another type's identity.
const FALLBACK = { icon: '📄', tile: 'bg-surface-2 border-line', iconColor: 'text-ink-muted' }

/** Icon and colours for a service type. */
export function utilityTypeStyle(type) {
  return UTILITY_TYPES[type] || FALLBACK
}
