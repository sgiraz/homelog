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

    <div v-else-if="comparisons.length === 0" class="text-center py-6">
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

    <div v-else class="space-y-4">
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
            :threshold="threshold"
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
  </Card>
</template>

<script setup>
import { ref, onMounted, h } from 'vue'
import { utilitiesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'

const props = defineProps({
  utilityId: {
    type: Number,
    required: true
  },
  utilityType: {
    type: String,
    required: true
  },
  threshold: {
    type: Number,
    default: 5
  }
})

const loading = ref(false)
const comparisons = ref([])

const debugInfo = ref('')

async function loadComparisons() {
  loading.value = true
  try {
    const { data } = await utilitiesAPI.compareReadings(props.utilityId, props.threshold)
    comparisons.value = data.comparisons || []
    debugInfo.value = `Bollette con letture: ${comparisons.value.length}, Soglia: ${props.threshold}%`
  } catch (err) {
    console.error('Failed to load comparisons:', err)
    debugInfo.value = `Errore: ${err.message}`
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('it-IT')
}

function formatNumber(value) {
  if (value == null) return '-'
  return value.toLocaleString('it-IT', { maximumFractionDigits: 2 })
}

function getUnit() {
  switch (props.utilityType) {
    case 'gas': return 'mc'
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
    const formatNum = (v) => v != null ? v.toLocaleString('it-IT', { maximumFractionDigits: 2 }) : '-'

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
      const threshold = props.threshold || 5
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
