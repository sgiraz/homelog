<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto">
      <!-- Header -->
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div :class="['w-12 h-12 rounded-xl flex items-center justify-center text-2xl', getUtilityIconClass(localUtility.type)]">
            {{ getUtilityIcon(localUtility.type) }}
          </div>
          <div>
            <h3 class="text-xl font-bold text-gray-900 dark:text-white">{{ localUtility.provider }}</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ getUtilityTypeLabel(localUtility.type) }}</p>
          </div>
        </div>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Utility Info -->
      <div class="grid grid-cols-2 gap-4 mb-6 text-sm">
        <div v-if="localUtility.service_code">
          <span class="text-gray-500 dark:text-gray-400">{{ localUtility.type === 'electricity' ? 'POD' : localUtility.type === 'gas' ? 'PDR' : 'Codice' }}</span>
          <p class="font-medium text-gray-900 dark:text-white">{{ localUtility.service_code }}</p>
        </div>
        <div v-if="localUtility.customer_code">
          <span class="text-gray-500 dark:text-gray-400">Codice Cliente</span>
          <p class="font-medium text-gray-900 dark:text-white">{{ localUtility.customer_code }}</p>
        </div>
        <div v-if="localUtility.power_capacity">
          <span class="text-gray-500 dark:text-gray-400">Potenza</span>
          <p class="font-medium text-gray-900 dark:text-white">{{ localUtility.power_capacity }} kW</p>
        </div>
        <div v-if="localUtility.customer_portal">
          <span class="text-gray-500 dark:text-gray-400">Area clienti</span>
          <a :href="localUtility.customer_portal" target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">
            Apri portale
          </a>
        </div>
      </div>

      <!-- Tabs -->
      <div class="flex border-b border-gray-200 dark:border-gray-700 mb-4">
        <button
          @click="activeTab = 'bills'"
          :class="[
            'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
            activeTab === 'bills'
              ? 'border-blue-500 text-blue-600 dark:text-blue-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Bollette ({{ localUtility.bills?.length || 0 }})
        </button>
        <button
          @click="activeTab = 'readings'"
          :class="[
            'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
            activeTab === 'readings'
              ? 'border-blue-500 text-blue-600 dark:text-blue-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Letture ({{ localUtility.readings?.length || 0 }})
        </button>
        <button
          v-if="localUtility.type !== 'waste'"
          @click="activeTab = 'comparison'"
          :class="[
            'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
            activeTab === 'comparison'
              ? 'border-blue-500 text-blue-600 dark:text-blue-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Confronto
        </button>
      </div>

      <!-- Bills Tab -->
      <div v-if="activeTab === 'bills'">
        <div class="flex justify-between items-center mb-4">
          <h4 class="font-medium text-gray-900 dark:text-white">Storico Bollette</h4>
          <Button size="sm" @click="openAddBill">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Aggiungi
          </Button>
        </div>

        <div v-if="!localUtility.bills?.length" class="text-center py-8 text-gray-500 dark:text-gray-400">
          Nessuna bolletta registrata
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="bill in localUtility.bills"
            :key="bill.id"
            class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg"
          >
            <div class="flex items-start justify-between">
              <div>
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">
                    {{ formatCurrency(bill.amount_total) }}
                  </span>
                  <span :class="[
                    'px-2 py-0.5 text-xs rounded-full',
                    bill.is_paid
                      ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                      : 'bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300'
                  ]">
                    {{ bill.is_paid ? 'Pagata' : 'Da pagare' }}
                  </span>
                  <span
                    v-if="isDueSoon(bill) && !bill.is_paid"
                    class="px-2 py-0.5 text-xs rounded-full bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300"
                  >
                    Scadenza vicina
                  </span>
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Periodo: {{ formatPeriod(bill.period_start, bill.period_end) }}
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-500">
                  Scadenza: {{ formatDate(bill.due_date) }}
                </div>
                <div v-if="bill.consumption_total" class="text-sm text-gray-500 dark:text-gray-500">
                  Consumo: {{ bill.consumption_total }} {{ getConsumptionUnit(localUtility.type) }}
                </div>
              </div>
              <div class="flex flex-col gap-5 items-end">

                <div class="flex gap-2">
                  <button
                    @click="openEditBill(bill)"
                    class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700"
                  >
                    Modifica
                  </button>
                  <button
                    @click="confirmDeleteBill(bill)"
                    class="text-sm text-red-600 dark:text-red-400 hover:text-red-700"
                  >
                    Elimina
                  </button>
                </div>
                <button
                  v-if="!bill.is_paid"
                  @click="markBillAsPaid(bill)"
                  class="text-sm px-3 py-1 rounded-lg bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300 hover:bg-green-200 dark:hover:bg-green-900 mt-1"
                >
                  Paga
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Readings Tab -->
      <div v-if="activeTab === 'readings'">
        <div class="flex justify-between items-center mb-4">
          <h4 class="font-medium text-gray-900 dark:text-white">Storico Letture (Autolettura)</h4>
          <Button size="sm" @click="openAddReading">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Aggiungi
          </Button>
        </div>

        <div v-if="!localUtility.readings?.length" class="text-center py-8 text-gray-500 dark:text-gray-400">
          Nessuna lettura registrata
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="reading in localUtility.readings"
            :key="reading.id"
            class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg"
          >
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium text-gray-900 dark:text-white">
                  <!-- For electricity show F1/F2/F3, for gas/water show single value -->
                  <template v-if="localUtility.type === 'electricity'">
                    <span v-if="reading.value_f1" class="mr-2">F1: {{ reading.value_f1 }}</span>
                    <span v-if="reading.value_f2" class="mr-2">F2: {{ reading.value_f2 }}</span>
                    <span v-if="reading.value_f3">F3: {{ reading.value_f3 }}</span>
                    <span class="text-gray-500 text-sm ml-1">kWh</span>
                  </template>
                  <template v-else>
                    {{ reading.value || reading.value_f1 || '-' }} {{ getConsumptionUnit(localUtility.type) }}
                  </template>
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  {{ formatDate(reading.reading_date) }}
                  <span v-if="reading.source === 'submitted'" class="ml-2 px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 text-xs rounded">
                    Inviata
                  </span>
                </div>
                <div v-if="reading.notes" class="text-sm text-gray-500 dark:text-gray-500 mt-1">
                  {{ reading.notes }}
                </div>
              </div>
              <div class="flex gap-2">
                <button
                  @click="openEditReading(reading)"
                  class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700"
                >
                  Modifica
                </button>
                <button
                  @click="confirmDeleteReading(reading)"
                  class="text-sm text-red-600 dark:text-red-400 hover:text-red-700"
                >
                  Elimina
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Comparison Tab -->
      <div v-if="activeTab === 'comparison'">
        <ReadingComparisonCard
          ref="comparisonCard"
          :utility-id="localUtility.id"
          :utility-type="localUtility.type"
        />
      </div>

      <!-- Footer Actions -->
      <div class="flex justify-between items-center mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
        <button
          @click="confirmDeleteUtility"
          class="text-red-600 dark:text-red-400 text-sm hover:text-red-700"
        >
          Elimina utenza
        </button>
        <Button variant="secondary" @click="$emit('close')">
          Chiudi
        </Button>
      </div>

      <!-- Add/Edit Bill Modal -->
      <AddBillModal
        v-if="showBillModal"
        :utility="localUtility"
        :bill="editingBill"
        @close="closeBillModal"
        @saved="onBillSaved"
      />

      <!-- Add/Edit Reading Modal -->
      <AddReadingModal
        v-if="showReadingModal"
        :utility="localUtility"
        :reading="editingReading"
        @close="closeReadingModal"
        @saved="onReadingSaved"
      />
    </Card>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddBillModal from './AddBillModal.vue'
