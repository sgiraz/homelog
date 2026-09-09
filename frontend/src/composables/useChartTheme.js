import { ref, onScopeDispose, computed } from 'vue'

/**
 * Chart colours resolved from the CSS design tokens.
 *
 * Chart.js takes plain colour strings and cannot consume `var(--…)`, so the
 * tokens have to be read out of the DOM. Reading them is driven by a
 * MutationObserver on <html> rather than by watching the theme refs: the refs
 * change *before* the class/attribute lands on the element, so a read keyed on
 * them can resolve the palette of the mode we are leaving — which is how the
 * legend ended up dark-on-dark. The observer fires after the DOM change, so
 * what we read is always what is painted.
 *
 * `main.css` stays the single source of truth: no chart hardcodes a colour,
 * and the number of series slots comes from the tokens themselves, so widening
 * the palette is a CSS-only edit.
 */

// Upper bound for slot discovery; the real count is however many
// `--c-series-N` the stylesheet defines, counted from 1 until the first gap.
const MAX_SLOTS = 24

function readVar(name) {
  if (typeof document === 'undefined') return ''
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

/**
 * Neutral tokens are stored as "R G B" channels (Tailwind v4 needs them that
 * way for opacity modifiers). Chart.js parses colours with @kurkle/color,
 * which only understands the legacy comma syntax: hand it `rgb(146 157 173)`
 * and parsing fails silently, leaving the text pure black. Hence the rebuild
 * into `rgb(r, g, b)` / `rgba(r, g, b, a)`.
 * Returns undefined when the token is missing so Chart.js keeps its own
 * default instead of painting an unparseable string.
 */
function readChannels(name, alpha = 1) {
  const channels = readVar(name).split(/\s+/).filter(Boolean)
  if (channels.length < 3) return undefined
  const rgb = channels.slice(0, 3).join(', ')
  return alpha === 1 ? `rgb(${rgb})` : `rgba(${rgb}, ${alpha})`
}

function readPalette() {
  const series = []
  for (let slot = 1; slot <= MAX_SLOTS; slot++) {
    const colour = readVar(`--c-series-${slot}`)
    if (!colour) break
    series.push(colour)
  }
  return {
    /** Categorical colours, fixed slot order. Never cycle. */
    series,
    /** Neutral bucket for the folded tail. */
    seriesOther: readVar('--c-series-other'),
    /** Single-series charts wear the brand accent. */
    accent: readChannels('--c-accent'),
    accentSoft: readChannels('--c-accent-soft'),
    accentFill: readChannels('--c-accent', 0.12),
    /** Chart furniture. */
    tick: readChannels('--c-ink-muted'),
    grid: readChannels('--c-line'),
    surface: readChannels('--c-surface'),
    ink: readChannels('--c-ink')
  }
}

// Shared module-level state: one observer for the whole app, no matter how
// many charts are mounted.
const palette = ref(readPalette())
let observer = null
let subscribers = 0

/**
 * Publish only when a token actually moved. <html> attributes change for
 * reasons that have nothing to do with the palette, and every publish
 * re-renders every chart — so identity churn here is a rendering cost, not a
 * correctness gain.
 */
function refresh() {
  const next = readPalette()
  if (JSON.stringify(next) !== JSON.stringify(palette.value)) palette.value = next
}

function startObserving() {
  if (observer || typeof document === 'undefined') return
  observer = new MutationObserver(refresh)
  // Light/dark rides on the class, the colour theme on data-theme.
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class', 'data-theme']
  })
  // The stylesheet may not be applied yet on the very first read.
  refresh()
  if (!palette.value.series.length && typeof requestAnimationFrame === 'function') {
    requestAnimationFrame(refresh)
  }
}

export function useChartTheme() {
  subscribers++
  startObserving()

  onScopeDispose(() => {
    subscribers--
    if (subscribers === 0 && observer) {
      observer.disconnect()
      observer = null
    }
  })

  return computed(() => palette.value)
}
