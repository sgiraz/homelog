<template>
  <Card class="p-4">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        Confronto Letture
      </h3>
      <button
        @click="loadComparisons"
        class="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400"
        :disabled="loading"
      >
        {{ loading ? 'Caricamento...' : 'Aggiorna' }}
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
      <p class="text-gray-700 dark:text-gray-300 font-medium">Nessun confronto disponibile</p>
      <div class="text-sm mt-3 text-left max-w-xs mx-auto space-y-2">
        <p class="text-gray-500 dark:text-gray-400">Per abilitare il confronto:</p>
        <ol class="list-decimal list-inside text-gray-500 dark:text-gray-400 space-y-1">
          <li>Carica le bollette tramite <strong>PDF</strong> per estrarre le letture del fornitore</li>
          <li>Oppure inserisci manualmente le <strong>letture del fornitore</strong> quando aggiungi una bolletta</li>
          <li>Inserisci le tue <strong>autoletture</strong> mensili</li>
        </ol>
      </div>
      <p class="text-xs text-gray-400 mt-3">
        Debug: {{ debugInfo }}
      </p>
    </div>

    <div v-else class="space-y-6">
      <!-- Cumulative Summary Alert (only for overcharges) -->
      <div
        v-if="consumptionSummary?.has_cumulative_alert"
        :class="[
          'p-4 rounded-lg border-2',
          consumptionSummary.cumulative_alert_level === 'alert'
            ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20'
            : 'border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-900/20'
        ]"
      >
        <div class="flex items-start gap-3">
          <svg class="w-6 h-6 flex-shrink-0" :class="consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-500' : 'text-yellow-500'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <h4 :class="['font-semibold', consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-700 dark:text-red-300' : 'text-yellow-700 dark:text-yellow-300']">
              {{ consumptionSummary.cumulative_alert_level === 'alert' ? 'Sovrafatturazione Rilevata!' : 'Attenzione: Differenza Accumulata' }}
            </h4>
            <p :class="['text-sm mt-1', consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-600 dark:text-red-400' : 'text-yellow-600 dark:text-yellow-400']">
              {{ consumptionSummary.cumulative_message }}
            </p>
          </div>
        </div>
      </div>

      <!-- Informational message (provider charged less - not a problem) -->
      <div
        v-else-if="consumptionSummary?.cumulative_message && !consumptionSummary?.has_cumulative_alert"
        class="p-3 rounded-lg border border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20"
      >
        <div class="flex items-start gap-2">
          <svg class="w-5 h-5 flex-shrink-0 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="text-sm text-blue-700 dark:text-blue-300">
            {{ consumptionSummary.cumulative_message }}
          </p>
        </div>
      </div>

      <!-- Consumption Summary Card -->
      <div v-if="consumptionSummary && (consumptionSummary.total_user > 0 || consumptionSummary.total_provider > 0)" class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <h4 class="font-semibold text-gray-900 dark:text-white mb-3">Riepilogo Consumi Cumulativi</h4>

        <div v-if="utilityType === 'electricity'" class="space-y-3">
          <div class="grid grid-cols-4 gap-3 text-sm">
            <div></div>
            <div class="text-center font-medium text-red-600 dark:text-red-400">F1</div>
            <div class="text-center font-medium text-yellow-600 dark:text-yellow-400">F2</div>
            <div class="text-center font-medium text-green-600 dark:text-green-400">F3</div>

            <div class="text-gray-600 dark:text-gray-400">Autoletture</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_user_f1) }}</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_user_f2) }}</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_user_f3) }}</div>

            <div class="text-gray-600 dark:text-gray-400">Fornitore</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_provider_f1) }}</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_provider_f2) }}</div>
            <div class="text-center text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_provider_f3) }}</div>

            <div class="text-gray-600 dark:text-gray-400 font-medium">Differenza</div>
            <div :class="['text-center font-medium', getDiffClass(consumptionSummary.cumulative_difference_f1)]">
              {{ formatDiff(consumptionSummary.cumulative_difference_f1) }}
            </div>
            <div :class="['text-center font-medium', getDiffClass(consumptionSummary.cumulative_difference_f2)]">
              {{ formatDiff(consumptionSummary.cumulative_difference_f2) }}
            </div>
            <div :class="['text-center font-medium', getDiffClass(consumptionSummary.cumulative_difference_f3)]">
              {{ formatDiff(consumptionSummary.cumulative_difference_f3) }}
            </div>
          </div>

          <!-- Total summary for electricity -->
          <div :class="[
            'p-3 rounded text-center',
            consumptionSummary.cumulative_difference > 1 ? 'bg-red-100 dark:bg-red-900/30' : 'bg-white dark:bg-gray-700'
          ]">
            <div class="flex justify-center items-center gap-4 text-sm">
              <span class="text-gray-600 dark:text-gray-400">Totale:</span>
              <span class="text-gray-900 dark:text-white">Autoletture: <strong>{{ formatNumber(consumptionSummary.total_user) }}</strong> kWh</span>
              <span class="text-gray-900 dark:text-white">Fornitore: <strong>{{ formatNumber(consumptionSummary.total_provider) }}</strong> kWh</span>
              <span :class="['font-semibold', getDiffClass(consumptionSummary.cumulative_difference)]">
                {{ consumptionSummary.cumulative_difference > 0 ? 'Sovrafatturato: ' : 'Diff: ' }}{{ formatDiff(consumptionSummary.cumulative_difference) }} kWh
              </span>
            </div>
          </div>
        </div>

        <div v-else class="grid grid-cols-3 gap-4 text-sm">
          <div class="text-center p-3 bg-white dark:bg-gray-700 rounded">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">Consumi Autoletture</p>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_user) }} {{ getUnit() }}</p>
          </div>
          <div class="text-center p-3 bg-white dark:bg-gray-700 rounded">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">Consumi Fatturati</p>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatNumber(consumptionSummary.total_provider) }} {{ getUnit() }}</p>
          </div>
          <div :class="[
            'text-center p-3 rounded',
            consumptionSummary.cumulative_difference > 1 ? 'bg-red-100 dark:bg-red-900/30' : 'bg-white dark:bg-gray-700'
          ]">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">
              {{ consumptionSummary.cumulative_difference > 0 ? 'Sovrafatturato' : 'Differenza' }}
            </p>
            <p :class="['text-lg font-semibold', getDiffClass(consumptionSummary.cumulative_difference)]">
              {{ formatDiff(consumptionSummary.cumulative_difference) }} {{ getUnit() }}
            </p>
            <p v-if="consumptionSummary.cumulative_difference > 1" class="text-xs text-red-600 dark:text-red-400 mt-1">
              Stai pagando di più!
            </p>
          </div>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
          Periodo: {{ formatDate(consumptionSummary.first_period) }} - {{ formatDate(consumptionSummary.last_period) }}
        </p>
      </div>

      <!-- Consumption Periods Table (like the CSV) -->
      <div v-if="consumptionPeriods && consumptionPeriods.length > 0" class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <h4 class="font-semibold text-gray-900 dark:text-white mb-3">Dettaglio Consumi per Periodo</h4>

        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-gray-300 dark:border-gray-600">
                <th class="text-left py-2 px-1 text-gray-600 dark:text-gray-400">Periodo</th>
                <th class="text-right py-2 px-1 text-gray-600 dark:text-gray-400">Consumo Effettivo</th>
                <th class="text-right py-2 px-1 text-gray-600 dark:text-gray-400">Consumo Fatturato</th>
                <th class="text-right py-2 px-1 text-gray-600 dark:text-gray-400">Differenza</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(period, idx) in consumptionPeriods"
                :key="idx"
                class="border-b border-gray-200 dark:border-gray-700"
              >
                <td class="py-2 px-1 text-gray-900 dark:text-white">
                  {{ formatDate(period.period_start) }} - {{ formatDate(period.period_end) }}
                </td>
                <td class="py-2 px-1 text-right text-gray-900 dark:text-white">
                  {{ period.user_consumption != null ? formatNumber(period.user_consumption) : '-' }}
                </td>
                <td class="py-2 px-1 text-right text-gray-900 dark:text-white">
                  {{ period.provider_consumption != null ? formatNumber(period.provider_consumption) : '-' }}
                </td>
                <td :class="['py-2 px-1 text-right font-medium', getDiffClass(period.difference)]">
                  {{ period.difference != null ? formatDiff(period.difference) : '-' }}
                </td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="border-t-2 border-gray-400 dark:border-gray-500 font-semibold">
                <td class="py-2 px-1 text-gray-900 dark:text-white">TOTALE</td>
                <td class="py-2 px-1 text-right text-gray-900 dark:text-white">
                  {{ formatNumber(consumptionSummary?.total_user || 0) }} {{ getUnit() }}
                </td>
                <td class="py-2 px-1 text-right text-gray-900 dark:text-white">
                  {{ formatNumber(consumptionSummary?.total_provider || 0) }} {{ getUnit() }}
                </td>
                <td :class="['py-2 px-1 text-right', getDiffClass(consumptionSummary?.cumulative_difference)]">
                  {{ formatDiff(consumptionSummary?.cumulative_difference || 0) }} {{ getUnit() }}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <!-- Individual Comparisons -->
      <div class="space-y-4">
        <h4 class="font-semibold text-gray-900 dark:text-white">Confronto per Bolletta</h4>
        <div
          v-for="comparison in comparisons"
          :key="comparison.bill_id"
          :class="[
            'p-4 rounded-lg border-2',
            getStatusClasses(comparison.status)
          ]"
        >
          <!-- Header with status icon -->
          <div class="flex items-start justify-between mb-3">
            <div>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                Bolletta {{ comparison.bill_number || '#' + comparison.bill_id }}
              </p>
              <p class="text-xs text-gray-400 dark:text-gray-500">
                Periodo fino al {{ formatDate(comparison.period_end) }}
              </p>
            </div>
            <div :class="['flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium', getStatusBadgeClasses(comparison.status)]">
              <component :is="getStatusIcon(comparison.status)" class="w-4 h-4" />
              <span>{{ getStatusLabel(comparison.status) }}</span>
            </div>
          </div>

          <!-- Days difference and effective threshold info -->
          <div v-if="comparison.days_difference > 0" class="mb-3 text-xs bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 px-3 py-2 rounded">
            <span class="font-medium">{{ comparison.days_difference }} giorni</span> di differenza tra le letture.
            Soglia effettiva: <span class="font-medium">{{ formatNumber(comparison.effective_threshold) }} {{ getUnit() }}</span>
          </div>

          <!-- Alert message if any -->
          <div v-if="comparison.alert_message && comparison.status !== 'ok'" class="mb-3 text-sm">
            <span :class="comparison.status === 'alert' ? 'text-red-600 dark:text-red-400' : 'text-yellow-600 dark:text-yellow-400'">
              {{ comparison.alert_message }}
            </span>
          </div>

          <!-- Electricity comparison (F1/F2/F3) -->
          <div v-if="utilityType === 'electricity'" class="grid grid-cols-3 gap-3">
            <ReadingBandComparison
              v-for="band in ['F1', 'F2', 'F3']"
              :key="band"
              :band="band"
              :providerValue="comparison['provider_' + band.toLowerCase()]"
              :userValue="comparison['user_' + band.toLowerCase()]"
              :difference="comparison['difference_' + band.toLowerCase()]"
              :threshold="comparison.effective_threshold"
              unit="kWh"
            />
          </div>

          <!-- Gas/Water comparison (single value) -->
          <div v-else class="grid grid-cols-2 gap-4">
            <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">Lettura Fornitore</p>
              <p class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ comparison.provider_reading != null ? formatNumber(comparison.provider_reading) : '-' }}
                <span class="text-sm font-normal text-gray-500">{{ getUnit() }}</span>
              </p>
            </div>
            <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">Tua Autolettura</p>
              <p class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ comparison.user_reading != null ? formatNumber(comparison.user_reading) : '-' }}
                <span class="text-sm font-normal text-gray-500">{{ getUnit() }}</span>
              </p>
            </div>
            <div v-if="comparison.difference != null" class="col-span-2 text-center">
              <span :class="[
                'text-sm font-medium',
                comparison.status === 'alert' ? 'text-red-600' :
                comparison.status === 'warning' ? 'text-yellow-600' : 'text-green-600'
              ]">
                Differenza: {{ comparison.difference > 0 ? '+' : '' }}{{ formatNumber(comparison.difference) }} {{ getUnit() }}
              </span>
            </div>
          </div>

          <!-- Reading type indicator -->
          <div class="mt-3 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span :class="[
              'px-2 py-0.5 rounded',
              comparison.reading_type === 'actual' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
              'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
            ]">
              {{ comparison.reading_type === 'actual' ? 'Lettura Effettiva' : 'Lettura Stimata' }}
            </span>
            <span v-if="comparison.provider_reading_date">
              del {{ formatDate(comparison.provider_reading_date) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue'
import { utilitiesAPI } from '@/api/client'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'

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

const debugInfo = ref('')

// Check if there's any summary data to display (even without individual comparisons)
const hasAnySummaryData = computed(() => {
  if (consumptionSummary.value) {
    return consumptionSummary.value.total_user > 0 || consumptionSummary.value.total_provider > 0
  }
  return consumptionPeriods.value.length > 0
})

async function loadComparisons() {
  loading.value = true
  try {
    const { data } = await utilitiesAPI.compareReadings(props.utilityId, props.baseThreshold, props.thresholdPerDay)
    comparisons.value = data.comparisons || []
    consumptionSummary.value = data.consumption_summary || null
    consumptionPeriods.value = data.consumption_periods || []
    debugInfo.value = `Bollette con letture: ${comparisons.value.length}, Periodi consumo: ${consumptionPeriods.value.length}`
  } catch (err) {
    console.error('Failed to load comparisons:', err)
    debugInfo.value = `Errore: ${err.message}`
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function formatNumber(value) {
  if (value == null) return '-'
  return value.toLocaleString('it-IT', { maximumFractionDigits: 3 })
}

function formatDiff(value) {
  if (value == null) return '-'
  const prefix = value > 0 ? '+' : ''
  return prefix + value.toLocaleString('it-IT', { maximumFractionDigits: 3 })
}

function getDiffClass(value) {
  if (value == null) return 'text-gray-500'
  if (Math.abs(value) < 1) return 'text-green-600 dark:text-green-400' // Negligible difference
  if (value > 0) return 'text-red-600 dark:text-red-400' // Provider charged MORE - problematic!
  return 'text-green-600 dark:text-green-400' // Provider charged less - not a problem
}

function getUnit() {
  switch (props.utilityType) {
    case 'gas': return 'Smc'
    case 'water': return 'mc'
    case 'electricity': return 'kWh'
    default: return ''
  }
}

function getStatusClasses(status) {
  switch (status) {
    case 'alert':
      return 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20'
    case 'warning':
      return 'border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-900/20'
    case 'no_data':
      return 'border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800'
    default:
      return 'border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-900/20'
  }
}

function getStatusBadgeClasses(status) {
  switch (status) {
    case 'alert':
      return 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300'
    case 'warning':
      return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/50 dark:text-yellow-300'
    case 'no_data':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    default:
      return 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300'
  }
}

function getStatusLabel(status) {
  switch (status) {
    case 'alert': return 'Anomalia'
    case 'warning': return 'Attenzione'
    case 'no_data': return 'No dati'
    default: return 'OK'
  }
}

function getStatusIcon(status) {
  // Return SVG components based on status
  const icons = {
    alert: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z' })
        ])
      }
    },
    warning: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    },
    no_data: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    },
    ok: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    }
  }
  return icons[status] || icons.ok
}

