<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white">Spese</h1>
      <Button v-if="activeTab === 'lista'" @click="showAddExpense = true">
        <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">Aggiungi Spesa</span>
      </Button>
    </div>

    <!-- Tab bar: Lista / Bilancio -->
    <div class="flex gap-1">
      <button
        v-for="tab in expenseTabs"
        :key="tab.id"
        @click="activeTab = tab.id"
        :class="[
          'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
          activeTab === tab.id
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
        ]"
      >
        <span>{{ tab.icon }}</span>
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <!-- Tab: Bilancio -->
    <BalanceSection v-if="activeTab === 'bilancio'" />

    <!-- Tab: Lista -->
    <template v-if="activeTab === 'lista'">

    <!-- Filtri -->
    <Card class="p-4">
      <!-- Mobile: filtri collassabili -->
      <div class="sm:hidden">
        <div class="flex items-center justify-between">
          <button
            @click="filtersOpen = !filtersOpen"
            class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300"
            :aria-expanded="filtersOpen"
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
          <div class="flex items-center gap-2">
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ expensesStore.total > 0 ? `${expensesStore.expenses.length} / ${expensesStore.total}` : '' }}
            </span>
            <Button v-if="hasActiveFilters" @click="resetFilters" variant="secondary" size="sm">
              Reset
            </Button>
          </div>
        </div>

        <!-- Collapsible filter panel (mobile) -->
        <Transition name="filter-expand">
          <div v-if="filtersOpen" class="mt-3 space-y-3 border-t border-gray-100 dark:border-gray-700 pt-3">
            <!-- Search -->
            <div class="relative">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0" />
              </svg>
              <input
                v-model="filters.search"
                @input="onFiltersChanged"
                type="search"
                placeholder="Cerca..."
                class="w-full pl-9 pr-4 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div class="grid grid-cols-2 gap-2">
              <select
                v-model="filters.categoryId"
                @change="onFiltersChanged"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Tutte categorie</option>
                <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.icon }} {{ cat.name }}</option>
              </select>
              <select
                v-model="filters.projectId"
                @change="onFiltersChanged"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Tutti progetti</option>
                <option v-for="proj in projects" :key="proj.id" :value="proj.id">{{ proj.icon }} {{ proj.name }}</option>
              </select>
              <input
                v-model="filters.from"
                @change="onFiltersChanged"
                type="date"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                v-model="filters.to"
                @change="onFiltersChanged"
                type="date"
                class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <select
              v-model="sortOption"
              class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="date_desc">Data ↓</option>
              <option value="date_asc">Data ↑</option>
              <option value="amount_desc">Importo ↓</option>
              <option value="amount_asc">Importo ↑</option>
              <option value="desc_asc">Descrizione A-Z</option>
            </select>
          </div>
        </Transition>
      </div>

      <!-- Desktop: filtri sempre visibili in riga -->
      <div class="hidden sm:block space-y-3">
        <div class="relative">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0" />
          </svg>
          <input
            v-model="filters.search"
            @input="onFiltersChanged"
            type="text"
            placeholder="Cerca per descrizione..."
            class="w-full pl-9 pr-4 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">Categoria:</label>
            <select
              v-model="filters.categoryId"
              @change="onFiltersChanged"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Tutte</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.icon }} {{ cat.name }}</option>
            </select>
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">Progetto:</label>
            <select
              v-model="filters.projectId"
              @change="onFiltersChanged"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Tutti</option>
              <option v-for="proj in projects" :key="proj.id" :value="proj.id">{{ proj.icon }} {{ proj.name }}</option>
            </select>
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400">Da:</label>
            <input
              v-model="filters.from"
              @change="onFiltersChanged"
              type="date"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400">A:</label>
            <input
              v-model="filters.to"
              @change="onFiltersChanged"
              type="date"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div class="flex items-center gap-2 ml-auto">
            <label class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">Ordina:</label>
            <select
              v-model="sortOption"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="date_desc">Data ↓</option>
              <option value="date_asc">Data ↑</option>
              <option value="amount_desc">Importo ↓</option>
              <option value="amount_asc">Importo ↑</option>
              <option value="desc_asc">Descrizione A-Z</option>
            </select>
          </div>

          <Button v-if="hasActiveFilters" @click="resetFilters" variant="secondary" size="sm" class="text-sm">
            Reset
          </Button>
        </div>

        <div v-if="hasActiveFilters || expensesStore.total > 0" class="text-xs text-gray-500 dark:text-gray-400">
          {{ expensesStore.expenses.length }} mostrate
          <span v-if="expensesStore.total > 0"> di {{ expensesStore.total }}</span>
          <span v-if="filters.projectId"> · Progetto: {{ selectedProjectName }}</span>
          <span v-if="filters.categoryId"> · Categoria: {{ selectedCategoryName }}</span>
        </div>
      </div>
    </Card>

    <!-- Lista spese -->
    <Card class="p-4 sm:p-6">
      <div v-if="expensesStore.loading && expensesStore.expenses.length === 0" class="text-center py-8 text-gray-600 dark:text-gray-400">
        Caricamento...
      </div>

      <div v-else-if="expensesStore.expenses.length === 0" class="text-center py-8 text-gray-600 dark:text-gray-400">
        <span v-if="hasActiveFilters">Nessuna spesa corrisponde ai filtri applicati.</span>
        <span v-else>Nessuna spesa registrata.</span>
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="expense in expensesStore.expenses"
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
                <span
                  v-if="expense.is_split"
                  class="px-2 py-0.5 text-xs rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300"
                >
                  Split
                </span>
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
                <span
                  v-if="expense.project"
                  class="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/40 text-purple-700 dark:text-purple-300 rounded text-xs"
                >
                  {{ expense.project.icon }} {{ expense.project.name }}
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
              <div v-if="expense.is_split && expense.splits?.length" class="text-xs text-gray-500 dark:text-gray-400">
                ({{ formatCurrency(expense.splits[0]?.amount || 0) }} a testa)
              </div>
              <div v-if="expense.bill_id" class="text-xs text-orange-600 dark:text-orange-400 mt-1">
                Da bolletta
              </div>
              <!-- Actions: on mobile always visible, on desktop hover -->
              <div
                v-if="isOwner(expense) && !expense.bill_id"
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
                  v-if="!(expense.is_split && isExpenseSettled(expense))"
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

        <!-- Infinite scroll sentinel + loading indicator -->
        <div ref="sentinel" class="py-2 flex justify-center">
          <svg
            v-if="expensesStore.loading && expensesStore.expenses.length > 0"
            class="w-5 h-5 animate-spin text-blue-500"
            fill="none" viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
          </svg>
          <span
            v-else-if="!expensesStore.hasMore && expensesStore.expenses.length > 0 && expensesStore.total > 0"
            class="text-xs text-gray-400 dark:text-gray-500"
          >
            Tutte le {{ expensesStore.total }} spese mostrate
          </span>
        </div>
      </div>
    </Card>

    </template>

    <AddExpenseModal
      v-if="showAddExpense"
      @close="showAddExpense = false"
      @created="onExpenseCreated"
    />

    <EditExpenseModal
      v-if="showEditExpense && editingExpense"
      :expense="editingExpense"
      @close="showEditExpense = false; editingExpense = null"
      @updated="onExpenseUpdated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import { categoriesAPI, projectsAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'
