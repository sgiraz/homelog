<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <p class="text-gray-600 dark:text-gray-400 mt-1">Panoramica delle tue spese</p>
      </div>
      <Button @click="showAddExpense = true">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Aggiungi Spesa
      </Button>
    </div>

    <!-- KPI Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Spese Mese</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ formatCurrency(totalMonth) }}
            </div>
          </div>
          <div class="w-12 h-12 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center">
            <svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </div>
      </Card>

      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Numero Spese</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ expensesStore.expenses.length }}
            </div>
          </div>
          <div class="w-12 h-12 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
            <svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
        </div>
      </Card>

      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Media Giornaliera</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ formatCurrency(dailyAverage) }}
            </div>
          </div>
          <div class="w-12 h-12 bg-purple-100 dark:bg-purple-900 rounded-full flex items-center justify-center">
            <svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </div>
        </div>
      </Card>

      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Spese Anno</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ formatCurrency(totalYear) }}
            </div>
          </div>
          <div class="w-12 h-12 bg-orange-100 dark:bg-orange-900 rounded-full flex items-center justify-center">
            <svg class="w-6 h-6 text-orange-600 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
        </div>
      </Card>
    </div>

    <!-- Grafici -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card class="p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Spese per Categoria</h3>
        <PieChart v-if="hasCategoryData" :chartData="categoryChartData" />
        <div v-else class="h-64 flex items-center justify-center text-gray-500">
          Nessun dato disponibile
        </div>
      </Card>

      <Card class="p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Trend Mensile</h3>
        <BarChart v-if="hasMonthlyData" :chartData="monthlyChartData" />
        <div v-else class="h-64 flex items-center justify-center text-gray-500">
          Nessun dato disponibile
        </div>
      </Card>
    </div>

    <!-- Filtri -->
    <Card class="p-4">
      <div class="flex flex-wrap items-center gap-4">
        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">Categoria:</label>
          <select
            v-model="filters.categoryId"
            @change="applyFilters"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
          >
            <option value="">Tutte</option>
            <option value="1">Casa</option>
            <option value="2">Alimentari</option>
            <option value="3">Trasporti</option>
            <option value="4">Salute</option>
            <option value="5">Intrattenimento</option>
            <option value="6">Famiglia</option>
            <option value="7">Abbigliamento</option>
            <option value="8">Istruzione</option>
            <option value="9">Tecnologia</option>
          </select>
        </div>

        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">Da:</label>
          <input
            v-model="filters.from"
            @change="applyFilters"
            type="date"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
          />
        </div>

        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">A:</label>
          <input
            v-model="filters.to"
            @change="applyFilters"
            type="date"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
          />
        </div>

        <Button v-if="hasFilters" @click="resetFilters" variant="secondary" class="text-sm">
          Reset Filtri
        </Button>
      </div>
    </Card>

    <!-- Lista Spese Recenti -->
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white">Spese Recenti</h2>
        <router-link
          v-if="expensesStore.expenses.length > 3"
          to="/expenses"
          class="text-blue-600 hover:text-blue-700 dark:text-blue-400 text-sm"
        >
          Vedi tutte
        </router-link>
      </div>

      <div v-if="expensesStore.loading" class="text-center py-8 text-gray-600 dark:text-gray-400">
        <svg class="animate-spin h-8 w-8 mx-auto mb-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
        </svg>
        Caricamento...
      </div>

      <div v-else-if="expensesStore.expenses.length === 0" class="text-center py-8">
        <svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
        <p class="text-gray-600 dark:text-gray-400">Nessuna spesa registrata</p>
        <Button @click="showAddExpense = true" class="mt-4">
          Aggiungi la tua prima spesa
        </Button>
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="expense in expensesStore.expenses.slice(0, 3)"
          :key="expense.id"
          class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg
                 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors group"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-medium text-gray-900 dark:text-white truncate">
                  {{ expense.description || 'Senza descrizione' }}
                </span>
                <!-- Badge Split -->
                <span
                  v-if="expense.is_split"
                  class="px-2 py-0.5 text-xs rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300"
                >
                  Split
                </span>
                <!-- Badge Saldato/Da saldare -->
                <span
                  v-if="expense.is_split"
                  :class="[
                    'px-2 py-0.5 text-xs rounded-full',
                    isExpenseSettled(expense)
                      ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                      : 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300'
                  ]"
                >
                  {{ isExpenseSettled(expense) ? 'Saldato' : 'Da saldare' }}
                </span>
              </div>
              <div class="text-sm text-gray-600 dark:text-gray-400 mt-1 flex flex-wrap items-center gap-2">
                <span>{{ formatDate(expense.date) }}</span>
                <span
                  v-if="expense.category"
                  class="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs"
                >
                  {{ expense.category.name }}
                </span>
                <span v-if="expense.is_split && expense.paid_by" class="text-xs flex items-center gap-1">
                  Pagato da {{ expense.paid_by.name }}
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                  {{ getSplitPartners(expense) }}
                </span>
              </div>
            </div>
            <div class="text-right ml-4">
              <div class="text-xl font-bold text-blue-600 dark:text-blue-400">
                {{ formatCurrency(expense.amount) }}
              </div>
              <!-- Mostra quota se split -->
              <div v-if="expense.is_split && expense.splits?.length" class="text-xs text-gray-500 dark:text-gray-400">
                ({{ formatCurrency(expense.splits[0]?.amount || 0) }} a testa)
              </div>
              <div class="flex gap-2 justify-end mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  v-if="canEditExpense(expense)"
                  @click="editExpense(expense)"
                  class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"
                >
                  Modifica
                </button>
                <button
                  @click="deleteExpenseConfirm(expense.id)"
                  class="text-sm text-red-600 hover:text-red-700 dark:text-red-400"
                >
                  Elimina
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- Modal Aggiungi Spesa -->
    <AddExpenseModal
      v-if="showAddExpense"
      @close="showAddExpense = false"
      @created="onExpenseCreated"
    />

    <!-- Modal Modifica Spesa -->
    <EditExpenseModal
      v-if="showEditExpense && editingExpense"
      :expense="editingExpense"
      @close="showEditExpense = false; editingExpense = null"
      @updated="onExpenseUpdated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import BarChart from '@/components/charts/BarChart.vue'