onMounted(() => {
  loadComparisons()
})

// Expose refresh method
defineExpose({ loadComparisons })
</script>

<!-- Sub-component for electricity band comparison -->
<script>
const ReadingBandComparison = {
  props: ['band', 'providerValue', 'userValue', 'difference', 'unit', 'threshold'],
  setup(props) {
    const formatNum = (v) => v != null ? v.toLocaleString('it-IT', { maximumFractionDigits: 3 }) : '-'

    const getBandColor = () => {
      switch (props.band) {
        case 'F1': return 'text-red-600 dark:text-red-400'
        case 'F2': return 'text-yellow-600 dark:text-yellow-400'
        case 'F3': return 'text-green-600 dark:text-green-400'
        default: return 'text-gray-600'
      }
    }

    const getDiffColor = () => {
      if (props.difference == null) return 'text-gray-500'
      const abs = Math.abs(props.difference)
      const threshold = props.threshold || 2
      if (abs > threshold * 2) return 'text-red-600'
      if (abs > threshold) return 'text-yellow-600'
      return 'text-green-600'
    }

    return () => h('div', { class: 'text-center p-2 bg-gray-50 dark:bg-gray-800 rounded-lg' }, [
      h('p', { class: ['text-xs font-medium mb-1', getBandColor()] }, props.band),
      h('div', { class: 'text-xs text-gray-500 dark:text-gray-400' }, [
        h('p', {}, ['Fornitore: ', h('span', { class: 'font-medium text-gray-700 dark:text-gray-300' }, formatNum(props.providerValue))]),
        h('p', {}, ['Tu: ', h('span', { class: 'font-medium text-gray-700 dark:text-gray-300' }, formatNum(props.userValue))]),
        props.difference != null && h('p', { class: getDiffColor() },
          (props.difference > 0 ? '+' : '') + formatNum(props.difference) + ' ' + props.unit
        )
      ])
    ])
  }
}
</script>
