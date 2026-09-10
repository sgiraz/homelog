/**
 * Shared slice folding for categorical charts.
 *
 * Every pie in the app draws the same shape of data — rows with an amount, a
 * long tail of near-zero ones — so the "how many slices do we name" rule lives
 * here instead of being re-invented per view. The limit is the palette's slot
 * count (see useChartTheme.js): we never cycle colours, so anything past the
 * last slot folds into one neutral bucket.
 *
 * @param {Array} rows - source rows, any shape
 * @param {(row: any) => number} amountOf - reads the value of a row
 * @param {number} limit - how many rows keep their own colour
 * @returns {Array<{row: object|null, amount: number, count?: number}>}
 *   Sorted biggest first. The folded bucket has `row: null` and a `count`; it
 *   carries no source row, so callers know it is not clickable.
 */
export function foldSlices(rows, amountOf, limit) {
  const sorted = [...(rows ?? [])].sort((a, b) => amountOf(b) - amountOf(a))
  // Folding a single row would only rename it, so keep one row over the limit.
  if (!limit || sorted.length <= limit + 1) {
    return sorted.map((row) => ({ row, amount: amountOf(row) }))
  }
  const tail = sorted.slice(limit)
  return [
    ...sorted.slice(0, limit).map((row) => ({ row, amount: amountOf(row) })),
    {
      row: null,
      amount: tail.reduce((sum, row) => sum + amountOf(row), 0),
      count: tail.length
    }
  ]
}

/**
 * Slot colour per slice. The folded bucket takes the neutral token, and so does
 * anything past the last slot (when the user expands a long list): cycling the
 * palette would give two categories the same colour, which is worse than no
 * colour at all.
 */
export function sliceColors(slices, theme) {
  return slices.map((slice, i) => (slice.row && theme.series[i]) || theme.seriesOther)
}
