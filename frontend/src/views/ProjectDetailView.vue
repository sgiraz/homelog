<template>
  <div class="space-y-4">
    <!-- Back + Header -->
    <div class="flex items-center gap-3">
      <button
        @click="goBack"
        class="p-2 -ml-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        aria-label="Torna ai progetti"
      >
        <svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <div v-if="project" class="flex items-center gap-3 flex-1 min-w-0">
        <div class="text-3xl flex-shrink-0">{{ project.icon || '🏗️' }}</div>
        <div class="min-w-0">
          <h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white truncate">{{ project.name }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ formatDateRange(project.start_date, project.end_date) }}</p>
        </div>
      </div>
      <!-- Actions -->
      <div v-if="project && canManage" class="flex items-center gap-1 flex-shrink-0">
        <button
          @click="showEditModal = true"
          class="p-2.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
          title="Modifica progetto"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        </button>
        <button
          @click="confirmDelete"
          class="p-2.5 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-500 dark:hover:text-red-400"
          title="Elimina progetto"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-gray-600">
      Caricamento progetto...
    </div>

    <!-- Not Found -->
    <div v-else-if="!project" class="text-center py-12">
      <div class="text-6xl mb-4">🔍</div>
      <h3 class="text-xl font-semibold mb-2">Progetto non trovato</h3>
      <Button @click="goBack">Torna ai progetti</Button>
    </div>

    <template v-else>
      <!-- KPI Cards -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <Card class="p-4">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400 mb-1">Budget</div>
          <div class="text-lg sm:text-2xl font-bold text-gray-900 dark:text-white">
            {{ formatCurrency(project.budget) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400 mb-1">Speso</div>
          <div class="text-lg sm:text-2xl font-bold text-blue-600">
            {{ formatCurrency(stats.total_spent) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400 mb-1">Rimanente</div>
          <div :class="[
            'text-lg sm:text-2xl font-bold',
            stats.remaining >= 0 ? 'text-green-600' : 'text-red-600'
          ]">
            {{ formatCurrency(Math.abs(stats.remaining)) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400 mb-1">Completamento</div>
          <div class="text-lg sm:text-2xl font-bold text-purple-600">
            {{ stats.percentage_spent.toFixed(1) }}%
          </div>
        </Card>
      </div>

      <!-- Progress Bar -->
      <div>
        <div class="flex justify-between text-sm mb-2">
          <span class="text-gray-600 dark:text-gray-400">Progresso Budget</span>
          <span :class="[
            'font-medium',
            stats.percentage_spent > 100 ? 'text-red-600' : 'text-gray-900 dark:text-white'
          ]">
            {{ stats.percentage_spent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            :class="[
              'h-3 rounded-full transition-all',
              stats.percentage_spent > 100 ? 'bg-red-600' : 'bg-blue-600'
            ]"
            :style="{ width: Math.min(stats.percentage_spent, 100) + '%' }"
          ></div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="overflow-x-auto -mx-4 sm:mx-0 px-4 sm:px-0 pb-1">
        <div class="flex gap-1 min-w-max sm:min-w-0 sm:flex-wrap">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            @click="activeTab = tab.value"
            :class="[
              'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
              activeTab === tab.value
                ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
            ]"
          >
            <span>{{ tab.icon }}</span>
            <span>{{ tab.label }}</span>
          </button>
        </div>
      </div>

      <!-- Tab: Info -->
      <div v-if="activeTab === 'info'" class="space-y-4">
        <Card v-if="project.description" class="p-4">
          <h4 class="font-medium text-gray-900 dark:text-white mb-2">Descrizione</h4>
          <p class="text-gray-600 dark:text-gray-400">{{ project.description }}</p>
        </Card>

        <Card class="p-4 space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-600 dark:text-gray-400">Status</span>
            <span :class="[
              'px-2 py-1 text-xs rounded-full font-medium',
              getStatusColor(project.status)
            ]">
              {{ getStatusLabel(project.status) }}
            </span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-600 dark:text-gray-400">Inizio</span>
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ _formatDate(project.start_date, settingsStore.dateSettings) }}
            </span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-600 dark:text-gray-400">Fine</span>
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ _formatDate(project.end_date, settingsStore.dateSettings) }}
            </span>
          </div>
        </Card>

        <!-- Members -->
        <Card v-if="project.members?.length > 0" class="p-4">
          <h4 class="font-medium text-gray-900 dark:text-white mb-3">Membri</h4>
          <div class="space-y-2">
            <div
              v-for="member in project.members"
              :key="member.id"
              class="flex items-center gap-3"
            >
              <div class="w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center text-xs font-medium text-purple-700 dark:text-purple-300">
                {{ member.name?.[0]?.toUpperCase() }}
              </div>
              <span class="text-sm text-gray-900 dark:text-white flex-1">{{ member.name }}</span>
              <span :class="[
                'text-xs px-2 py-0.5 rounded-full',
                member.role === 'creator' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' :
                member.role === 'owner' ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' :
                'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
              ]">
                {{ member.role === 'creator' ? 'Creatore' : member.role === 'owner' ? 'Co-proprietario' : 'Membro' }}
              </span>
            </div>
          </div>
        </Card>
      </div>

      <!-- Tab: Expenses -->
      <div v-if="activeTab === 'expenses'" class="space-y-4">
        <!-- Stats + Category Chart (unified card) -->
        <Card v-if="project.expenses?.length > 0" class="p-4 space-y-4">
          <div class="grid grid-cols-4 gap-3 text-center">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Spesa media</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatCurrency(stats.expense_count > 0 ? stats.total_spent / stats.expense_count : 0) }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Budget/giorno</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatCurrency(dailyBudget) }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Giorni rimasti</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ daysRemaining >= 0 ? daysRemaining : 0 }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Nr. spese</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ stats.expense_count }}
              </div>
            </div>
          </div>
          <template v-if="categoryBreakdown.length > 1">
            <div class="border-t border-gray-200 dark:border-gray-700"></div>
            <div>
              <h4 class="font-medium text-gray-900 dark:text-white mb-3">Spese per Categoria</h4>
              <PieChart :chartData="categoryChartData" />
            </div>
          </template>
        </Card>

        <!-- Expense List Header -->
        <div class="flex items-center justify-between">
          <h4 class="font-medium text-gray-900 dark:text-white">Lista Spese</h4>
          <Button size="sm" @click="showAddExpense = true">
            <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Aggiungi Spesa
          </Button>
        </div>

        <div v-if="project.expenses?.length > 0" class="space-y-3">
          <Card
            v-for="expense in sortedExpenses"
            :key="expense.id"
            class="p-4"
          >
            <div class="flex justify-between items-start">
              <div class="min-w-0 flex-1">
                <div class="font-medium text-gray-900 dark:text-white truncate">
                  {{ expense.description }}
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  {{ _formatDate(expense.date, settingsStore.dateSettings) }}
                  <span v-if="expense.category"> · {{ expense.category.name }}</span>
                </div>
              </div>
              <div class="text-lg font-bold text-gray-900 dark:text-white ml-3 flex-shrink-0">
                {{ formatCurrency(expense.amount) }}
              </div>
            </div>
          </Card>
        </div>
        <div v-else class="text-center py-8">
          <div class="text-4xl mb-3">📋</div>
          <p class="text-gray-600 dark:text-gray-400 mb-4">Nessuna spesa associata</p>
        </div>
      </div>
    </template>

    <!-- Add Expense Modal -->
    <AddExpenseModal
      v-if="showAddExpense"
      :project-id="project?.id"
      @close="showAddExpense = false"
      @created="onExpenseCreated"
    />

    <!-- Edit Project Modal -->
    <EditProjectModal
      v-if="showEditModal"
      :project="project"
      @close="showEditModal = false"
      @updated="onProjectUpdated"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'ProjectDetailView' })

