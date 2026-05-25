<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl sm:text-3xl font-bold text-ink">{{ t('dashboard.title') }}</h1>
        <p class="text-ink-soft mt-1 text-sm sm:text-base">{{ t('dashboard.subtitle') }}</p>
      </div>
      <Button @click="showAddExpense = true">
        <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">{{ t('dashboard.addExpenseButton') }}</span>
      </Button>
    </div>

    <!-- Pending Join Request / No Property Banner -->
    <Card v-if="!settingsStore.hasProperty" className="p-6">
      <div v-if="settingsStore.pendingJoinRequest" class="text-center space-y-3">
        <div class="text-4xl">⏳</div>
        <h2 class="text-lg font-semibold text-ink">{{ t('dashboard.noProperty.pendingTitle') }}</h2>
        <p class="text-ink-soft">
          <i18n-t keypath="dashboard.noProperty.pendingMessage" tag="span">
            <template #name><strong>{{ settingsStore.pendingJoinRequest.property_name }}</strong></template>
          </i18n-t>
        </p>
      </div>
      <div v-else class="text-center space-y-3">
        <div class="text-4xl">🏠</div>
        <h2 class="text-lg font-semibold text-ink">{{ t('dashboard.noProperty.noneTitle') }}</h2>
        <p class="text-ink-soft">{{ t('dashboard.noProperty.noneDescription') }}</p>
        <Button @click="$router.push('/onboarding')">{{ t('dashboard.noProperty.configureButton') }}</Button>
      </div>
    </Card>

    <!-- Main content (only when user has a property) -->
    <template v-if="settingsStore.hasProperty">

    <!-- Da gestire — brief operativo concreto (prima "cosa fare", poi il riepilogo) -->
    <AttentionPanel
      :items="attentionItems"
      :empty-detail="attentionEmptyDetail"
      @action="onAttentionAction"
    />

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

    <!-- Modal Salda (aperto inline dal pannello "Da gestire") -->
    <SettlementModal
      v-if="showSettle && balanceInfo"
      :balance="balanceInfo.balance"
      :other-member-name="balanceInfo.other_member_name"
      :other-member-id="balanceInfo.other_member_id"
      :current-member-id="balanceInfo.current_member_id"
      :property-id="settingsStore.householdPropertyId"
      @close="showSettle = false"
      @created="onSettled"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'DashboardView' })

import { ref, computed, onMounted, onActivated, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { categoriesAPI, projectsAPI, expensesAPI, utilitiesAPI, balanceAPI } from '@/api/client'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import { useConfirm } from '@/composables/useConfirm'
import Button from '@/components/common/Button.vue'
import Card from '@/components/common/Card.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'
import KpiCards from '@/components/dashboard/KpiCards.vue'
import AttentionPanel from '@/components/dashboard/AttentionPanel.vue'
import DashboardCharts from '@/components/dashboard/DashboardCharts.vue'
import DashboardFilters from '@/components/dashboard/DashboardFilters.vue'
import RecentExpensesList from '@/components/dashboard/RecentExpensesList.vue'
import SettlementModal from '@/components/balance/SettlementModal.vue'

const { t } = useI18n()
const expensesStore = useExpensesStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const showAddExpense = ref(false)
const showEditExpense = ref(false)
const showSettle = ref(false)
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
  stats.value?.is_subcategory ? t('dashboard.charts.categoryBySubcategory') : t('dashboard.charts.categoryByCategory')
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
    case 'day': return t('dashboard.charts.trendDaily')
    case 'quarter': return t('dashboard.charts.trendQuarterly')
    default: return t('dashboard.charts.trendMonthly')
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
    datasets: [{ label: t('dashboard.charts.datasetLabel'), data: items.map(i => i.amount), backgroundColor: '#3B82F6', borderRadius: 4 }]
  }
})

