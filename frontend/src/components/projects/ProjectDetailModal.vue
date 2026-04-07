<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
       @click.self="$emit('close')">
    <Card class="w-full max-w-4xl p-6 max-h-[90vh] overflow-y-auto">
      <!-- Header -->
      <div class="flex items-start justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="text-4xl">{{ projectData.icon || '🏗️' }}</div>
          <div>
            <h3 class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ projectData.name }}
            </h3>
            <p class="text-sm text-gray-600 dark:text-gray-400">
              {{ formatDateRange(projectData.start_date, projectData.end_date) }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Button v-if="canManage" variant="secondary" @click="showEditModal = true">
            Modifica
          </Button>
          <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- KPI Cards -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <Card class="p-4">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Budget Totale</div>
          <div class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ formatCurrency(projectData.budget) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Speso</div>
          <div class="text-2xl font-bold text-blue-600">
            {{ formatCurrency(projectData.stats.total_spent) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Rimanente</div>
          <div :class="[
            'text-2xl font-bold',
            projectData.stats.remaining >= 0 ? 'text-green-600' : 'text-red-600'
          ]">
            {{ formatCurrency(Math.abs(projectData.stats.remaining)) }}
          </div>
        </Card>

        <Card class="p-4">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Completamento</div>
          <div class="text-2xl font-bold text-purple-600">
            {{ projectData.stats.percentage_spent.toFixed(1) }}%
          </div>
        </Card>
      </div>

      <!-- Progress Bar -->
      <div class="mb-6">
        <div class="flex justify-between text-sm mb-2">
          <span class="text-gray-600 dark:text-gray-400">Progresso Budget</span>
          <span :class="[
            'font-medium',
            projectData.stats.percentage_spent > 100 ? 'text-red-600' : 'text-gray-900 dark:text-white'
          ]">
            {{ projectData.stats.percentage_spent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            :class="[
              'h-3 rounded-full transition-all',
              projectData.stats.percentage_spent > 100 ? 'bg-red-600' : 'bg-blue-600'
            ]"
            :style="{ width: Math.min(projectData.stats.percentage_spent, 100) + '%' }"
          ></div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="border-b border-gray-200 dark:border-gray-700 mb-6">
        <div class="flex gap-4">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            @click="activeTab = tab.value"
            :class="[
              'pb-3 px-2 text-sm font-medium transition-colors',
              activeTab === tab.value
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
            ]"
          >
            {{ tab.label }}
          </button>
        </div>
      </div>

      <!-- Tab Content: Info -->
      <div v-if="activeTab === 'info'" class="space-y-4">
        <div v-if="projectData.description">
          <h4 class="font-medium text-gray-900 dark:text-white mb-2">Descrizione</h4>
          <p class="text-gray-600 dark:text-gray-400">{{ projectData.description }}</p>
        </div>

        <div>
          <h4 class="font-medium text-gray-900 dark:text-white mb-2">Status</h4>
          <span :class="[
            'inline-block px-3 py-1 rounded-full text-sm font-medium',
            getStatusColor(projectData.status)
          ]">
            {{ getStatusLabel(projectData.status) }}
          </span>
        </div>

        <div>
          <h4 class="font-medium text-gray-900 dark:text-white mb-2">Date</h4>
          <div class="text-sm text-gray-600 dark:text-gray-400">
            <div>Inizio: {{ _formatDate(projectData.start_date, settingsStore.dateSettings) }}</div>
            <div>Fine: {{ _formatDate(projectData.end_date, settingsStore.dateSettings) }}</div>
          </div>
        </div>

        <div v-if="canManage" class="pt-4">
          <Button variant="secondary" @click="confirmDelete" class="text-red-600">
            Elimina Progetto
          </Button>
        </div>
      </div>

      <!-- Tab Content: Expenses -->
      <div v-if="activeTab === 'expenses'">
        <div v-if="projectData.expenses && projectData.expenses.length > 0" class="space-y-3">
          <div
            v-for="expense in projectData.expenses"
            :key="expense.id"
            class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg"
          >
            <div class="flex justify-between items-start">
              <div>
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ expense.description }}
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  {{ _formatDate(expense.date, settingsStore.dateSettings) }}
                  <span v-if="expense.category"> - {{ expense.category.name }}</span>
                </div>
              </div>
              <div class="text-lg font-bold text-gray-900 dark:text-white">
                {{ formatCurrency(expense.amount) }}
              </div>
            </div>
          </div>
        </div>
        <div v-else class="text-center py-8 text-gray-600 dark:text-gray-400">
          Nessuna spesa associata a questo progetto
        </div>
      </div>

      <!-- Tab Content: Stats -->
      <div v-if="activeTab === 'stats'" class="space-y-4">
        <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <div class="text-sm text-gray-700 dark:text-gray-300 space-y-2">
            <div class="flex justify-between">
              <span>Numero spese:</span>
              <span class="font-medium">{{ projectData.stats.expense_count }}</span>
            </div>
            <div class="flex justify-between">
              <span>Spesa media:</span>
              <span class="font-medium">
                {{ formatCurrency(projectData.stats.expense_count > 0
                  ? projectData.stats.total_spent / projectData.stats.expense_count
                  : 0) }}
              </span>
            </div>
            <div class="flex justify-between">
              <span>Budget giornaliero previsto:</span>
              <span class="font-medium">
                {{ formatCurrency(calculateDailyBudget()) }}/giorno
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Edit Modal (nested) -->
      <EditProjectModal
        v-if="showEditModal"
        :project="projectData"
        @close="showEditModal = false"
        @updated="handleUpdated"
      />
    </Card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useProjectsStore } from '@/stores/projects'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency, formatDate as _formatDate } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import EditProjectModal from './EditProjectModal.vue'

const props = defineProps({
  project: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated', 'deleted'])
const projectsStore = useProjectsStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const projectData = ref(props.project)
const activeTab = ref('info')
const showEditModal = ref(false)

const canManage = computed(() => {
  const p = projectData.value
  if (p.user_id === authStore.user?.id) return true
  return p.members?.some(m => m.id === authStore.user?.id && m.role === 'owner') || false
})

const tabs = [
  { value: 'info', label: 'Info' },
  { value: 'expenses', label: `Spese (${props.project.stats?.expense_count || 0})` },
  { value: 'stats', label: 'Statistiche' }
]

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
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

function formatDateRange(start, end) {
  const s = _formatDate(start, settingsStore.dateSettings)
  const e = _formatDate(end, settingsStore.dateSettings)
  return `${s} - ${e}`
}

function calculateDailyBudget() {
  const start = new Date(projectData.value.start_date)
  const end = new Date(projectData.value.end_date)
  const days = Math.ceil((end - start) / (1000 * 60 * 60 * 24))
  return days > 0 ? projectData.value.budget / days : 0
}

async function confirmDelete() {
  if (!confirm(`Eliminare il progetto "${projectData.value.name}"?`)) {
    return
  }

  try {
    await projectsStore.deleteProject(projectData.value.id)
    emit('deleted')
    emit('close')
  } catch (err) {
    alert('Errore: ' + (err.response?.data?.error || err.message))
  }
}

function handleUpdated() {
  showEditModal.value = false
  emit('updated')
  emit('close')
}
</script>