import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProjectsStore } from '@/stores/projects'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency, formatDate as _formatDate } from '@/utils/dateFormatter'
import { useConfirm } from '@/composables/useConfirm'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditProjectModal from '@/components/projects/EditProjectModal.vue'
import PieChart from '@/components/charts/PieChart.vue'

const route = useRoute()
const router = useRouter()
const projectsStore = useProjectsStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const project = ref(null)
const loading = ref(true)
const activeTab = ref('expenses')
const showAddExpense = ref(false)
const showEditModal = ref(false)

const defaultStats = { total_budget: 0, total_spent: 0, remaining: 0, percentage_spent: 0, expense_count: 0 }

const stats = computed(() => project.value?.stats || defaultStats)

const canManage = computed(() => {
  if (!project.value) return false
  if (project.value.user_id === authStore.user?.id) return true
  return project.value.members?.some(m => m.id === authStore.user?.id && m.role === 'owner') || false
})

const tabs = computed(() => [
  { value: 'expenses', label: `Spese (${stats.value.expense_count})`, icon: '💰' },
  { value: 'info', label: 'Info', icon: 'ℹ️' }
])

const sortedExpenses = computed(() => {
  if (!project.value?.expenses) return []
  return [...project.value.expenses].sort((a, b) => new Date(b.date) - new Date(a.date))
})

