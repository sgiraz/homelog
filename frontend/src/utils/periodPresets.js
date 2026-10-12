import { dateOnly } from '@/utils/dateFormatter'

// Calendar-to-date windows. `compare` is what /expenses/stats weighs the total
// against: the same stretch of the previous month/year, not the days just
// before. Order is the tie-break when a range fits two presets (all of January
// is both "this month" and "this year").
export const PERIOD_PRESETS = [
  { key: 'thisMonth', compare: 'month', start: (now) => new Date(now.getFullYear(), now.getMonth(), 1) },
  { key: 'thisYear', compare: 'year', start: (now) => new Date(now.getFullYear(), 0, 1) }
]

/** `{ from, to }` ("YYYY-MM-DD") for a preset, ending on `now`. */
export function presetRange(preset, now = new Date()) {
  return { from: dateOnly(preset.start(now)), to: dateOnly(now) }
}

function fits(preset, from, to, now) {
  return Boolean(from) && to === dateOnly(now) && presetRange(preset, now).from === from
}

/**
 * The preset whose range is exactly `from`..`to`, or null (custom). `prefer` is
 * the key the user last picked, so a range that fits two presets keeps the one
 * they clicked.
 */
export function matchPreset(from, to, prefer = null, now = new Date()) {
  const candidates = [...PERIOD_PRESETS].sort((a, b) => (b.key === prefer) - (a.key === prefer))
  return candidates.find((p) => fits(p, from, to, now)) ?? null
}