import BalanceSection from '@/components/balance/BalanceSection.vue'

const route = useRoute()
const expensesStore = useExpensesStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const expenseTabs = [
  { id: 'lista',    label: 'Lista',    icon: '📋' },
  { id: 'bilancio', label: 'Bilancio', icon: '⚖️' },
]

const activeTab = ref(route.query.tab === 'bilancio' ? 'bilancio' : 'lista')
const showAddExpense = ref(false)
const showEditExpense = ref(false)
const editingExpense = ref(null)
const categories = ref([])
const projects = ref([])
const filtersOpen = ref(false)

const filters = ref({
  search: '',
  categoryId: '',
  projectId: '',
  from: '',
  to: ''
})

const sortOption = ref('date_desc')

// Current filters snapshot for infinite scroll
const currentFilters = ref({})

// Watch sort changes → reset and refetch server-side
watch(sortOption, () => onFiltersChanged())

const hasActiveFilters = computed(() =>
  filters.value.search || filters.value.categoryId || filters.value.projectId ||
  filters.value.from || filters.value.to
)

const activeFiltersCount = computed(() => {
  let count = 0
  if (filters.value.search) count++
  if (filters.value.categoryId) count++
  if (filters.value.projectId) count++
  if (filters.value.from) count++
  if (filters.value.to) count++
  return count
})

const selectedCategoryName = computed(() => {
  const cat = categories.value.find(c => c.id === filters.value.categoryId)
  return cat ? `${cat.icon} ${cat.name}` : ''
})

const selectedProjectName = computed(() => {
  const proj = projects.value.find(p => p.id === filters.value.projectId)
  return proj ? `${proj.icon || ''} ${proj.name}` : ''
})

function buildParams() {
  const params = { sort: sortOption.value }
  if (filters.value.search) params.search = filters.value.search
  if (filters.value.categoryId) params.category_id = filters.value.categoryId
  if (filters.value.projectId) params.project_id = filters.value.projectId
  if (filters.value.from) params.from = filters.value.from
  if (filters.value.to) params.to = filters.value.to
  return params
}

function onFiltersChanged() {
  const params = buildParams()
  currentFilters.value = params
  expensesStore.fetchExpenses(params, { page: 1 })
}

function applyFilters() {
  onFiltersChanged()
}

function resetFilters() {
  filters.value = { search: '', categoryId: '', projectId: '', from: '', to: '' }
  const params = { sort: sortOption.value }
  currentFilters.value = params
  expensesStore.fetchExpenses(params, { page: 1 })
}

async function loadMore() {
  await expensesStore.fetchMore(currentFilters.value)
}

// ── Infinite scroll ──────────────────────────────────────────────────────────
const sentinel = ref(null)
let observer = null

function setupIntersectionObserver() {
  if (!sentinel.value) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) loadMore()
    },
    { rootMargin: '200px' } // trigger 200px before sentinel enters viewport
  )
  observer.observe(sentinel.value)
}

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
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
  return expense.splits
    .filter(s => s.member_id !== expense.paid_by_member_id && s.member)
    .map(s => s.member.name)
    .join(', ')
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

onMounted(async () => {
  fetchFiltersData()
  const params = { sort: sortOption.value }
  currentFilters.value = params
  await expensesStore.fetchExpenses(params, { page: 1 })
  setupIntersectionObserver()
})

onUnmounted(() => {
  observer?.disconnect()
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