import PieChart from '@/components/charts/PieChart.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'

const expensesStore = useExpensesStore()
const settingsStore = useSettingsStore()

const showAddExpense = ref(false)
const showEditExpense = ref(false)
const editingExpense = ref(null)
const filters = ref({
  categoryId: '',
  from: '',
  to: ''
})

// Computed for KPIs
const totalMonth = computed(() => {
  const now = new Date()
  return expensesStore.expenses
    .filter(e => {
      const expenseDate = new Date(e.date)
      return expenseDate.getMonth() === now.getMonth() && expenseDate.getFullYear() === now.getFullYear()
    })
    .reduce((sum, e) => sum + e.amount, 0)
})

const totalYear = computed(() => {
  const now = new Date()
  return expensesStore.expenses
    .filter(e => {
      const expenseDate = new Date(e.date)
      return expenseDate.getFullYear() === now.getFullYear()
    })
    .reduce((sum, e) => sum + e.amount, 0)
})

const dailyAverage = computed(() => {
  if (expensesStore.expenses.length === 0) return 0
  const daysInMonth = new Date().getDate()
  return totalMonth.value / daysInMonth
})

const hasFilters = computed(() => {
  return filters.value.categoryId || filters.value.from || filters.value.to
})

// Chart data
const categoryColors = [
  '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#14B8A6', '#F97316', '#6366F1'
]

const hasCategoryData = computed(() => {
  return Object.keys(categoryData.value).length > 0
})

const categoryData = computed(() => {
  const categories = {}
  expensesStore.expenses.forEach(e => {
    const catName = e.category?.name || 'Altro'
    categories[catName] = (categories[catName] || 0) + e.amount
  })
  return categories
})

const categoryChartData = computed(() => {
  const labels = Object.keys(categoryData.value)
  const data = Object.values(categoryData.value)

  return {
    labels,
    datasets: [{
      data,
      backgroundColor: categoryColors.slice(0, labels.length),
      borderWidth: 0
    }]
  }
})

const hasMonthlyData = computed(() => {
  return Object.keys(monthlyData.value).length > 0
})

const monthlyData = computed(() => {
  const months = {}
  const monthNames = ['Gen', 'Feb', 'Mar', 'Apr', 'Mag', 'Giu', 'Lug', 'Ago', 'Set', 'Ott', 'Nov', 'Dic']

  expensesStore.expenses.forEach(e => {
    const date = new Date(e.date)
    const key = `${monthNames[date.getMonth()]} ${date.getFullYear()}`
    months[key] = (months[key] || 0) + e.amount
  })

  return months
})

const monthlyChartData = computed(() => {
  const labels = Object.keys(monthlyData.value).slice(-6) // Last 6 months
  const data = labels.map(l => monthlyData.value[l])

  return {
    labels,
    datasets: [{
      label: 'Spese',
      data,
      backgroundColor: '#3B82F6',
      borderRadius: 4
    }]
  }
})

// Methods
function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR'
  }).format(value || 0)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function applyFilters() {
  const params = {}
  if (filters.value.categoryId) params.category_id = filters.value.categoryId
  if (filters.value.from) params.from = filters.value.from
  if (filters.value.to) params.to = filters.value.to

  expensesStore.fetchExpenses(params)
}

function resetFilters() {
  filters.value = { categoryId: '', from: '', to: '' }
  expensesStore.fetchExpenses()
}

function isExpenseSettled(expense) {
  if (!expense.is_split || !expense.splits || expense.splits.length === 0) return true
  return expense.splits.every(s => s.is_settled)
}

function canEditExpense(expense) {
  if (!expense.is_split) return true
  if (!expense.splits || expense.splits.length === 0) return true
  return !expense.splits.some(s => s.is_settled)
}

function getSplitPartners(expense) {
  if (!expense.is_split || !expense.splits) return ''
  const partners = expense.splits
    .filter(s => s.member_id !== expense.paid_by_member_id && s.member)
    .map(s => s.member.name)
  return partners.join(', ')
}

function editExpense(expense) {
  editingExpense.value = expense
  showEditExpense.value = true
}

function onExpenseCreated() {
  showAddExpense.value = false
  expensesStore.fetchExpenses()
}

function onExpenseUpdated() {
  showEditExpense.value = false
  editingExpense.value = null
  expensesStore.fetchExpenses()
}

async function deleteExpenseConfirm(id) {
  if (confirm('Sei sicuro di voler eliminare questa spesa?')) {
    try {
      await expensesStore.deleteExpense(id)
    } catch (err) {
      alert('Errore eliminazione: ' + err.message)
    }
  }
}

onMounted(() => {
  expensesStore.fetchExpenses()
})
</script>
