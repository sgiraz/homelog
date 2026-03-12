<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div>
        <h1 class="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white">Servizi</h1>
        <p class="text-gray-600 dark:text-gray-400 text-sm mt-1">
          Utenze, abbonamenti e servizi ricorrenti
        </p>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <!-- Property selector -->
        <select
          v-if="properties.length > 0"
          v-model="selectedPropertyId"
          @change="onPropertyChange"
          class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option v-for="p in properties" :key="p.id" :value="p.id">
            {{ p.name }}
          </option>
        </select>

        <button
          @click="showTemplatesManager = true"
          class="p-2.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300
                 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          title="Gestisci template estrazione"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </button>
        <Button v-if="authStore.isAdmin" @click="showAddUtility = true">
          <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <span class="hidden sm:inline">Aggiungi</span>
        </Button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="utilitiesStore.loading" class="text-center py-12 text-gray-600 dark:text-gray-400">
      Caricamento...
    </div>

    <!-- Empty State -->
    <div v-else-if="utilitiesStore.utilities.length === 0" class="text-center py-12">
      <Card class="p-8">
        <svg class="w-16 h-16 mx-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <h3 class="mt-4 text-lg font-medium text-gray-900 dark:text-white">Nessun servizio configurato</h3>
        <p class="mt-2 text-gray-500 dark:text-gray-400">Aggiungi utenze, abbonamenti o servizi ricorrenti</p>
        <Button v-if="authStore.isAdmin" class="mt-4" @click="showAddUtility = true">
          Aggiungi il primo servizio
        </Button>
      </Card>
    </div>

    <template v-else>
      <!-- KPI Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <Card class="p-4">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Spesa mensile media</div>
          <div class="text-xl font-bold text-gray-900 dark:text-white">{{ formatCurrency(dashboardKPIs.avgMonthly) }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Da pagare</div>
          <div class="text-xl font-bold" :class="dashboardKPIs.unpaidTotal > 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white'">
            {{ formatCurrency(dashboardKPIs.unpaidTotal) }}
          </div>
        </Card>
        <Card class="p-4">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Bollette non pagate</div>
          <div class="text-xl font-bold text-gray-900 dark:text-white">{{ dashboardKPIs.unpaidCount }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Servizi attivi</div>
          <div class="text-xl font-bold text-gray-900 dark:text-white">{{ utilitiesStore.utilities.length }}</div>
        </Card>
      </div>

      <!-- Alerts -->
      <div v-if="alerts.length > 0" class="space-y-2">
        <div
          v-for="(alert, i) in alerts"
          :key="i"
          :class="[
            'flex items-center gap-3 p-3 rounded-xl border text-sm',
            alert.type === 'warning'
              ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-300'
              : 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800 text-orange-800 dark:text-orange-300'
          ]"
        >
          <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <span class="flex-1">{{ alert.message }}</span>
          <button
            v-if="alert.utilityId"
            @click="$router.push(`/utilities/${alert.utilityId}`)"
            class="text-xs font-medium underline flex-shrink-0"
          >
            Vai
          </button>
        </div>
      </div>

      <!-- Utility Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <router-link
          v-for="utility in sortedUtilities"
          :key="utility.id"
          :to="`/utilities/${utility.id}`"
          class="block"
        >
          <Card class="p-5 hover:shadow-lg hover:border-blue-200 dark:hover:border-blue-800 transition-all h-full">
            <!-- Header -->
            <div class="flex items-center gap-3 mb-3">
              <div :class="['p-2.5 rounded-xl border', getUtilityColorClasses(utility.type)]">
                <component :is="getUtilityIcon(utility.type)" class="w-6 h-6" />
              </div>
              <div class="flex-1 min-w-0">
                <h3 class="font-bold text-gray-900 dark:text-white truncate">{{ getUtilityName(utility.type) }}</h3>
                <p class="text-sm text-gray-500 dark:text-gray-400 truncate">{{ utility.provider }}</p>
              </div>
              <svg class="w-5 h-5 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>

            <!-- Stats -->
            <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
              <div class="text-gray-500 dark:text-gray-400">{{ isMeteredType(utility.type) ? 'Ultima bolletta' : 'Ultimo importo' }}</div>
              <div class="text-right font-medium text-gray-900 dark:text-white">{{ getLastBillAmount(utility) }}</div>
              <div class="text-gray-500 dark:text-gray-400">Scadenza</div>
              <div class="text-right text-gray-900 dark:text-white">{{ getLastBillDueDate(utility) }}</div>
              <template v-if="isMeteredType(utility.type)">
                <div class="text-gray-500 dark:text-gray-400">Consumo</div>
                <div class="text-right text-gray-900 dark:text-white">{{ getLastConsumption(utility) }}</div>
              </template>
              <template v-else-if="utility.recurring_amount">
                <div class="text-gray-500 dark:text-gray-400">Canone</div>
                <div class="text-right text-gray-900 dark:text-white">{{ formatCurrency(utility.recurring_amount) }}/mese</div>
              </template>
            </div>

            <!-- Alert (metered only) -->
            <div
              v-if="shouldShowReadingAlert(utility)"
              class="mt-3 p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg text-xs text-orange-600 dark:text-orange-400"
            >
              {{ getReadingAlertMessage(utility) }}
            </div>

            <!-- Quick Actions -->
            <div class="mt-3 flex gap-2">
              <button
                v-if="isMeteredType(utility.type)"
                @click.prevent="openAddReading(utility)"
                class="flex-1 py-2.5 text-sm border border-gray-200 dark:border-gray-700 rounded-lg
                       hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-gray-600 dark:text-gray-300
                       flex items-center justify-center gap-1.5"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Lettura
              </button>
              <button
                @click.prevent="openAddBill(utility)"
                class="flex-1 py-2.5 text-sm border border-gray-200 dark:border-gray-700 rounded-lg
                       hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-gray-600 dark:text-gray-300
                       flex items-center justify-center gap-1.5"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                {{ isMeteredType(utility.type) ? 'Bolletta' : 'Fattura' }}
              </button>
            </div>
          </Card>
        </router-link>
      </div>
    </template>

    <!-- Add Utility Modal -->
    <AddUtilityModal
      v-if="showAddUtility"
      :default-property-id="selectedPropertyId"
      @close="showAddUtility = false"
      @created="onUtilityCreated"
    />

    <!-- Add Reading Modal (quick access) -->
    <AddReadingModal
      v-if="showAddReading && readingUtility"
      :utility="readingUtility"
      @close="showAddReading = false"
      @saved="onReadingSaved"
    />

    <!-- Add Bill Modal (quick access) -->
    <AddBillModal
      v-if="showAddBill && billUtility"
      :utility="billUtility"
      @close="showAddBill = false; billUtility = null"
      @saved="onBillSaved"
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
import { useAuthStore } from '@/stores/auth'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddUtilityModal from '@/components/utilities/AddUtilityModal.vue'
import AddReadingModal from '@/components/utilities/AddReadingModal.vue'
import AddBillModal from '@/components/utilities/AddBillModal.vue'
import TemplatesManager from '@/components/utilities/TemplatesManager.vue'

const authStore = useAuthStore()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const showAddUtility = ref(false)
const showAddReading = ref(false)
const showAddBill = ref(false)
const showTemplatesManager = ref(false)
const readingUtility = ref(null)
const billUtility = ref(null)
const properties = ref([])
const selectedPropertyId = ref(null)

// ── Dashboard Computeds ──

const dashboardKPIs = computed(() => {
  const utilities = utilitiesStore.utilities
  let unpaidTotal = 0
  let unpaidCount = 0
  let totalLast12 = 0
  const now = new Date()
  const oneYearAgo = new Date(now)
  oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1)

  utilities.forEach(u => {
    (u.bills || []).forEach(b => {
      if (!b.is_paid) {
        unpaidTotal += b.amount_total || 0
        unpaidCount++
      }
      if (new Date(b.period_end) >= oneYearAgo) {
        totalLast12 += b.amount_total || 0
      }
    })
  })

  return {
    avgMonthly: totalLast12 / 12,
    unpaidTotal,
    unpaidCount,
  }
})