const dailyBudget = computed(() => {
  if (!project.value) return 0
  const start = new Date(project.value.start_date)
  const end = new Date(project.value.end_date)
  const days = Math.ceil((end - start) / (1000 * 60 * 60 * 24))
  return days > 0 ? project.value.budget / days : 0
})

const daysRemaining = computed(() => {
  if (!project.value) return 0
  const end = new Date(project.value.end_date)
  const now = new Date()
  return Math.ceil((end - now) / (1000 * 60 * 60 * 24))
})

const categoryBreakdown = computed(() => {
  if (!project.value?.expenses?.length) return []
  const map = {}
  for (const exp of project.value.expenses) {
    const catId = exp.category_id || 0
    if (!map[catId]) {
      map[catId] = {
        category_id: catId,
        category_name: exp.category?.name || 'Senza categoria',
        category_icon: exp.category?.icon || '📦',
        category_color: exp.category?.color || '#6B7280',
        total: 0,
        count: 0
      }
    }
    map[catId].total += exp.amount
    map[catId].count++
  }
  return Object.values(map).sort((a, b) => b.total - a.total)
})

const categoryChartData = computed(() => {
  const items = categoryBreakdown.value
  return {
    labels: items.map(i => i.category_name),
    datasets: [{
      data: items.map(i => i.total),
      backgroundColor: items.map(i => i.category_color),
      borderWidth: 0
    }]
  }
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatDateRange(start, end) {
  const s = _formatDate(start, settingsStore.dateSettings)
  const e = _formatDate(end, settingsStore.dateSettings)
  return `${s} - ${e}`
}

function getStatusLabel(status) {
  const map = { planned: 'Pianificato', active: 'In Corso', completed: 'Completato', cancelled: 'Annullato' }
  return map[status] || status
}

function getStatusColor(status) {
  const map = {
    planned: 'bg-blue-100 text-blue-700',
    active: 'bg-green-100 text-green-700',
    completed: 'bg-gray-100 text-gray-700',
    cancelled: 'bg-red-100 text-red-700'
  }
  return map[status] || 'bg-gray-100 text-gray-700'
}

function goBack() {
  router.push('/projects')
}

async function loadProject() {
  loading.value = true
  try {
    const data = await projectsStore.fetchProject(route.params.id)
    if (!data.stats) {
      data.stats = { ...defaultStats, total_budget: data.budget || 0, remaining: data.budget || 0 }
    }
    project.value = data
  } catch {
    project.value = null
  } finally {
    loading.value = false
  }
}

async function onExpenseCreated() {
  showAddExpense.value = false
  await loadProject()
}

async function onProjectUpdated() {
  showEditModal.value = false
  await loadProject()
}

async function confirmDelete() {
  const ok = await confirm({
    title: 'Elimina Progetto',
    message: `Eliminare il progetto "${project.value.name}"?`,
    confirmText: 'Elimina',
    confirmVariant: 'danger'
  })
  if (!ok) return

  try {
    await projectsStore.deleteProject(project.value.id)
    window.$toast?.success('Progetto eliminato')
    router.push('/projects')
  } catch (err) {
    window.$toast?.error('Errore: ' + (err.response?.data?.error || err.message))
  }
}

onMounted(() => {
  loadProject()
})
</script>