import AddReadingModal from './AddReadingModal.vue'
import ReadingComparisonCard from './ReadingComparisonCard.vue'

const props = defineProps({
  utility: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated'])

const utilitiesStore = useUtilitiesStore()

// Local reactive copy of utility for immediate updates
const localUtility = reactive({ ...props.utility })

// Watch for external updates
watch(() => props.utility, (newVal) => {
  Object.assign(localUtility, newVal)
}, { deep: true })

const activeTab = ref('bills')
const showBillModal = ref(false)
const showReadingModal = ref(false)
const editingBill = ref(null)
const editingReading = ref(null)

function getUtilityIcon(type) {
  const icons = {
    electricity: '\u26A1',
    gas: '\uD83D\uDD25',
    water: '\uD83D\uDCA7',
    waste: '\u267B\uFE0F'
  }
  return icons[type] || '\u26A1'
}

function getUtilityIconClass(type) {
  const classes = {
    electricity: 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-600',
    gas: 'bg-orange-100 dark:bg-orange-900/50 text-orange-600',
    water: 'bg-blue-100 dark:bg-blue-900/50 text-blue-600',
    waste: 'bg-green-100 dark:bg-green-900/50 text-green-600'
  }
  return classes[type] || classes.electricity
}

function getUtilityTypeLabel(type) {
  const labels = {
    electricity: 'Luce',
    gas: 'Gas',
    water: 'Acqua',
    waste: 'Rifiuti'
  }
  return labels[type] || type
}

function getConsumptionUnit(type) {
  const units = {
    electricity: 'kWh',
    gas: 'Smc',
    water: 'mc',
    waste: ''
  }
  return units[type] || ''
}

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(value || 0)
}

function formatDate(dateStr) {
  return new Date(dateStr).toLocaleDateString('it-IT', { day: '2-digit', month: 'short', year: 'numeric' })
}

function formatPeriod(start, end) {
  const startDate = new Date(start)
  const endDate = new Date(end)
  const options = { month: 'short', year: 'numeric' }
  return `${startDate.toLocaleDateString('it-IT', options)} - ${endDate.toLocaleDateString('it-IT', options)}`
}

function isDueSoon(bill) {
  const now = new Date()
  const dueDate = new Date(bill.due_date)
  const threeDaysFromNow = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)
  return dueDate <= threeDaysFromNow && dueDate >= now
}

