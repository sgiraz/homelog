<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <p class="text-gray-600 dark:text-gray-400 mt-1 text-sm sm:text-base">Panoramica delle tue spese</p>
      </div>
      <Button @click="showAddExpense = true">
        <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">Aggiungi Spesa</span>
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
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Spese nel Periodo</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ stats?.count ?? expensesStore.total }}
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
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ categoryChartTitle }}</h3>
        <PieChart
          v-if="hasCategoryData"
          :chartData="categoryChartData"
          :currency="settingsStore.currency"
          :isSubcategory="!!stats?.is_subcategory"
          @slice-click="onPieSliceClick"
        />
        <div v-else class="h-64 flex items-center justify-center text-gray-500">
          Nessun dato disponibile
        </div>
      </Card>

      <Card class="p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ trendChartTitle }}</h3>
          <div class="flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-0.5">
            <button
              @click="trendChartType = 'line'"
              :class="[
                'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
                trendChartType === 'line'
                  ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
              ]"
              title="Grafico a linee"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
              </svg>
            </button>
            <button
              @click="trendChartType = 'bar'"
              :class="[
                'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
                trendChartType === 'bar'
                  ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
              ]"
              title="Grafico a barre"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </button>
          </div>
        </div>
        <template v-if="hasTrendData">
          <LineChart v-if="trendChartType === 'line'" :chartData="trendLineChartData" :currency="settingsStore.currency" />
          <BarChart v-else :chartData="trendBarChartData" :currency="settingsStore.currency" />
        </template>
        <div v-else class="h-64 flex items-center justify-center text-gray-500">
          Nessun dato disponibile
        </div>
      </Card>
    </div>

    <!-- Filtri -->
    <Card class="p-4">
      <!-- Mobile: filtri collassabili -->
      <div class="sm:hidden">
        <div class="flex items-center justify-between">
          <button
            @click="filtersOpen = !filtersOpen"
            class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
            </svg>
            Filtri
            <span
              v-if="activeFiltersCount > 0"
              class="inline-flex items-center justify-center w-5 h-5 text-xs font-bold text-white bg-blue-600 rounded-full"
            >
              {{ activeFiltersCount }}
            </span>
          </button>
          <Button v-if="hasActiveFilters" @click="resetFilters" variant="secondary" size="sm">
            Reset
          </Button>
        </div>
        <Transition name="filter-expand">
          <div v-if="filtersOpen" class="mt-3 space-y-3 border-t border-gray-100 dark:border-gray-700 pt-3">
            <div class="grid grid-cols-2 gap-2">
              <select
                v-model="filters.categoryId"
                @change="applyFilters"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Tutte categorie</option>
                <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.icon }} {{ cat.name }}</option>
              </select>
              <select
                v-model="filters.projectId"
                @change="applyFilters"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Tutti progetti</option>
                <option v-for="proj in projects" :key="proj.id" :value="proj.id">{{ proj.icon }} {{ proj.name }}</option>
              </select>
              <input
                v-model="filters.from"
                @change="applyFilters"
                type="date"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                v-model="filters.to"
                @change="applyFilters"
                type="date"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
        </Transition>
      </div>

      <!-- Desktop: filtri sempre visibili -->
      <div class="hidden sm:flex flex-wrap items-center gap-4">
        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">Categoria:</label>
          <select
            v-model="filters.categoryId"
            @change="applyFilters"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Tutte</option>
            <option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.icon }} {{ cat.name }}
            </option>
          </select>
        </div>

        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600 dark:text-gray-400">Progetto:</label>
          <select
            v-model="filters.projectId"
            @change="applyFilters"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Tutti</option>
            <option v-for="proj in projects" :key="proj.id" :value="proj.id">
              {{ proj.icon }} {{ proj.name }}
            </option>
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

        <Button @click="resetFilters" variant="secondary" class="text-sm">
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
          class="p-3 sm:p-4 border border-gray-200 dark:border-gray-700 rounded-lg
                 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors group"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-medium text-gray-900 dark:text-white line-clamp-2">
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
                      : 'bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-700'
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
                <span v-if="expense.is_split && expense.paid_by" class="text-xs flex items-center gap-1 max-w-full overflow-hidden">
                  <span class="hidden sm:inline">Pagato da</span>
                  <span class="truncate">{{ expense.paid_by.name }}</span>
                  <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                  <span class="truncate">{{ getSplitPartners(expense) }}</span>
                </span>
              </div>
            </div>
            <div class="text-right shrink-0">
              <div class="text-xl font-bold text-blue-600 dark:text-blue-400">
                {{ formatCurrency(expense.amount) }}
              </div>
              <!-- Mostra quota se split -->
              <div v-if="expense.is_split && expense.splits?.length" class="text-xs text-gray-500 dark:text-gray-400">
                ({{ formatCurrency(expense.splits[0]?.amount || 0) }} a testa)
              </div>
              <!-- Bill-linked indicator -->
              <div v-if="expense.bill_id" class="text-xs text-orange-600 dark:text-orange-400 mt-1 text-right">
                Da bolletta
              </div>
              <!-- Actions: only visible to the creator, not for bill-linked or fully-settled expenses -->
              <div
                v-if="isOwner(expense) && !expense.bill_id && !(expense.is_split && isExpenseSettled(expense))"
                class="flex gap-1 justify-end mt-1 sm:opacity-0 sm:group-hover:opacity-100 sm:transition-opacity"
              >
                <button
                  @click="editExpense(expense)"
                  class="p-1.5 text-blue-600 hover:text-blue-700 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded"
                  aria-label="Modifica spesa"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                </button>
                <button
                  @click="deleteExpenseConfirm(expense.id)"
                  class="p-1.5 text-red-600 hover:text-red-700 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                  aria-label="Elimina spesa"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
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
import { ref, computed, onMounted, Transition } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { categoriesAPI, projectsAPI, expensesAPI } from '@/api/client'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import { useConfirm } from '@/composables/useConfirm'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import BarChart from '@/components/charts/BarChart.vue'
import LineChart from '@/components/charts/LineChart.vue'
import PieChart from '@/components/charts/PieChart.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'

