<template>
  <div class="space-y-4">
    <!-- Period Filter -->
    <Card class="p-4">
      <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
        <span class="text-sm font-medium text-ink-soft">{{ t('utilities.analysisTab.periodLabel') }}</span>
        <div class="flex gap-2 flex-wrap">
          <button
            v-for="preset in periodPresets"
            :key="preset.id"
            @click="analysisPeriod = preset.id"
            :class="[
              'px-3 py-2 text-sm rounded-lg transition-colors',
              analysisPeriod === preset.id
                ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 font-medium'
                : 'text-ink-soft hover:bg-surface-2'
            ]"
          >
            {{ preset.label }}
          </button>
        </div>
      </div>
      <div v-if="analysisPeriod === 'custom'" class="flex gap-2 mt-3">
        <input v-model="analysisFrom" type="date" class="px-3 py-2.5 border border-line rounded-lg bg-surface text-ink text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <input v-model="analysisTo" type="date" class="px-3 py-2.5 border border-line rounded-lg bg-surface text-ink text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
    </Card>

    <!-- Analysis KPIs -->
    <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
      <Card class="p-4 text-center">
        <div class="text-2xl font-bold text-ink">{{ formatCurrency(analysisData.totalSpent) }}</div>
        <div class="text-xs text-ink-muted mt-1">{{ t('utilities.analysisTab.kpi.totalSpent') }}</div>
      </Card>
      <Card class="p-4 text-center">
        <div class="text-2xl font-bold text-ink">{{ formatConsumption(analysisData.totalConsumption) }}</div>
        <div class="text-xs text-ink-muted mt-1">{{ t('utilities.analysisTab.kpi.totalConsumption', { unit: consumptionUnit }) }}</div>
      </Card>
      <Card class="p-4 text-center col-span-2 sm:col-span-1">
        <div class="text-2xl font-bold text-ink">{{ analysisData.billCount }}</div>
        <div class="text-xs text-ink-muted mt-1">{{ t('utilities.analysisTab.kpi.billCount') }}</div>
      </Card>
    </div>

    <!-- Comparison Section -->
    <div v-if="utility.type !== 'waste'" class="space-y-4">
      <!-- Threshold Settings (collapsible) -->
      <Card class="p-4">
        <button
          @click="showThresholdSettings = !showThresholdSettings"
          class="flex items-center justify-between w-full text-sm font-medium text-ink-soft"
        >
          <span>{{ t('utilities.analysisTab.thresholdSettings') }}</span>
          <svg :class="['w-4 h-4 transition-transform', showThresholdSettings ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <div v-if="showThresholdSettings" class="mt-3 space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm text-ink-soft">{{ t('utilities.analysisTab.baseThreshold') }}</div>
              <div class="text-xs text-gray-400">{{ t('utilities.analysisTab.baseThresholdHint') }}</div>
            </div>
            <div class="flex items-center gap-2">
              <input v-model.number="thresholdValue" type="number" min="0.5" max="50" step="0.5"
                class="w-16 px-2 py-2 text-sm text-center border border-line rounded-lg bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
              <span class="text-xs text-gray-400">{{ consumptionUnit }}</span>
            </div>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm text-ink-soft">{{ t('utilities.analysisTab.perDayThreshold') }}</div>
              <div class="text-xs text-gray-400">{{ t('utilities.analysisTab.perDayThresholdHint') }}</div>
            </div>
            <div class="flex items-center gap-2">
              <input v-model.number="thresholdPerDayValue" type="number" min="0.1" max="10" step="0.1"
                class="w-16 px-2 py-2 text-sm text-center border border-line rounded-lg bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
              <span class="text-xs text-gray-400">{{ t('utilities.analysisTab.perDayUnit', { unit: consumptionUnit }) }}</span>
            </div>
          </div>
          <Button v-if="hasThresholdChanges" size="sm" @click="saveThreshold" :disabled="savingThreshold">
            {{ savingThreshold ? t('utilities.analysisTab.saving') : t('utilities.analysisTab.saveThreshold') }}
          </Button>
        </div>
      </Card>

      <!-- Comparison Card -->
      <ReadingComparisonCard
        :key="comparisonKey"
        :utility-id="utility.id"
        :utility-type="utility.type"
        :base-threshold="utility.comparison_threshold || 2"
        :threshold-per-day="utility.threshold_per_day || 1"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { formatNumber as _formatNumber, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import ReadingComparisonCard from '@/components/utilities/ReadingComparisonCard.vue'

defineOptions({ name: 'AnalysisTab' })

const props = defineProps({
  utility: { type: Object, required: true },
  consumptionUnit: { type: String, default: '' },
})

const emit = defineEmits(['threshold-saved'])

const { t } = useI18n()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const comparisonKey = ref(0)
const showThresholdSettings = ref(false)
const thresholdValue = ref(props.utility.comparison_threshold || 2)
const thresholdPerDayValue = ref(props.utility.threshold_per_day || 1)
const savingThreshold = ref(false)

const analysisPeriod = ref('12m')
const analysisFrom = ref('')
const analysisTo = ref('')

const periodPresets = computed(() => [
  { id: '3m', label: t('utilities.analysisTab.presets.3m'), months: 3 },
  { id: '6m', label: t('utilities.analysisTab.presets.6m'), months: 6 },
  { id: '12m', label: t('utilities.analysisTab.presets.12m'), months: 12 },
  { id: 'all', label: t('utilities.analysisTab.presets.all'), months: 0 },
  { id: 'custom', label: t('utilities.analysisTab.presets.custom'), months: 0 },
])

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatConsumption(value) {
  if (value == null || value === 0) return '0'
  return _formatNumber(parseFloat(value), settingsStore.formatSettings)
}

const hasThresholdChanges = computed(() => {
  return thresholdValue.value !== (props.utility.comparison_threshold || 2) ||
         thresholdPerDayValue.value !== (props.utility.threshold_per_day || 1)
})

const analysisData = computed(() => {
  const bills = props.utility.bills || []
  let filtered = bills

  if (analysisPeriod.value !== 'all') {
    let from, to
    if (analysisPeriod.value === 'custom') {
      from = analysisFrom.value ? new Date(analysisFrom.value) : null
      to = analysisTo.value ? new Date(analysisTo.value) : null
    } else {
      const preset = periodPresets.value.find(p => p.id === analysisPeriod.value)
      if (preset?.months) {
        to = new Date()
        from = new Date()
        from.setMonth(from.getMonth() - preset.months)
      }
    }
    if (from) filtered = filtered.filter(b => new Date(b.period_end) >= from)
    if (to) filtered = filtered.filter(b => new Date(b.period_start) <= to)
  }

  return {
    totalSpent: filtered.reduce((s, b) => s + (b.amount_total || 0), 0),
    totalConsumption: filtered.reduce((s, b) => s + (b.consumption_total || 0), 0),
    billCount: filtered.length,
  }
})

async function saveThreshold() {
  savingThreshold.value = true
  try {
    await utilitiesStore.updateUtility(props.utility.id, {
      comparison_threshold: thresholdValue.value,
      threshold_per_day: thresholdPerDayValue.value
    })
    emit('threshold-saved', {
      comparison_threshold: thresholdValue.value,
      threshold_per_day: thresholdPerDayValue.value
    })
    comparisonKey.value++
  } catch (err) {
    console.error('Error saving threshold:', err)
  } finally {
    savingThreshold.value = false
  }
}

function refreshComparison() {
  comparisonKey.value++
}

defineExpose({ refreshComparison })
</script>
