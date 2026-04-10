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

    <!-- Pending Join Request / No Property Banner -->
    <Card v-if="!settingsStore.hasProperty" className="p-6">
      <div v-if="settingsStore.pendingJoinRequest" class="text-center space-y-3">
        <div class="text-4xl">⏳</div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Richiesta in attesa</h2>
        <p class="text-gray-600 dark:text-gray-400">
          La tua richiesta di accesso a <strong>{{ settingsStore.pendingJoinRequest.property_name }}</strong>
          è in attesa di approvazione da parte di un amministratore.
        </p>
      </div>
      <div v-else class="text-center space-y-3">
        <div class="text-4xl">🏠</div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Nessuna proprietà</h2>
        <p class="text-gray-600 dark:text-gray-400">Crea una proprietà o unisciti a una esistente per iniziare.</p>
        <Button @click="$router.push('/onboarding')">Configura</Button>
      </div>
    </Card>

    <!-- Main content (only when user has a property) -->
    <template v-if="settingsStore.hasProperty">

    <!-- KPI Cards -->
    <KpiCards
      :monthTotal="totalMonth"
      :periodCount="stats?.count ?? expensesStore.total"
      :dailyAverage="dailyAverage"
      :yearTotal="totalYear"
      :formatCurrency="formatCurrency"
    />

    <!-- Grafici -->
    <DashboardCharts
      :categoryChartTitle="categoryChartTitle"
      :hasCategoryData="hasCategoryData"
      :categoryChartData="categoryChartData"
      :currency="settingsStore.currency"
      :isSubcategory="!!stats?.is_subcategory"
      :trendChartTitle="trendChartTitle"
      :hasTrendData="hasTrendData"
      :trendBarChartData="trendBarChartData"
      :trendLineChartData="trendLineChartData"
      :trendChartType="trendChartType"
      @update:trendChartType="trendChartType = $event"
      @slice-click="onPieSliceClick"
    />

    <!-- Filtri -->
    <DashboardFilters
      :filters="filters"
      :categories="categories"
      :projects="projects"
      :filtersOpen="filtersOpen"
      :hasActiveFilters="hasActiveFilters"
      :activeFiltersCount="activeFiltersCount"
      @update:filters="filters = $event"
      @update:filtersOpen="filtersOpen = $event"
      @apply="applyFilters"
      @reset="resetFilters"
    />

    <!-- Lista Spese Recenti -->
    <RecentExpensesList
      :expenses="expensesStore.expenses"
      :loading="expensesStore.loading"
      :formatCurrency="formatCurrency"
      :formatDate="formatDate"
      :currentUserId="authStore.user?.id"
      @add="showAddExpense = true"
      @edit="editExpense"
      @delete="deleteExpenseConfirm"
    />

    </template>

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
defineOptions({ name: 'DashboardView' })

import { ref, computed, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { categoriesAPI, projectsAPI, expensesAPI } from '@/api/client'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import { useConfirm } from '@/composables/useConfirm'
import Button from '@/components/common/Button.vue'
import Card from '@/components/common/Card.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'
import KpiCards from '@/components/dashboard/KpiCards.vue'
import DashboardCharts from '@/components/dashboard/DashboardCharts.vue'
import DashboardFilters from '@/components/dashboard/DashboardFilters.vue'
import RecentExpensesList from '@/components/dashboard/RecentExpensesList.vue'

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
  return _formatCurrency(value, settingsStore.formatSettings)
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
