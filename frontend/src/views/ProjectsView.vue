<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Progetti</h1>
      <Button @click="showAddModal = true">
        <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">Nuovo Progetto</span>
      </Button>
    </div>

    <!-- Filters -->
    <div class="overflow-x-auto -mx-4 sm:mx-0 px-4 sm:px-0 pb-1">
      <div class="flex gap-1 min-w-max sm:min-w-0 sm:flex-wrap">
        <button
          v-for="status in statuses"
          :key="status.value"
          @click="selectedStatus = status.value"
          :class="[
            'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
            selectedStatus === status.value
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
          ]"
        >
          <span>{{ status.icon }}</span>
          <span>{{ status.label }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="projectsStore.loading" class="text-center py-12 text-gray-600">
      Caricamento progetti...
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredProjects.length === 0" class="text-center py-12">
      <div class="text-6xl mb-4">🏗️</div>
      <h3 class="text-xl font-semibold mb-2">
        {{ selectedStatus === '' ? 'Nessun progetto' : 'Nessun progetto ' + statuses.find(s => s.value === selectedStatus)?.label.toLowerCase() }}
      </h3>
      <p class="text-gray-600 mb-6">
        Crea progetti per tracciare lavori casa, ristrutturazioni, eventi con budget dedicato
      </p>
      <Button @click="showAddModal = true">
        + Crea Primo Progetto
      </Button>
    </div>

    <!-- Projects Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card
        v-for="project in filteredProjects"
        :key="project.id"
        class="p-6 cursor-pointer hover:shadow-lg transition-shadow"
        @click="viewProject(project)"
      >
        <!-- Header -->
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="text-3xl">{{ project.icon || '🏗️' }}</div>
            <div>
              <h3 class="text-lg font-bold text-gray-900 dark:text-white">
                {{ project.name }}
              </h3>
              <p class="text-sm text-gray-600 dark:text-gray-400">
                {{ formatDateRange(project.start_date, project.end_date) }}
              </p>
            </div>
          </div>

            <!-- Status + Shared badges -->
          <div class="flex flex-col items-end gap-1">
            <span :class="[
              'px-2 py-1 text-xs rounded-full font-medium',
              getStatusColor(project.status)
            ]">
              {{ getStatusLabel(project.status) }}
            </span>
            <span v-if="!isOwner(project)" class="px-2 py-0.5 text-xs rounded-full bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300 font-medium">
              Condiviso
            </span>
          </div>
        </div>

        <!-- Description -->
        <p v-if="project.description" class="text-sm text-gray-600 dark:text-gray-400 mb-4 line-clamp-2">
          {{ project.description }}
        </p>

        <!-- Budget Progress -->
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-gray-600 dark:text-gray-400">Budget:</span>
            <span class="font-medium">{{ formatCurrency(project.budget) }}</span>
          </div>

          <div class="flex justify-between text-sm">
            <span class="text-gray-600 dark:text-gray-400">Speso:</span>
            <span :class="[
              'font-medium',
              (project.stats?.total_spent || 0) > project.budget ? 'text-red-600' : 'text-gray-900 dark:text-white'
            ]">
              {{ formatCurrency(project.stats?.total_spent || 0) }}
            </span>
          </div>

          <!-- Progress Bar -->
          <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
            <div
              :class="[
                'h-2.5 rounded-full transition-all',
                (project.stats?.percentage_spent || 0) > 100 ? 'bg-red-600' : 'bg-blue-600'
              ]"
              :style="{ width: Math.min(project.stats?.percentage_spent || 0, 100) + '%' }"
            ></div>
          </div>

          <div class="flex justify-between text-xs text-gray-600 dark:text-gray-400">
            <span>{{ (project.stats?.percentage_spent || 0).toFixed(1) }}% utilizzato</span>
            <span>
              {{ (project.stats?.remaining ?? 0) >= 0 ? 'Rimangono' : 'Sforato' }}
              {{ formatCurrency(Math.abs(project.stats?.remaining ?? 0)) }}
            </span>
          </div>
        </div>

        <!-- Footer Stats -->
        <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 flex justify-between items-center text-sm">
          <span class="text-gray-600 dark:text-gray-400">
            {{ project.stats?.expense_count || 0 }} spese
          </span>
          <div class="flex items-center gap-2">
            <!-- Shared member avatars -->
            <div v-if="project.shared_with?.length > 0" class="flex -space-x-1">
              <div
                v-for="user in project.shared_with.slice(0, 3)"
                :key="user.id"
                class="w-6 h-6 rounded-full bg-purple-100 dark:bg-purple-900 border-2 border-white dark:border-gray-800 flex items-center justify-center text-xs font-medium text-purple-700 dark:text-purple-300"
                :title="user.name"
              >
                {{ user.name?.[0]?.toUpperCase() }}
              </div>
              <div v-if="project.shared_with.length > 3" class="w-6 h-6 rounded-full bg-gray-200 dark:bg-gray-600 border-2 border-white dark:border-gray-800 flex items-center justify-center text-xs text-gray-600 dark:text-gray-300">
                +{{ project.shared_with.length - 3 }}
              </div>
            </div>
            <span v-if="isOverdue(project)" class="text-red-600 font-medium">
              Scaduto
            </span>
          </div>
        </div>
      </Card>
    </div>

    <!-- Add Modal -->
    <AddProjectModal
      v-if="showAddModal"
      @close="showAddModal = false"
      @created="onProjectCreated"
    />

    <!-- Detail Modal -->
    <ProjectDetailModal
      v-if="selectedProject"
      :project="selectedProject"
      @close="selectedProject = null"
      @updated="onProjectUpdated"
      @deleted="onProjectDeleted"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useProjectsStore } from '@/stores/projects'
