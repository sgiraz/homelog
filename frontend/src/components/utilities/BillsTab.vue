<template>
  <div>
    <!-- Filters -->
    <div class="flex flex-col sm:flex-row gap-2 mb-4">
      <div class="flex gap-2 flex-1">
        <input
          v-model="billSearch"
          type="search"
          placeholder="Cerca bolletta..."
          class="flex-1 px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <select
          v-model="billStatusFilter"
          class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">Tutte</option>
          <option value="unpaid">Qualcuna non pagata</option>
          <option value="paid">Tutte pagate</option>
        </select>
      </div>
      <div class="flex gap-2">
        <input
          v-model="billDateFrom"
          type="date"
          class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
          title="Da data"
        />
        <input
          v-model="billDateTo"
          type="date"
          class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
          title="A data"
        />
      </div>
    </div>

    <!-- Bill Summary -->
    <div v-if="filteredBills.length > 0" class="mb-3 flex items-center justify-between">
      <span class="text-sm text-gray-500 dark:text-gray-400">
        {{ filteredBills.length }} bollette — Totale: {{ formatCurrency(filteredBillsTotal) }}
      </span>
      <Button size="sm" @click="openAddBill">
        <svg class="w-4 h-4 sm:mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">Aggiungi</span>
      </Button>
    </div>

    <!-- Empty -->
    <div v-if="filteredBills.length === 0" class="text-center py-8">
      <p class="text-gray-500 dark:text-gray-400 mb-3">
        {{ utility.bills?.length ? 'Nessuna bolletta corrisponde ai filtri' : 'Nessuna bolletta registrata' }}
      </p>
      <Button size="sm" @click="openAddBill">Aggiungi bolletta</Button>
    </div>

    <!-- Bills List -->
    <div v-else class="space-y-2">
      <div
        v-for="bill in filteredBills"
        :key="bill.id"
        :ref="(el) => registerRow(bill.id, el)"
        class="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl
               hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
        :class="{ 'search-flash': isHighlighted(bill.id) }"
      >
        <div v-if="hasMultipleInstallments(bill)" class="mb-2 flex items-center justify-between">
          <button
            type="button"
            class="text-xs text-purple-700 dark:text-purple-300 hover:underline flex items-center gap-1"
            @click="toggleInstallments(bill.id)"
          >
            <svg class="w-3 h-3 transition-transform" :class="{ 'rotate-90': expandedBills.has(bill.id) }"
              fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
            {{ paidInstallmentsCount(bill) }}/{{ bill.installments.length }} rate pagate
          </button>
        </div>
        <template v-if="hasMultipleInstallments(bill) && expandedBills.has(bill.id)">
          <div class="mb-3 space-y-1.5 pl-4 border-l-2 border-purple-300 dark:border-purple-700">
            <div v-for="inst in bill.installments" :key="inst.id"
              class="flex items-center justify-between py-1.5 text-sm">
              <label class="flex items-center gap-2 flex-1 min-w-0"
                :class="inst.is_locked ? 'cursor-not-allowed' : 'cursor-pointer'">
                <input
                  type="checkbox"
                  :checked="inst.is_paid"
                  :disabled="inst.is_locked"
                  @change="toggleInstallmentPaid(bill, inst, $event.target.checked)"
                  class="w-4 h-4 text-purple-600 rounded border-gray-300 focus:ring-purple-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  :title="inst.is_locked ? lockedHint : ''"
                />
                <span class="text-gray-500 dark:text-gray-400 w-10">#{{ inst.number }}</span>
                <span class="text-gray-900 dark:text-white font-medium">{{ formatCurrency(inst.amount) }}</span>
                <span class="text-xs text-gray-400 dark:text-gray-500">scad. {{ formatDate(inst.due_date) }}</span>
              </label>
              <span v-if="inst.is_locked" class="text-xs text-amber-600 dark:text-amber-400" :title="lockedHint">🔒 saldata</span>
              <span v-else-if="inst.is_paid" class="text-xs text-green-600 dark:text-green-400">✓ pagata</span>
            </div>
          </div>
        </template>
        <div class="flex items-start justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-semibold text-gray-900 dark:text-white">
                {{ formatCurrency(bill.amount_total) }}
              </span>
              <span
                v-if="bill.original_amount != null && bill.original_currency"
                class="text-xs text-gray-500 dark:text-gray-400"
                :title="'Importo originale fatturato'"
              >
                ({{ formatOriginal(bill.original_amount, bill.original_currency) }})
              </span>
              <span :class="[
                'px-2 py-0.5 text-xs rounded-full font-medium',
                bill.is_paid
                  ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                  : isDueSoon(bill)
                    ? 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300'
                    : 'bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300'
              ]">
                {{ bill.is_paid ? 'Pagata' : isDueSoon(bill) ? 'In scadenza' : 'Da pagare' }}
              </span>
            </div>
            <div class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {{ formatPeriod(bill.period_start, bill.period_end) }}
            </div>
            <div class="flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500 mt-1">
              <span>Scad. {{ formatDate(bill.due_date) }}</span>
              <span v-if="bill.consumption_total">{{ formatConsumption(bill.consumption_total) }} {{ consumptionUnit }}</span>
              <span v-if="bill.bill_number" class="font-mono">N. {{ bill.bill_number }}</span>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1 flex-shrink-0">
            <span v-if="bill.is_locked" class="px-2 py-0.5 text-xs rounded-full font-medium bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300" :title="lockedHint">
              🔒 Saldata
            </span>
            <button
              v-if="!bill.is_paid && !hasMultipleInstallments(bill) && !bill.is_locked"
              @click="markBillAsPaid(bill)"
              class="p-2.5 rounded-lg text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20"
              title="Segna come pagata"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
            </button>
            <button
              @click="openEditBill(bill)"
              class="p-2.5 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20"
              :title="bill.is_locked ? 'Visualizza dettagli' : 'Modifica'"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button
              :disabled="bill.is_locked"
              @click="confirmDeleteBill(bill)"
              class="p-2.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-gray-400"
              :title="bill.is_locked ? lockedHint : 'Elimina'"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Bill Modal -->
    <AddBillModal
      v-if="showBillModal"
      :utility="utility"
      :bill="editingBill"
      @close="closeBillModal"
      @saved="onBillSaved"
      @installment-updated="onInstallmentUpdated"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { utilitiesAPI } from '@/api/client'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { useHighlight } from '@/composables/useHighlight'
