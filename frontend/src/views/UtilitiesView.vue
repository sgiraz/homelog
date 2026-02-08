<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Utilities</h2>
        <p class="text-gray-600 dark:text-gray-400 text-sm mt-1">
          {{ currentProperty?.name || 'Seleziona una proprieta' }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="showTemplatesManager = true"
          class="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300
                 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          title="Gestisci template estrazione"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </button>
        <Button @click="showAddUtility = true">
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Aggiungi
        </Button>
      </div>
    </div>

    <!-- Alerts Section -->
    <div v-if="utilitiesStore.dueSoonBillsCount > 0"
         class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-xl p-4">
      <div class="flex items-center gap-3 text-yellow-800 dark:text-yellow-300">
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <span class="font-medium">{{ utilitiesStore.dueSoonBillsCount }} bollette in scadenza nei prossimi 3 giorni</span>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="utilitiesStore.loading" class="text-center py-12 text-gray-600 dark:text-gray-400">
      Caricamento...
    </div>

    <!-- Empty State -->
    <div v-else-if="utilitiesStore.utilities.length === 0" class="text-center py-12">
      <Card class="p-8">
        <svg class="w-16 h-16 mx-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <h3 class="mt-4 text-lg font-medium text-gray-900 dark:text-white">Nessuna utenza configurata</h3>
        <p class="mt-2 text-gray-500 dark:text-gray-400">Aggiungi le tue utenze per monitorare consumi e bollette</p>
        <Button class="mt-4" @click="showAddUtility = true">
          Aggiungi la prima utenza
        </Button>
      </Card>
    </div>

    <!-- Utilities Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Card
        v-for="utility in utilitiesStore.utilities"
        :key="utility.id"
        class="p-6 hover:shadow-lg transition-all cursor-pointer backdrop-blur-xl bg-opacity-70"
      >
        <!-- Header with Icon and Info -->
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div :class="[
              'p-3 rounded-xl border',
              getUtilityColorClasses(utility.type)
            ]">
              <component :is="getUtilityIcon(utility.type)" class="w-6 h-6" />
            </div>
            <div>
              <h3 class="font-bold text-gray-900 dark:text-white">{{ getUtilityName(utility.type) }}</h3>
              <p class="text-sm text-gray-600 dark:text-gray-400">{{ utility.provider }}</p>
            </div>
          </div>
        </div>

        <!-- Stats -->
        <div class="space-y-2 mb-4">
          <!-- Consumption -->
          <div class="flex justify-between">
            <span class="text-gray-600 dark:text-gray-400">Consumo:</span>
            <span class="text-gray-900 dark:text-white font-medium">
              {{ getLastConsumption(utility) }}
            </span>
          </div>

          <!-- Last Bill -->
          <div class="flex justify-between">
            <span class="text-gray-600 dark:text-gray-400">Ultima bolletta:</span>
            <span class="font-medium text-gray-900 dark:text-white">
              {{ getLastBillAmount(utility) }}
            </span>
          </div>

          <!-- Due Date -->
          <div class="flex justify-between">
            <span class="text-gray-600 dark:text-gray-400">Scadenza:</span>
            <span class="text-gray-900 dark:text-white">
              {{ getLastBillDueDate(utility) }}
            </span>
          </div>

          <!-- Last Reading -->
          <div v-if="utility.readings?.length > 0" class="flex justify-between">
            <span class="text-gray-600 dark:text-gray-400">Ultima lettura:</span>
            <span class="text-gray-900 dark:text-white">
              {{ formatDate(utility.readings[0].reading_date) }}
            </span>
          </div>
        </div>

        <!-- Alert for meter reading -->
        <div v-if="shouldShowReadingAlert(utility)"
             class="mb-4 p-3 bg-orange-50 dark:bg-orange-900/20 rounded-lg text-sm text-orange-600 dark:text-orange-400 border border-orange-200 dark:border-orange-800">
          {{ getReadingAlertMessage(utility) }}
        </div>

        <!-- Action Buttons -->
        <div class="flex gap-2">
          <button
            @click.stop="openUtilityDetail(utility)"
            class="flex-1 px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-lg
                   hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-gray-700 dark:text-gray-300"
          >
            Dettagli
          </button>
          <button
            @click.stop="openBills(utility)"
            class="flex-1 px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-lg
                   hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-gray-700 dark:text-gray-300"
          >
            Bollette
          </button>
          <button
            @click.stop="openAddReading(utility)"
            class="flex-1 px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-lg
                   hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-gray-700 dark:text-gray-300"
          >
            Lettura
          </button>
        </div>
      </Card>
    </div>

    <!-- Add Utility Modal -->
    <AddUtilityModal
      v-if="showAddUtility"
      @close="showAddUtility = false"
      @created="onUtilityCreated"
    />

    <!-- Utility Detail Modal -->
    <UtilityDetailModal
      v-if="showUtilityDetail && selectedUtility"
      :utility="selectedUtility"
      :initial-tab="initialTab"
      @close="closeUtilityDetail"
      @updated="onUtilityUpdated"
    />

    <!-- Add Reading Modal (quick access) -->
    <AddReadingModal
      v-if="showAddReading && readingUtility"
      :utility="readingUtility"
      @close="showAddReading = false"
      @saved="onReadingSaved"
    />

    <!-- Templates Manager Modal -->
    <TemplatesManager
      v-if="showTemplatesManager"
      @close="showTemplatesManager = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddUtilityModal from '@/components/utilities/AddUtilityModal.vue'
