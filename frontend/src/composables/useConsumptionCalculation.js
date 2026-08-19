import { computed, watch } from 'vue'

/**
 * Composable that encapsulates consumption calculation logic for bills.
 *
 * @param {import('vue').Ref} form - reactive form object with bill fields
 * @param {import('vue').ComputedRef} utility - the parent utility (props.utility)
 * @param {import('vue').ComputedRef} isEditing - whether we're editing an existing bill
 * @param {Object|null} editingBill - the bill being edited (props.bill), for exclusion from previousBill lookup
 */
export function useConsumptionCalculation(form, utility, isEditing, editingBill) {

  // Find the previous bill based on the period_start of the bill being uploaded.
  // Looks for the most recent bill whose period_end is before the current bill's period_start.
  const previousBill = computed(() => {
    const bills = utility.value?.bills
    if (!bills?.length) return null

    const currentStart = form.value.period_start
    if (!currentStart) return null
    const currentStartDate = new Date(currentStart)
    if (isNaN(currentStartDate.getTime())) return null

    // Bills are sorted by period_end DESC from backend
    for (const bill of bills) {
      if (isEditing.value && editingBill?.id === bill.id) continue
      const billEnd = new Date(bill.period_end)
      if (isNaN(billEnd.getTime())) continue
      if (billEnd <= currentStartDate) {
        return bill
      }
    }
    return null
  })

  const previousBillHasEstimate = computed(() => {
    return previousBill.value?.estimated_consumption != null
  })

  // Get the effective previous reading: use estimated_reading if previous bill had an estimate
  const previousReading = computed(() => {
    const prev = previousBill.value
    if (!prev) return null
    if (prev.estimated_reading != null) return prev.estimated_reading
    return prev.provider_reading
  })

  // Calculate estimated consumption from estimated_reading and provider_reading
  const calculatedEstimatedConsumption = computed(() => {
    if (!form.value.has_estimated) return null
    if (utility.value?.type === 'water') {
      const estReading = parseFloat(form.value.estimated_reading)
      const provReading = parseFloat(form.value.provider_reading)
      if (isNaN(estReading) || isNaN(provReading)) return null
      const diff = estReading - provReading
      if (diff < 0) return null
      return Math.round(diff * 1000000) / 1000000
    }
    const estReading = parseFloat(form.value.estimated_reading)
    const provReading = parseFloat(form.value.provider_reading)
    const C = parseFloat(form.value.conversion_coefficient)
    if (isNaN(estReading) || isNaN(provReading) || isNaN(C) || C <= 0) return null
    const diff = estReading - provReading
    if (diff < 0) return null
    return Math.round(diff * C * 1000000) / 1000000
  })

  // Calculate consumption from provider readings difference
  function calculateConsumption() {
    const type = utility.value?.type
    const prev = previousBill.value

    if (type === 'electricity') {
      if (!prev) return
      const f1Curr = parseFloat(form.value.provider_reading_f1)
      const f2Curr = parseFloat(form.value.provider_reading_f2)
      const f3Curr = parseFloat(form.value.provider_reading_f3)
      const f1Prev = prev.provider_reading_f1
      const f2Prev = prev.provider_reading_f2
      const f3Prev = prev.provider_reading_f3
      let total = 0
      let hasPair = false
      if (!isNaN(f1Curr) && f1Prev != null) { total += f1Curr - f1Prev; hasPair = true }
      if (!isNaN(f2Curr) && f2Prev != null) { total += f2Curr - f2Prev; hasPair = true }
      if (!isNaN(f3Curr) && f3Prev != null) { total += f3Curr - f3Prev; hasPair = true }
      if (!hasPair || total < 0) return
      form.value.consumption_total = Math.round(total * 1000) / 1000
      return
    }

    if (type !== 'gas' && type !== 'water') return

    const current = parseFloat(form.value.provider_reading)
    if (!current || isNaN(current)) return
    const prevReadingVal = previousReading.value
    if (prevReadingVal == null) return

    const diff = current - prevReadingVal
    if (diff < 0) return

    if (type === 'gas') {
      const C = parseFloat(form.value.conversion_coefficient)
      if (C > 0) {
        let consumption = diff * C
        // If previous bill had estimated_consumption but NO estimated_reading (legacy data),
        // subtract it because previousReading used provider_reading as base.
        const prevBill = previousBill.value
        if (previousBillHasEstimate.value && prevBill?.estimated_reading == null) {
          const prevEstimated = parseFloat(form.value.previous_estimated_consumption)
          if (!isNaN(prevEstimated) && prevEstimated > 0) {
            consumption -= prevEstimated
          }
        }
        if (calculatedEstimatedConsumption.value != null) {
          consumption += calculatedEstimatedConsumption.value
        }
        form.value.consumption_total = Math.round(consumption * 1000) / 1000
      }
    } else { //TODO: water consumption needs to be verified (not clear to me)
      let consumption = diff
      const prevBill = previousBill.value
      if (previousBillHasEstimate.value && prevBill?.estimated_reading == null) {
        const prevEstimated = parseFloat(form.value.previous_estimated_consumption)
        if (!isNaN(prevEstimated) && prevEstimated > 0) {
          consumption -= prevEstimated
        }
      }
      if (calculatedEstimatedConsumption.value != null) {
        consumption += calculatedEstimatedConsumption.value
      }
      form.value.consumption_total = Math.round(consumption * 1000) / 1000
    }
  }

  // Watchers that trigger recalculation
  watch(() => form.value.provider_reading, () => calculateConsumption())
  watch(() => form.value.conversion_coefficient, () => calculateConsumption())
  watch(() => form.value.provider_reading_f1, () => calculateConsumption())
  watch(() => form.value.provider_reading_f2, () => calculateConsumption())
  watch(() => form.value.provider_reading_f3, () => calculateConsumption())
  watch(() => form.value.previous_estimated_consumption, () => calculateConsumption())
  watch(() => form.value.estimated_reading, () => calculateConsumption())
  watch(() => form.value.has_estimated, () => calculateConsumption())
  watch(() => form.value.period_start, () => calculateConsumption())

  return {
    previousBill,
    previousBillHasEstimate,
    previousReading,
    calculatedEstimatedConsumption,
    calculateConsumption
  }
}