const expensesStore = useExpensesStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const showAddExpense = ref(false)
const showEditExpense = ref(false)
const editingExpense = ref(null)
const trendChartType = ref('line')
const filtersOpen = ref(false)
const categories = ref([])
const projects = ref([])
function defaultDateRange() {
  const today = new Date()
  const from = new Date(today)
  from.setFullYear(from.getFullYear() - 1)
  from.setDate(from.getDate() + 1)
  return {
    from: from.toISOString().split('T')[0],
    to: today.toISOString().split('T')[0]
  }
}

const { from: defaultFrom, to: defaultTo } = defaultDateRange()

const filters = ref({
  categoryId: '',
  projectId: '',
  from: defaultFrom,
  to: defaultTo
})

const hasActiveFilters = computed(() =>
  filters.value.categoryId || filters.value.projectId
)

const activeFiltersCount = computed(() => {
  let count = 0
  if (filters.value.categoryId) count++
  if (filters.value.projectId) count++
  if (filters.value.from) count++
  if (filters.value.to) count++
  return count
})

// Stats from API (used for KPIs and charts)
const stats = ref(null)

// KPIs from stats API
const totalMonth = computed(() => stats.value?.total_month ?? 0)
const totalYear = computed(() => stats.value?.total_year ?? 0)
const dailyAverage = computed(() => {
  if (!stats.value?.total_month) return 0
  return stats.value.total_month / new Date().getDate()
})


// Chart data from stats API
const categoryColors = [
  '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#14B8A6', '#F97316', '#6366F1'
]

const categoryChartTitle = computed(() =>
  stats.value?.is_subcategory ? 'Spese per Sottocategoria' : 'Spese per Categoria'
)

const hasCategoryData = computed(() => (stats.value?.by_category?.length ?? 0) > 0)

const categoryChartData = computed(() => {
  const items = stats.value?.by_category ?? []
  return {
    labels: items.map(i => i.category_name),
    datasets: [{
      data: items.map(i => i.amount),
      backgroundColor: categoryColors.slice(0, items.length),
      borderWidth: 0
    }]
  }
})

const hasTrendData = computed(() => (stats.value?.trend?.length ?? 0) > 0)

const trendChartTitle = computed(() => {
  switch (stats.value?.granularity) {
    case 'day': return 'Trend Giornaliero'
    case 'quarter': return 'Trend Trimestrale'
    default: return 'Trend Mensile'
  }
})

