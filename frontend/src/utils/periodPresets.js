import { dateOnly } from '@/utils/dateFormatter'

// Calendar-to-date windows. `compare` is what /expenses/stats weighs the total
// against: the same stretch of the previous month/year, not the days just
// before.
export const PERIOD_PRESETS = [
  { key: 'thisMonth', compare: 'month', start: (now) => new Date(now.getFullYear(), now.getMonth(), 1) },
  { key: 'thisYear', compare: 'year', start: (now) => new Date(now.getFullYear(), 0, 1) }
]

/** `{ from, to }` ("YYYY-MM-DD") for a preset, ending on `now`. */
export function presetRange(preset, now = new Date()) {
  return { from: dateOnly(preset.start(now)), to: dateOnly(now) }
}
