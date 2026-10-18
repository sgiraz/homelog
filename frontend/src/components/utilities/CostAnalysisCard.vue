<template>
  <Card class="p-4 space-y-4">
    <div class="flex items-center justify-between">
      <h3 class="text-lg font-semibold text-ink">{{ t('utilities.costAnalysis.title') }}</h3>
      <span v-if="loading" class="text-sm text-ink-muted">{{ t('utilities.costAnalysis.loading') }}</span>
    </div>

    <p v-if="data && !data.current.bill_count" class="text-sm text-ink-muted">{{ t('utilities.costAnalysis.empty') }}</p>

    <template v-else-if="data">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div v-for="tile in tiles" :key="tile.id" class="rounded-lg bg-surface-2 p-3">
          <div class="text-xs text-ink-muted">{{ tile.label }}</div>
          <div class="text-xl font-bold text-ink mt-0.5">{{ tile.value }}</div>
          <div v-if="tile.change != null" class="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-muted">
            <Badge :variant="changeVariant(tile.change)">{{ formatChange(tile.change) }}</Badge>
            <span>{{ t('utilities.costAnalysis.previous', { value: tile.previous }) }}</span>
          </div>
          <div v-if="tile.hint" class="mt-1 text-xs text-ink-faint">{{ tile.hint }}</div>
        </div>
      </div>

      <div v-if="isMetered && data.current.consumption_gap != null" class="space-y-1">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-sm font-medium text-ink-soft">{{ t('utilities.costAnalysis.billedVsMeasured') }}</span>
          <Badge :variant="GAP_VARIANTS[data.current.gap_level || 'ok']">
            {{ t(`utilities.costAnalysis.gap.${data.current.gap_level || 'ok'}`) }}
          </Badge>
        </div>
        <div class="text-sm text-ink">
          {{ t('utilities.costAnalysis.billedMeasuredValues', {
            billed: fmtNum(data.current.billed_consumption),
            measured: fmtNum(data.current.measured_consumption),
            gap: fmtDiff(data.current.consumption_gap),
            unit: consumptionUnit,
          }) }}
        </div>
      </div>

      <div v-if="isMetered && data.anomalies.length" class="space-y-2">
        <div class="text-sm font-medium text-ink-soft">{{ t('utilities.costAnalysis.anomaliesTitle') }}</div>
        <ul class="space-y-2">
          <li v-for="a in data.anomalies" :key="`${a.kind}-${a.bill_id}-${a.register}`" class="flex items-start gap-2 text-sm text-ink">
            <Badge :variant="a.kind === 'reading_decrease' ? 'danger' : 'warning'">
              {{ t(`utilities.costAnalysis.anomaly.${a.kind}.badge`) }}
            </Badge>
            <span>{{ anomalyText(a) }}</span>
          </li>
        </ul>
      </div>
    </template>
  </Card>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { utilitiesAPI } from '@/api/client'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency, formatNumber, formatDiff, formatDate } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Badge from '@/components/common/Badge.vue'

defineOptions({ name: 'CostAnalysisCard' })

const props = defineProps({
  utilityId: { type: Number, required: true },
  isMetered: { type: Boolean, default: false },
  consumptionUnit: { type: String, default: '' },
  from: { type: String, default: null },
  to: { type: String, default: null },
})

const { t } = useI18n()
const settingsStore = useSettingsStore()

const GAP_VARIANTS = { ok: 'positive', warning: 'warning', alert: 'danger', credit: 'info' }

const data = ref(null)
const loading = ref(false)
let requestId = 0

async function load() {
  const id = ++requestId
  loading.value = true
  try {
    const params = {}
    if (props.from) params.from = props.from
    if (props.to) params.to = props.to
    const { data: result } = await utilitiesAPI.getCostAnalysis(props.utilityId, params)
    if (id === requestId) data.value = result
  } catch (err) {
    console.error('Failed to load cost analysis:', err)
    if (id === requestId) data.value = null
  } finally {
    if (id === requestId) loading.value = false
  }
}

watch(() => [props.utilityId, props.from, props.to], load, { immediate: true })

function fmtNum(value) {
  return formatNumber(value, settingsStore.formatSettings, { maximumFractionDigits: 1 })
}

function fmtDiff(value) {
  return formatDiff(Math.round(value * 10) / 10, settingsStore.formatSettings)
}

function fmtPrice(value, options) {
  return formatCurrency(value, settingsStore.formatSettings, options)
}

function formatChange(change) {
  return formatNumber(change, settingsStore.formatSettings, { style: 'percent', maximumFractionDigits: 1, signDisplay: 'exceptZero' })
}

function changeVariant(change) {
  if (Math.abs(change) < 0.005) return 'neutral'
  return change > 0 ? 'warning' : 'positive'
}

const tiles = computed(() => {
  const { current, previous } = data.value
  const list = []
  if (props.isMetered) {
    const unitPrice = (v) => v == null ? '–' : t('utilities.costAnalysis.perUnit', { price: fmtPrice(v, { maximumFractionDigits: 4 }), unit: props.consumptionUnit })
    const total = current.bill_count
    list.push({
      id: 'unit',
      label: t('utilities.costAnalysis.unitCost'),
      value: unitPrice(current.unit_cost),
      change: data.value.unit_cost_change,
      previous: unitPrice(previous?.unit_cost),
      hint: current.unit_cost == null
        ? t('utilities.costAnalysis.unitCostMissing')
        : t(`utilities.costAnalysis.${total === 1 ? 'measuredBills_one' : 'measuredBills_other'}`, { n: current.measured_bill_count, total }),
    })
  }
  const perDay = (v) => v == null ? '–' : t('utilities.costAnalysis.perDay', { price: fmtPrice(v) })
  list.push({
    id: 'day',
    label: t('utilities.costAnalysis.costPerDay'),
    value: perDay(current.cost_per_day),
    change: data.value.cost_per_day_change,
    previous: perDay(previous?.cost_per_day),
    hint: t(`utilities.costAnalysis.${current.days === 1 ? 'coveredDays_one' : 'coveredDays_other'}`, { n: current.days }),
  })
  return list
})

function anomalyText(a) {
  const params = {
    number: a.bill_number,
    date: formatDate(a.period_end, settingsStore.dateSettings),
    count: a.count,
    previous: formatNumber(a.previous, settingsStore.formatSettings),
    current: formatNumber(a.current, settingsStore.formatSettings),
    unit: props.consumptionUnit,
    register: a.register?.toUpperCase(),
  }
  const key = a.kind === 'reading_decrease' && a.register ? 'textRegister' : 'text'
  return t(`utilities.costAnalysis.anomaly.${a.kind}.${key}`, params)
}

defineExpose({ load })
</script>