// Bill functions
function openAddBill() {
  editingBill.value = null
  showBillModal.value = true
}

function openEditBill(bill) {
  editingBill.value = bill
  showBillModal.value = true
}

function closeBillModal() {
  showBillModal.value = false
  editingBill.value = null
}

async function onBillSaved() {
  closeBillModal()
  await refreshUtility()
  emit('updated')
}

async function markBillAsPaid(bill) {
  try {
    await utilitiesStore.updateBill(localUtility.id, bill.id, {
      is_paid: true,
      paid_date: new Date().toISOString()
    })
    await refreshUtility()
    emit('updated')
  } catch (err) {
    console.error('Error marking bill as paid:', err)
  }
}

async function confirmDeleteBill(bill) {
  if (confirm('Sei sicuro di voler eliminare questa bolletta?')) {
    try {
      await utilitiesStore.deleteBill(localUtility.id, bill.id)
      await refreshUtility()
      emit('updated')
    } catch (err) {
      console.error('Error deleting bill:', err)
    }
  }
}

// Reading functions
function openAddReading() {
  editingReading.value = null
  showReadingModal.value = true
}

function openEditReading(reading) {
  editingReading.value = reading
  showReadingModal.value = true
}

function closeReadingModal() {
  showReadingModal.value = false
  editingReading.value = null
}

async function onReadingSaved() {
  closeReadingModal()
  await refreshUtility()
  emit('updated')
}

async function confirmDeleteReading(reading) {
  if (confirm('Sei sicuro di voler eliminare questa lettura?')) {
    try {
      await utilitiesStore.deleteReading(localUtility.id, reading.id)
      await refreshUtility()
      emit('updated')
    } catch (err) {
      console.error('Error deleting reading:', err)
    }
  }
}

// Utility functions
async function refreshUtility() {
  try {
    const updated = await utilitiesStore.fetchUtility(localUtility.id)
    Object.assign(localUtility, updated)
  } catch (err) {
    console.error('Error refreshing utility:', err)
  }
}

async function confirmDeleteUtility() {
  if (confirm('Sei sicuro di voler eliminare questa utenza e tutti i dati associati?')) {
    try {
      await utilitiesStore.deleteUtility(localUtility.id)
      emit('close')
      emit('updated')
    } catch (err) {
      console.error('Error deleting utility:', err)
    }
  }
}
</script>