const trendLineChartData = computed(() => {
  const items = stats.value?.trend ?? []
  return {
    labels: abbreviateTrendLabels(items),
    datasets: [{
      label: t('dashboard.charts.datasetLabel'),
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

// ── "Casa oggi" brief data source ──
// One utilities.list call carries embedded bills + readings; balance is a
// single endpoint; projects are already fetched for the filters. No N+1.
const utilities = ref([])
const balanceInfo = ref(null)

const MS_PER_DAY = 24 * 60 * 60 * 1000
const METERED_TYPES = ['electricity', 'gas', 'water', 'waste']

async function fetchActionables() {
  // The brief is scoped to the current property. householdPropertyId is
  // populated asynchronously at startup, so bail out until it's known rather
  // than calling /utilities unscoped (which would return cross-property data
  // and skip the balance). The watcher below re-runs us once it's set.
  const propertyId = settingsStore.householdPropertyId
  if (!propertyId) return
  try {
    const { data } = await utilitiesAPI.list({ property_id: propertyId })
    utilities.value = data || []
  } catch {
    utilities.value = []
  }
  try {
    const { data } = await balanceAPI.get(propertyId)
    balanceInfo.value = data
  } catch {
    balanceInfo.value = null
  }
}

// Localized service name; fall back to the raw type if no translation.
function serviceTypeName(type) {
  const key = `utilities.utilityTypes.${type}`
  const label = t(key)
  return label === key ? type : label
}

// Brief title for a service row: "Enel · Luce" (or just the type if no provider).
function serviceLabel(u) {
  const typeName = serviceTypeName(u.type)
  return u.provider ? `${u.provider} · ${typeName}` : typeName
}

// Concrete, per-item "to handle" brief. Each row names the specific service,
// shows its amount/date, and links to the most specific destination (or opens
// a modal). Priority: overdue bills → due-soon → open balance → missing
// readings → projects over budget. Capped to 5 (panel shows 3 on mobile).
const attentionItems = computed(() => {
  const items = []
  const todayStart = new Date()
  todayStart.setHours(0, 0, 0, 0)

  // Calendar-day difference from today (local). Reads the date's Y-M-D
  // directly so a date-only / UTC-midnight value isn't shifted across the day
  // boundary by timezone. <0 = past, 0 = today, >0 = future.
  const dayDiff = (dateStr) => {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(dateStr))
    const due = m
      ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
      : new Date(dateStr)
    due.setHours(0, 0, 0, 0)
    return Math.round((due - todayStart) / MS_PER_DAY)
  }

  const bills = []
  const readings = []

  utilities.value.forEach((u) => {
    (u.bills || []).forEach((b) => {
      if (b.is_paid) return
      const diff = dayDiff(b.due_date)
      if (diff < 0) bills.push({ u, b, overdue: true, days: -diff })
      else if (diff <= 7) bills.push({ u, b, overdue: false, days: diff })
    })
    if (METERED_TYPES.includes(u.type)) {
      const last = u.readings && u.readings[0]
      if (!last) readings.push({ u, days: null })
      else {
        const ago = -dayDiff(last.reading_date)
        if (ago >= 30) readings.push({ u, days: ago })
      }
    }
  })

  // Overdue first (most overdue first), then due-soon (soonest first).
  bills.sort((a, b) => {
    if (a.overdue !== b.overdue) return a.overdue ? -1 : 1
    return a.overdue ? b.days - a.days : a.days - b.days
  })

  bills.forEach(({ u, b, overdue, days }) => {
    const when = overdue
      ? t(`dashboard.attention.overdueBy_${days === 1 ? 'one' : 'other'}`, { n: days })
      : days <= 0
        ? t('dashboard.attention.dueToday')
        : t(`dashboard.attention.dueIn_${days === 1 ? 'one' : 'other'}`, { n: days })
    items.push({
      key: `bill-${b.id}`,
      tone: overdue ? 'danger' : 'warn',
      icon: 'bill',
      title: serviceLabel(u),
      detail: `${when} · ${formatCurrency(b.amount_total || 0)}`,
      to: `/utilities/${u.id}`,
      action: t('dashboard.attention.open'),
    })
  })

  const bal = balanceInfo.value?.balance || 0
  if (Math.abs(bal) >= 0.01) {
    const name = balanceInfo.value?.other_member_name || ''
    items.push({
      key: 'balance',
      tone: bal > 0 ? 'positive' : 'accent',
      icon: 'balance',
      title: bal > 0 ? t('dashboard.attention.owedToYou', { name }) : t('dashboard.attention.youOwe', { name }),
      detail: formatCurrency(Math.abs(bal)),
      action: t('dashboard.attention.settle'),
    })
  }

  readings.forEach(({ u, days }) => {
    items.push({
      key: `reading-${u.id}`,
      tone: 'info',
      icon: 'reading',
      title: serviceLabel(u),
      detail: days === null
        ? t('dashboard.attention.readingMissing')
        : t(`dashboard.attention.readingLast_${days === 1 ? 'one' : 'other'}`, { n: days }),
      to: `/utilities/${u.id}`,
      action: t('dashboard.attention.logReading'),
    })
  })

  ;(projects.value || []).forEach((p) => {
    if (p.status === 'active' && (p.stats?.percentage_spent ?? 0) > 100) {
      items.push({
        key: `project-${p.id}`,
        tone: 'accent',
        icon: 'project',
        title: p.name,
        detail: t('dashboard.attention.overBudget', { pct: Math.round(p.stats.percentage_spent) }),
        to: `/projects/${p.id}`,
        action: t('dashboard.attention.open'),
      })
    }
  })

  return items.slice(0, 5)
})

// Micro-line for the calm empty state: last expense + last reading dates.
const attentionEmptyDetail = computed(() => {
  const parts = []
  const lastExpense = expensesStore.expenses?.[0]
  if (lastExpense?.date) {
    parts.push(t('dashboard.attention.lastExpense', { date: formatDate(lastExpense.date) }))
  }
  let lastReadingStr = null
  let lastReadingTime = 0
  utilities.value.forEach((u) => {
    const r = u.readings && u.readings[0]
    if (r?.reading_date) {
      const ts = new Date(r.reading_date).getTime()
      if (ts > lastReadingTime) {
        lastReadingTime = ts
        lastReadingStr = r.reading_date
      }
    }
  })
  if (lastReadingStr) {
    parts.push(t('dashboard.attention.lastReading', { date: formatDate(lastReadingStr) }))
  }
  return parts.join(' · ')
})

// Items without a `to` (the open balance) ask the panel to act inline.
function onAttentionAction(item) {
  if (item.key === 'balance') showSettle.value = true
}

function onSettled() {
  showSettle.value = false
  fetchActionables()
  applyFilters()
}

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
  fetchActionables()
}

function onExpenseUpdated() {
  showEditExpense.value = false
  editingExpense.value = null
  applyFilters()
  fetchActionables()
}

async function deleteExpenseConfirm(id) {
  const ok = await confirm({
    title: t('expenses.deleteConfirmTitle'),
    message: t('expenses.deleteConfirmMessage'),
    confirmText: t('expenses.deleteConfirmAction'),
    variant: 'danger'
  })
  if (ok) {
    try {
      await expensesStore.deleteExpense(id)
      fetchActionables()
    } catch (err) {
      window.$toast?.error(t('expenses.deleteError', { error: err.response?.data?.error || err.message }))
    }
  }
}

onMounted(() => {
  fetchFiltersData()
  applyFilters()
})

// "Casa oggi" is scoped to the current property, whose id arrives
// asynchronously after mount. Fetch as soon as it's known (covers first load),
// and re-fetch whenever it changes.
watch(
  () => settingsStore.householdPropertyId,
  (id) => { if (id) fetchActionables() }
)

// keep-alive caches this view, so onMounted won't re-run on return. Refresh on
// re-activation (and on initial mount, when the id is already known) so paid
// bills / new readings / settled balances drop off without a hard reload.
onActivated(() => {
  if (settingsStore.householdPropertyId) fetchActionables()
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
