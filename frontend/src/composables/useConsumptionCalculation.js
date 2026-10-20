import { computed, watch } from 'vue'

/**
 * Composable that fills a bill's consumption from the provider's closing readings.
 *
 * @param {import('vue').Ref} form - reactive form object with bill fields
 * @param {import('vue').ComputedRef} utility - the parent utility (props.utility)
 * @param {import('vue').ComputedRef} isEditing - whether we're editing an existing bill
 * @param {Object|null} editingBill - the bill being edited (props.bill), for exclusion from previousBill lookup
 */
export function useConsumptionCalculation(form, utility, isEditing, editingBill) {

  // The bill closing right before this one. Periods overlap when a bill
  // re-bills from the last real reading, so its start says nothing: order by
  // period_end. A different meter starts from another zero: no previous bill.
  const previousBill = computed(() => {
    const currentEnd = new Date(form.value.period_end)
    if (isNaN(currentEnd.getTime())) return null

    let prev = null
    for (const bill of utility.value?.bills || []) {
      if (isEditing.value && editingBill?.id === bill.id) continue
      const billEnd = new Date(bill.period_end)
      if (isNaN(billEnd.getTime()) || billEnd >= currentEnd) continue
      if (!prev || billEnd > new Date(prev.period_end)) prev = bill
    }
    const serial = form.value.meter_serial?.trim()
    if (prev?.meter_serial && serial && prev.meter_serial !== serial) return null
    return prev
  })

  // The printed consumption (from the PDF, typed, or already saved) wins: only
  // a value filled in here is recomputed when the readings change.
  let autoConsumption = null
  function setConsumption(value) {
    const current = form.value.consumption_total
    if (current != null && current !== '' && current !== autoConsumption) return
    autoConsumption = Math.round(value * 1000) / 1000
    form.value.consumption_total = autoConsumption
  }

  function calculateConsumption() {
    const type = utility.value?.type
    const prev = previousBill.value
    if (!prev) return

    if (type === 'electricity') {
      let total = 0
      let hasPair = false
      for (const key of ['provider_reading_f1', 'provider_reading_f2', 'provider_reading_f3']) {
        const curr = parseFloat(form.value[key])
        if (!isNaN(curr) && prev[key] != null) { total += curr - prev[key]; hasPair = true }
      }
      if (!hasPair || total < 0) return
      setConsumption(total)
      return
    }

    if (type !== 'gas' && type !== 'water') return

    const current = parseFloat(form.value.provider_reading)
    if (isNaN(current) || prev.provider_reading == null) return
    const diff = current - prev.provider_reading
    if (diff < 0) return

    if (type === 'water') {
      setConsumption(diff)
      return
    }
    const C = parseFloat(form.value.conversion_coefficient)
    if (C > 0) setConsumption(diff * C)
  }

  watch(() => form.value.provider_reading, () => calculateConsumption())
  watch(() => form.value.conversion_coefficient, () => calculateConsumption())
  watch(() => form.value.provider_reading_f1, () => calculateConsumption())
  watch(() => form.value.provider_reading_f2, () => calculateConsumption())
  watch(() => form.value.provider_reading_f3, () => calculateConsumption())
  watch(() => form.value.period_end, () => calculateConsumption())
  watch(() => form.value.meter_serial, () => calculateConsumption())
}