const alerts = computed(() => {
  const result = []
  const now = new Date()
  const threeDays = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)

  utilitiesStore.utilities.forEach(u => {
    // Due soon bills
    const dueSoon = (u.bills || []).filter(b => {
      if (b.is_paid) return false
      const d = new Date(b.due_date)
      return d <= threeDays && d >= now
    })
    if (dueSoon.length > 0) {
      const name = getUtilityName(u.type)
      result.push({
        type: 'warning',
        message: `${name}: ${dueSoon.length} bolletta/e in scadenza`,
        utilityId: u.id
      })
    }

    // Reading alerts (metered only)
    if (u.is_metered !== false && u.readings?.length > 0) {
      const lastReading = new Date(u.readings[0].reading_date)
      const days = Math.floor((now - lastReading) / (1000 * 60 * 60 * 24))
      if (days >= 30) {
        result.push({
          type: 'info',
          message: `${getUtilityName(u.type)}: autolettura consigliata (${days} giorni dall'ultima)`,
          utilityId: u.id
        })
      }
    }
  })

  return result
})

// ── Sorted Utilities (fixed order by type) ──

const typeOrder = {
  electricity: 0, gas: 1, water: 2, waste: 3,
  internet: 4, insurance: 5, affitto: 6, mutuo: 7
}