import UtilityDetailModal from '@/components/utilities/UtilityDetailModal.vue'
import AddReadingModal from '@/components/utilities/AddReadingModal.vue'
import TemplatesManager from '@/components/utilities/TemplatesManager.vue'

const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const showAddUtility = ref(false)
const showUtilityDetail = ref(false)
const showAddReading = ref(false)
const showTemplatesManager = ref(false)
const selectedUtility = ref(null)
const readingUtility = ref(null)
const initialTab = ref('bills')
const currentProperty = ref(null)

// Utility type icons
const ElectricityIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M13 10V3L4 14h7v7l9-11h-7z' })
    ])
  }
}

const GasIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z' }),
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9.879 16.121A3 3 0 1012.015 11L11 14H9c0 .768.293 1.536.879 2.121z' })
    ])
  }
}

const WaterIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707' }),
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 9a4 4 0 100 8 4 4 0 000-8z' })
    ])
  }
}

const WasteIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16' })
    ])
  }
}

function getUtilityIcon(type) {
  const icons = {
    electricity: ElectricityIcon,
    gas: GasIcon,
    water: WaterIcon,
    waste: WasteIcon
  }
  return icons[type] || ElectricityIcon
}

function getUtilityColorClasses(type) {
  const classes = {
    electricity: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800 text-yellow-500',
    gas: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800 text-orange-500',
    water: 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800 text-cyan-500',
    waste: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-500'
  }
  return classes[type] || classes.electricity
}

function getUtilityName(type) {
  const names = {
    electricity: 'Luce',
    gas: 'Gas',
    water: 'Acqua',
    waste: 'Rifiuti'
  }
  return names[type] || type
}

function getConsumptionUnit(type) {
  const units = {
    electricity: 'kWh',
    gas: 'Smc',
    water: 'mc',
    waste: 'mq'
  }
  return units[type] || ''
}

function getLastConsumption(utility) {
  if (utility.bills?.length > 0) {
    const bill = utility.bills[0]
    const num = parseFloat(bill.consumption_total)
    const truncated = Math.trunc(num * 1000) / 1000
    const value = bill.consumption_total != null ? truncated.toFixed(3).replace(/\.?0+$/, '') : bill.consumption_total
    return `${value} ${getConsumptionUnit(utility.type)}`
  }
  return '-'
}

function getLastBillAmount(utility) {
  if (utility.bills?.length > 0) {
    return formatCurrency(utility.bills[0].amount_total)
  }
  return '-'
}

function getLastBillDueDate(utility) {
  if (utility.bills?.length > 0) {
    return formatDate(utility.bills[0].due_date)
  }
  return '-'
}

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(value || 0)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function shouldShowReadingAlert(utility) {
  if (!utility.readings || utility.readings.length === 0) return false

  const lastReading = new Date(utility.readings[0].reading_date)
  const now = new Date()
  const daysSinceReading = Math.floor((now - lastReading) / (1000 * 60 * 60 * 24))

  // Show alert if more than 25 days since last reading (approaching monthly reading period)
  return daysSinceReading >= 25
}

function getReadingAlertMessage(utility) {
  if (!utility.readings || utility.readings.length === 0) return ''

  const lastReading = new Date(utility.readings[0].reading_date)
  const now = new Date()
  const daysSinceReading = Math.floor((now - lastReading) / (1000 * 60 * 60 * 24))

  if (daysSinceReading >= 30) {
    return 'Autolettura consigliata'
  }
  return `Autolettura consigliata tra ${30 - daysSinceReading} giorni`
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      currentProperty.value = data.find(p => p.is_current) || data[0]
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function openUtilityDetail(utility) {
  selectedUtility.value = utility
  initialTab.value = 'bills'
  await utilitiesStore.fetchUtility(utility.id)
  selectedUtility.value = utilitiesStore.selectedUtility
  showUtilityDetail.value = true
}

async function openBills(utility) {
  selectedUtility.value = utility
  initialTab.value = 'bills'
  await utilitiesStore.fetchUtility(utility.id)
  selectedUtility.value = utilitiesStore.selectedUtility
  showUtilityDetail.value = true
}

function openAddReading(utility) {
  readingUtility.value = utility
  showAddReading.value = true
}

function closeUtilityDetail() {
  showUtilityDetail.value = false
  selectedUtility.value = null
  utilitiesStore.clearSelectedUtility()
}

function onUtilityCreated() {
  showAddUtility.value = false
  utilitiesStore.fetchUtilities()
}

function onUtilityUpdated() {
  utilitiesStore.fetchUtilities()
}

function onReadingSaved() {
  showAddReading.value = false
  readingUtility.value = null
  utilitiesStore.fetchUtilities()
}

onMounted(() => {
  utilitiesStore.fetchUtilities()
  fetchCurrentProperty()
})
</script>