function abbreviateTrendLabel(label) {
  // "Mar 2025" → "Mar" (show year only on first item or when year changes)
  const parts = label.split(' ')
  if (parts.length === 2) return parts[0] // Return just the month abbreviation
  return label
}

function abbreviateTrendLabels(items) {
  let lastYear = null
  return items.map(i => {
    const parts = i.label.split(' ')
    if (parts.length === 2) {
      const [month, year] = parts
      if (year !== lastYear) {
        lastYear = year
        return `${month} '${year.slice(-2)}`
      }
      return month
    }
    return i.label
  })
}

const trendBarChartData = computed(() => {
  const items = stats.value?.trend ?? []
  return {
    labels: abbreviateTrendLabels(items),
    datasets: [{ label: 'Spese', data: items.map(i => i.amount), backgroundColor: '#3B82F6', borderRadius: 4 }]
  }
})

const trendLineChartData = computed(() => {
  const items = stats.value?.trend ?? []
  return {
    labels: abbreviateTrendLabels(items),
    datasets: [{
      label: 'Spese',
      data: items.map(i => i.amount),
      borderColor: '#3B82F6',
      backgroundColor: 'rgba(59, 130, 246, 0.1)',
      fill: true,
      pointBackgroundColor: '#3B82F6',
      pointBorderColor: '#fff',
      pointBorderWidth: 2
    }]
  }
})

// Methods
async function fetchStats(params = {}) {
  try {
    const { data } = await expensesAPI.stats(params)
    stats.value = data
  } catch (err) {
    console.error('Error fetching stats:', err)
  }
}

async function fetchFiltersData() {
  try {
    const [catRes, projRes] = await Promise.all([
      categoriesAPI.list(),
      projectsAPI.list()
    ])
    categories.value = catRes.data || []
    projects.value = projRes.data?.projects || projRes.data || []
  } catch (err) {
    console.error('Error fetching filter data:', err)
  }
}

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: settingsStore.currency || 'EUR'
  }).format(value || 0)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function buildStatsParams() {
  const params = {}
  if (filters.value.categoryId) params.category_id = filters.value.categoryId
  if (filters.value.projectId) params.project_id = filters.value.projectId
  if (filters.value.from) params.from = filters.value.from
  if (filters.value.to) params.to = filters.value.to
  return params
}

function onPieSliceClick(index) {
  const item = stats.value?.by_category?.[index]
  if (!item || !item.category_id) return
  filters.value.categoryId = item.category_id
  applyFilters()
}

function applyFilters() {
  const params = buildStatsParams()
  expensesStore.fetchExpenses(params)
  fetchStats(params)
}

async function resetFilters() {
  filters.value = { categoryId: '', projectId: '', from: '', to: '' }
  // Fetch stats with full range to discover first expense date → today
  await fetchStats({ all: true })
  // Populate date fields from the returned period (never empty)
  if (stats.value?.period) {
    filters.value.from = stats.value.period.start
    filters.value.to = stats.value.period.end
  }
  expensesStore.fetchExpenses(buildStatsParams())
}

function isOwner(expense) {
  return expense.user_id === authStore.user?.id
}

function isExpenseSettled(expense) {
  if (!expense.is_split || !expense.splits || expense.splits.length === 0) return true
  return expense.splits.every(s => s.is_settled)
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
  applyFilters()
}

function onExpenseUpdated() {
  showEditExpense.value = false
  editingExpense.value = null
  applyFilters()
}

async function deleteExpenseConfirm(id) {
  const ok = await confirm({
    title: 'Elimina spesa',
    message: 'Sei sicuro di voler eliminare questa spesa?',
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (ok) {
    try {
      await expensesStore.deleteExpense(id)
    } catch (err) {
      window.$toast?.error('Errore eliminazione: ' + (err.response?.data?.error || err.message))
    }
  }
}

onMounted(() => {
  fetchFiltersData()
  applyFilters()
})
</script>

<style scoped>
.filter-expand-enter-active,
.filter-expand-leave-active {
  transition: opacity 0.2s ease, max-height 0.25s ease;
  overflow: hidden;
  max-height: 500px;
}
.filter-expand-enter-from,
.filter-expand-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
