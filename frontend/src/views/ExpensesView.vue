<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Tutte le Spese</h1>
      <Button @click="showAddExpense = true">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Aggiungi Spesa
      </Button>
    </div>

    <!-- Filtri e Ricerca -->
    <Card class="p-4">
      <div class="space-y-3">
        <!-- Search bar -->
        <div class="relative">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0" />
          </svg>
          <input
            v-model="filters.search"
            @input="applyFilters"
            type="text"
            placeholder="Cerca per descrizione..."
            class="w-full pl-9 pr-4 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <!-- Filter row -->
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">Categoria:</label>
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
            <label class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">Progetto:</label>
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
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600 dark:text-gray-400">A:</label>
            <input
              v-model="filters.to"
              @change="applyFilters"
              type="date"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <!-- Sort -->
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

          <Button v-if="hasActiveFilters" @click="resetFilters" variant="secondary" class="text-sm">
            Reset
          </Button>
        </div>

        <!-- Active filter summary -->
        <div v-if="hasActiveFilters" class="text-xs text-gray-500 dark:text-gray-400">
          {{ expensesStore.expenses.length }} risultati
          <span v-if="filters.projectId"> · Progetto: {{ selectedProjectName }}</span>
          <span v-if="filters.categoryId"> · Categoria: {{ selectedCategoryName }}</span>
        </div>
      </div>
    </Card>

    <Card class="p-6">
      <div v-if="expensesStore.loading" class="text-center py-8 text-gray-600 dark:text-gray-400">
        Caricamento...
      </div>

      <div v-else-if="expensesStore.expenses.length === 0" class="text-center py-8 text-gray-600 dark:text-gray-400">
        <span v-if="hasActiveFilters">Nessuna spesa corrisponde ai filtri applicati.</span>
        <span v-else>Nessuna spesa registrata.</span>
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="expense in sortedExpenses"
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
                <span
                  v-if="expense.project"
                  class="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/40 text-purple-700 dark:text-purple-300 rounded text-xs"
                >
                  {{ expense.project.icon }} {{ expense.project.name }}
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
                class="flex gap-2 justify-end mt-1 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <button
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
import { ref, computed, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import { categoriesAPI, projectsAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'

const expensesStore = useExpensesStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const showAddExpense = ref(false)
const showEditExpense = ref(false)
const editingExpense = ref(null)
const categories = ref([])
const projects = ref([])

const filters = ref({
  search: '',
  categoryId: '',
  projectId: '',
  from: '',
  to: ''
})

const sortOption = ref('date_desc')

const sortedExpenses = computed(() => {
  const list = [...expensesStore.expenses]
  switch (sortOption.value) {
    case 'date_asc':
      return list.sort((a, b) => new Date(a.date) - new Date(b.date))
    case 'date_desc':
      return list.sort((a, b) => new Date(b.date) - new Date(a.date))
    case 'amount_desc':
      return list.sort((a, b) => b.amount - a.amount)
    case 'amount_asc':
      return list.sort((a, b) => a.amount - b.amount)
    case 'desc_asc':
      return list.sort((a, b) => (a.description || '').localeCompare(b.description || ''))
    default:
      return list
  }
})

const hasActiveFilters = computed(() =>
  filters.value.search || filters.value.categoryId || filters.value.projectId ||
  filters.value.from || filters.value.to
)

const selectedCategoryName = computed(() => {
  const cat = categories.value.find(c => c.id === filters.value.categoryId)
  return cat ? `${cat.icon} ${cat.name}` : ''
})

const selectedProjectName = computed(() => {
  const proj = projects.value.find(p => p.id === filters.value.projectId)
  return proj ? `${proj.icon || ''} ${proj.name}` : ''
})

function applyFilters() {
  const params = {}
  if (filters.value.search) params.search = filters.value.search
  if (filters.value.categoryId) params.category_id = filters.value.categoryId
  if (filters.value.projectId) params.project_id = filters.value.projectId
  if (filters.value.from) params.from = filters.value.from
  if (filters.value.to) params.to = filters.value.to
  expensesStore.fetchExpenses(params)
}

function resetFilters() {
  filters.value = { search: '', categoryId: '', projectId: '', from: '', to: '' }
  expensesStore.fetchExpenses()
}

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(value || 0)
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
  if (confirm('Sei sicuro di voler eliminare questa spesa?')) {
    try {
      await expensesStore.deleteExpense(id)
    } catch (err) {
      alert('Errore eliminazione: ' + (err.response?.data?.error || err.message))
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

onMounted(() => {
  fetchFiltersData()
  expensesStore.fetchExpenses()
})
</script>
