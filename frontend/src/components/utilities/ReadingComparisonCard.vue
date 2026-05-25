<template>
  <Card class="p-4">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-ink">
        {{ t('utilities.comparisonCard.title') }}
      </h3>
      <button
        @click="loadComparisons"
        class="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400"
        :disabled="loading"
      >
        {{ loading ? t('utilities.comparisonCard.loading') : t('utilities.comparisonCard.refresh') }}
      </button>
    </div>

    <div v-if="loading" class="flex justify-center py-8">
      <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

    <div v-else-if="comparisons.length === 0 && !hasAnySummaryData" class="text-center py-6">
      <svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
      <p class="text-ink-soft font-medium">{{ t('utilities.comparisonCard.empty') }}</p>
      <div class="text-sm mt-3 text-left max-w-xs mx-auto space-y-2">
        <p class="text-ink-muted">{{ t('utilities.comparisonCard.emptyHint') }}</p>
        <ol class="list-decimal list-inside text-ink-muted space-y-1">
          <li>{{ t('utilities.comparisonCard.step1') }} <strong>{{ t('utilities.comparisonCard.step1bold') }}</strong> {{ t('utilities.comparisonCard.step1cont') }}</li>
          <li>{{ t('utilities.comparisonCard.step2') }} <strong>{{ t('utilities.comparisonCard.step2bold') }}</strong> {{ t('utilities.comparisonCard.step2cont') }}</li>
          <li>{{ t('utilities.comparisonCard.step3') }} <strong>{{ t('utilities.comparisonCard.step3bold') }}</strong> {{ t('utilities.comparisonCard.step3cont') }}</li>
        </ol>
      </div>
    </div>

    <div v-else class="space-y-6">
      <ConsumptionSummary
        :consumptionSummary="consumptionSummary"
        :utilityType="utilityType"
        :fmtNum="fmtNum"
        :fmtDiff="fmtDiff"
        :formatDate="formatDate"
        :getDiffClass="getDiffClass"
        :getUnit="getUnit"
      />

      <ConsumptionPeriodsTable
        :consumptionAnalysis="consumptionPeriods"
        :utilityType="utilityType"
        :consumptionSummary="consumptionSummary"
        :fmtNum="fmtNum"
        :fmtDiff="fmtDiff"
        :formatDate="formatDate"
        :formatPeriodCompact="formatPeriodCompact"
        :getDiffClass="getDiffClass"
        :getUnit="getUnit"
      />

      <ComparisonAccordion
        :comparisons="comparisons"
        :expandedCards="expandedCards"
        :utilityType="utilityType"
        :fmtNum="fmtNum"
        :fmtDiff="fmtDiff"
        :formatDate="formatDate"
        :getUnit="getUnit"
        @toggle-card="toggleCard"
      />
    </div>
  </Card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { utilitiesAPI } from '@/api/client'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate, formatPeriodCompact as _formatPeriodCompact, formatNumber as _fmtNum, formatDiff as _fmtDiff } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import ConsumptionSummary from './ConsumptionSummary.vue'
import ConsumptionPeriodsTable from './ConsumptionPeriodsTable.vue'
import ComparisonAccordion from './ComparisonAccordion.vue'

defineOptions({ name: 'ReadingComparisonCard' })

const { t } = useI18n()
const settingsStore = useSettingsStore()

const props = defineProps({
  utilityId: {
    type: Number,
    required: true
  },
  utilityType: {
    type: String,
    required: true
  },
  baseThreshold: {
    type: Number,
    default: 2
  },
  thresholdPerDay: {
    type: Number,
    default: 1
  }
})

const loading = ref(false)
const comparisons = ref([])
const consumptionSummary = ref(null)
const consumptionPeriods = ref([])

// Accordion state — expanded card IDs
const expandedCards = ref(new Set())

// Check if there's any summary data to display (even without individual comparisons)
const hasAnySummaryData = computed(() => {
  if (consumptionSummary.value) {
    return consumptionSummary.value.total_user > 0 || consumptionSummary.value.total_provider > 0
  }
  return consumptionPeriods.value.length > 0
})

function toggleCard(billId) {
  if (expandedCards.value.has(billId)) {
    expandedCards.value.delete(billId)
  } else {
    expandedCards.value.add(billId)
  }
  // Force reactivity
  expandedCards.value = new Set(expandedCards.value)
}

async function loadComparisons() {
  loading.value = true
  try {
    const { data } = await utilitiesAPI.compareReadings(props.utilityId, props.baseThreshold, props.thresholdPerDay)
    comparisons.value = data.comparisons || []
    consumptionSummary.value = data.consumption_summary || null
    consumptionPeriods.value = data.consumption_periods || []

    // Auto-expand anomalies and warnings, collapse OK
    expandedCards.value = new Set()
    for (const c of comparisons.value) {
      if (c.status === 'alert' || c.status === 'warning') {
        expandedCards.value.add(c.bill_id)
      }
    }
  } catch (err) {
    console.error('Failed to load comparisons:', err)
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function formatPeriodCompact(start, end) {
  return _formatPeriodCompact(start, end, settingsStore.formatSettings)
}

function fmtNum(value) {
  return _fmtNum(value, settingsStore.formatSettings)
}

function fmtDiff(value) {
  return _fmtDiff(value, settingsStore.formatSettings)
}

function getDiffClass(value) {
  if (value == null) return 'text-ink-muted'
  if (Math.abs(value) < 1) return 'text-green-600 dark:text-green-400'
  if (value > 0) return 'text-red-600 dark:text-red-400'
  return 'text-green-600 dark:text-green-400'
}

function getUnit() {
  switch (props.utilityType) {
    case 'gas': return 'Smc'
    case 'water': return 'mc'
    case 'electricity': return 'kWh'
    default: return ''
  }
}

onMounted(() => {
  loadComparisons()
})

// Expose refresh method
defineExpose({ loadComparisons })
</script>