const sortedUtilities = computed(() => {
  return [...utilitiesStore.utilities].sort((a, b) => {
    return (typeOrder[a.type] ?? 99) - (typeOrder[b.type] ?? 99)
  })
})

// ── Utility Icons & Helpers ──

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
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 21c-4.418 0-8-3.134-8-7 0-4.5 8-11 8-11s8 6.5 8 11c0 3.866-3.582 7-8 7z' })
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

const InternetIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9' })
    ])
  }
}

const InsuranceIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' })
    ])
  }
}

const RentIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' })
    ])
  }
}

const MortgageIcon = {
  render() {
    return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4' })
    ])
  }
}

function getUtilityIcon(type) {
  const icons = {
    electricity: ElectricityIcon, gas: GasIcon, water: WaterIcon, waste: WasteIcon,
    internet: InternetIcon, insurance: InsuranceIcon, affitto: RentIcon, mutuo: MortgageIcon
  }
  return icons[type] || ElectricityIcon
}

function getUtilityColorClasses(type) {
  const classes = {
    electricity: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800 text-yellow-500',
    gas: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800 text-orange-500',
    water: 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800 text-cyan-500',
    waste: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-500',
    internet: 'bg-indigo-50 dark:bg-indigo-900/20 border-indigo-200 dark:border-indigo-800 text-indigo-500',
    insurance: 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-200 dark:border-emerald-800 text-emerald-500',
    affitto: 'bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800 text-purple-500',
    mutuo: 'bg-sky-50 dark:bg-sky-900/20 border-sky-200 dark:border-sky-800 text-sky-500',
  }
  return classes[type] || classes.electricity
}

function getUtilityName(type) {
  const names = {
    electricity: 'Luce', gas: 'Gas', water: 'Acqua', waste: 'Rifiuti',
    internet: 'Internet', insurance: 'Assicurazione', affitto: 'Affitto', mutuo: 'Mutuo'
  }
  return names[type] || type
}

function isMeteredType(type) {
  return ['electricity', 'gas', 'water', 'waste'].includes(type)
}

function getConsumptionUnit(type) {
  const units = { electricity: 'kWh', gas: 'Smc', water: 'mc', waste: 'mq' }
  return units[type] || ''
}

function getLastConsumption(utility) {
  if (!isMeteredType(utility.type)) return null
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
  if (!isMeteredType(utility.type)) return false
  if (!utility.readings || utility.readings.length === 0) return false
  const lastReading = new Date(utility.readings[0].reading_date)
  const now = new Date()
  const daysSinceReading = Math.floor((now - lastReading) / (1000 * 60 * 60 * 24))
  return daysSinceReading >= 25
}

function getReadingAlertMessage(utility) {
  if (!utility.readings || utility.readings.length === 0) return ''
  const lastReading = new Date(utility.readings[0].reading_date)
  const now = new Date()
  const daysSinceReading = Math.floor((now - lastReading) / (1000 * 60 * 60 * 24))
  if (daysSinceReading >= 30) return 'Autolettura consigliata'
  return `Autolettura consigliata tra ${30 - daysSinceReading} giorni`
}

// ── Actions ──

async function fetchProperties() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      properties.value = data
      const current = data.find(p => p.is_current) || data[0]
      selectedPropertyId.value = current.id
      utilitiesStore.fetchUtilities({ property_id: current.id })
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

function onPropertyChange() {
  if (selectedPropertyId.value) {
    utilitiesStore.fetchUtilities({ property_id: selectedPropertyId.value })
  }
}

function openAddReading(utility) {
  readingUtility.value = utility
  showAddReading.value = true
}

function openAddBill(utility) {
  billUtility.value = utility
  showAddBill.value = true
}

function onBillSaved() {
  showAddBill.value = false
  billUtility.value = null
  if (selectedPropertyId.value) {
    utilitiesStore.fetchUtilities({ property_id: selectedPropertyId.value })
  } else {
    utilitiesStore.fetchUtilities()
  }
}

function onUtilityCreated() {
  showAddUtility.value = false
  if (selectedPropertyId.value) {
    utilitiesStore.fetchUtilities({ property_id: selectedPropertyId.value })
  } else {
    utilitiesStore.fetchUtilities()
  }
}

function onReadingSaved() {
  showAddReading.value = false
  readingUtility.value = null
  if (selectedPropertyId.value) {
    utilitiesStore.fetchUtilities({ property_id: selectedPropertyId.value })
  } else {
    utilitiesStore.fetchUtilities()
  }
}

onMounted(() => {
  fetchProperties()
})
</script>