import { useAuthStore } from '@/stores/auth'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddProjectModal from '@/components/projects/AddProjectModal.vue'
import ProjectDetailModal from '@/components/projects/ProjectDetailModal.vue'

const projectsStore = useProjectsStore()
const authStore = useAuthStore()

function isOwner(project) {
  return project.user_id === authStore.user?.id
}
const showAddModal = ref(false)
const selectedProject = ref(null)
const selectedStatus = ref('')
const currentPropertyId = ref(null)

const statuses = [
  { value: '', label: 'Tutti', icon: '📋' },
  { value: 'planned', label: 'Pianificati', icon: '📅' },
  { value: 'active', label: 'Attivi', icon: '🔨' },
  { value: 'completed', label: 'Completati', icon: '✅' },
  { value: 'cancelled', label: 'Annullati', icon: '❌' }
]

const defaultStats = { total_budget: 0, total_spent: 0, remaining: 0, percentage_spent: 0, expense_count: 0 }

function ensureStats(project) {
  if (!project.stats) project.stats = { ...defaultStats, total_budget: project.budget || 0, remaining: project.budget || 0 }
  return project
}

const filteredProjects = computed(() => {
  const all = projectsStore.projects.map(ensureStats)
  if (selectedStatus.value === '') return all
  return all.filter(p => p.status === selectedStatus.value)
})

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(value || 0)
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
  const startDate = new Date(start).toLocaleDateString('it-IT', { day: '2-digit', month: 'short' })
  const endDate = new Date(end).toLocaleDateString('it-IT', { day: '2-digit', month: 'short', year: 'numeric' })
  return `${startDate} - ${endDate}`
}

function isOverdue(project) {
  if (project.status === 'completed' || project.status === 'cancelled') return false
  return new Date(project.end_date) < new Date()
}

function viewProject(project) {
  selectedProject.value = project
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

function loadProjects() {
  if (currentPropertyId.value) {
    projectsStore.fetchProjects(currentPropertyId.value)
  }
}

function onProjectCreated() {
  showAddModal.value = false
  loadProjects()
}

function onProjectUpdated() {
  selectedProject.value = null
  loadProjects()
}

function onProjectDeleted() {
  selectedProject.value = null
  loadProjects()
}

onMounted(async () => {
  await fetchCurrentProperty()
  loadProjects()
})
</script>