import { formatDate as _formatDate, formatPeriod as _formatPeriod, formatNumber as _formatNumber, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Button from '@/components/common/Button.vue'
import AddBillModal from '@/components/utilities/AddBillModal.vue'

defineOptions({ name: 'BillsTab' })

const props = defineProps({
  utility: { type: Object, required: true },
  consumptionUnit: { type: String, default: '' },
})

const emit = defineEmits(['bill-saved', 'bill-deleted', 'bill-updated'])

const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

// Bill filters
const billSearch = ref('')
const billStatusFilter = ref('all')
const billDateFrom = ref('')
const billDateTo = ref('')

// Bill modal
const showBillModal = ref(false)
const editingBill = ref(null)

// Installment expand state
const expandedBills = ref(new Set())

const lockedHint = 'Spesa già saldata da uno o più membri. Annulla i pagamenti dal Bilancio per modificare la bolletta.'

function hasMultipleInstallments(bill) {
  return Array.isArray(bill.installments) && bill.installments.length > 1
}

function paidInstallmentsCount(bill) {
  return (bill.installments || []).filter(i => i.is_paid).length
}

function toggleInstallments(billId) {
  const s = expandedBills.value
  if (s.has(billId)) s.delete(billId)
  else s.add(billId)
  expandedBills.value = new Set(s)
}

async function toggleInstallmentPaid(bill, inst, newValue) {
  try {
    await utilitiesAPI.updateInstallment(props.utility.id, bill.id, inst.id, {
      is_paid: newValue,
      paid_at: newValue ? new Date().toISOString() : null
    })
    emit('bill-updated')
  } catch (err) {
    console.error('Error toggling installment:', err)
    if (err?.response?.status === 409) {
      await confirm({
        title: 'Operazione bloccata',
        message: err.response.data?.error || lockedHint,
        confirmText: 'Ho capito',
        variant: 'info'
      })
      emit('bill-updated')
    }
  }
}

// ── Computed ──

const filteredBills = computed(() => {
  let bills = props.utility?.bills || []

  if (billStatusFilter.value === 'paid') bills = bills.filter(b => b.is_paid)
  else if (billStatusFilter.value === 'unpaid') bills = bills.filter(b => !b.is_paid)

  if (billSearch.value.trim()) {
    const q = billSearch.value.toLowerCase()
    bills = bills.filter(b =>
      b.bill_number?.toLowerCase().includes(q) ||
      String(b.amount_total).includes(q) ||
      b.provider?.toLowerCase().includes(q)
    )
  }

  if (billDateFrom.value) {
    const from = new Date(billDateFrom.value)
    bills = bills.filter(b => new Date(b.period_end) >= from)
  }
  if (billDateTo.value) {
    const to = new Date(billDateTo.value)
    bills = bills.filter(b => new Date(b.period_start) <= to)
  }

  return bills
})

const filteredBillsTotal = computed(() => {
  return filteredBills.value.reduce((sum, b) => sum + (b.amount_total || 0), 0)
})

const { isHighlighted, registerRow } = useHighlight({
  source: () => filteredBills.value,
})

// ── Functions ──

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatOriginal(value, currency) {
  return _formatCurrency(value, { ...settingsStore.formatSettings, currency })
}

function formatConsumption(value) {
  if (value == null || value === 0) return '0'
  return _formatNumber(parseFloat(value), settingsStore.formatSettings)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function formatPeriod(start, end) {
  return _formatPeriod(start, end, settingsStore.dateSettings)
}

function isDueSoon(bill) {
  const now = new Date()
  const dueDate = new Date(bill.due_date)
  const threeDays = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)
  return dueDate <= threeDays && dueDate >= now
}

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

function onBillSaved() {
  closeBillModal()
  emit('bill-saved')
}

// Fired when a single installment was toggled from inside the open modal.
// Refresh list state (header "N/M pagate", bill.is_paid) WITHOUT closing the modal.
function onInstallmentUpdated() {
  emit('bill-updated')
}

async function markBillAsPaid(bill) {
  try {
    await utilitiesStore.updateBill(props.utility.id, bill.id, {
      is_paid: true,
      paid_date: new Date().toISOString()
    })
    emit('bill-updated')
  } catch (err) {
    console.error('Error marking bill as paid:', err)
    if (err?.response?.status === 409) {
      await confirm({
        title: 'Operazione bloccata',
        message: err.response.data?.error || lockedHint,
        confirmText: 'Ho capito',
        variant: 'info'
      })
      emit('bill-updated')
    }
  }
}

async function confirmDeleteBill(bill) {
  const ok = await confirm({
    title: 'Elimina bolletta',
    message: `Eliminare la bolletta di ${formatCurrency(bill.amount_total)}?`,
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return
  try {
    await utilitiesStore.deleteBill(props.utility.id, bill.id)
    emit('bill-deleted')
  } catch (err) {
    console.error('Error deleting bill:', err)
    if (err?.response?.status === 409) {
      await confirm({
        title: 'Operazione bloccata',
        message: err.response.data?.error || lockedHint,
        confirmText: 'Ho capito',
        variant: 'info'
      })
      emit('bill-updated')
    }
  }
}
</script>

<style scoped>
.search-flash {
  animation: search-flash 2.2s ease-out;
}
@keyframes search-flash {
  0%   { box-shadow: 0 0 0 3px rgba(59,130,246,.55); background-color: rgba(59,130,246,.12); }
  70%  { box-shadow: 0 0 0 3px rgba(59,130,246,.25); background-color: rgba(59,130,246,.04); }
  100% { box-shadow: 0 0 0 0 transparent; background-color: transparent; }
}
</style>
