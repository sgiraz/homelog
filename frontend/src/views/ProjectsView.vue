<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-ink">{{ t('projects.title') }}</h1>
      <Button @click="showAddModal = true">
        <svg class="w-5 h-5 sm:mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">{{ t('projects.newProjectButton') }}</span>
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
              : 'text-ink-soft hover:bg-surface-2'
          ]"
        >
          <span>{{ status.icon }}</span>
          <span>{{ status.label }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="projectsStore.loading" class="text-center py-12 text-ink-soft">
      {{ t('projects.loading') }}
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredProjects.length === 0" class="text-center py-12">
      <div class="text-6xl mb-4">🏗️</div>
      <h3 class="text-xl font-semibold mb-2">
        {{ selectedStatus === '' ? t('projects.emptyAll') : t('projects.emptyFiltered', { filter: statuses.find(s => s.value === selectedStatus)?.label.toLowerCase() }) }}
      </h3>
      <p class="text-ink-soft mb-6">
        {{ t('projects.emptyDescription') }}
      </p>
      <Button @click="showAddModal = true">
        {{ t('projects.createFirstButton') }}
      </Button>
    </div>

    <!-- Projects Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card
        v-for="project in filteredProjects"
        :key="project.id"
        :ref="(el) => registerRow(project.id, el?.$el || el)"
        class="p-6 cursor-pointer hover:shadow-lg transition-shadow"
        :class="{ 'search-flash': isHighlighted(project.id) }"
        @click="viewProject(project)"
      >
        <!-- Header -->
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="text-3xl">{{ project.icon || '🏗️' }}</div>
            <div>
              <h3 class="text-lg font-bold text-ink">
                {{ project.name }}
              </h3>
              <p class="text-sm text-ink-soft">
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
              {{ t('projects.card.shared') }}
            </span>
          </div>
        </div>

        <!-- Description -->
        <p v-if="project.description" class="text-sm text-ink-soft mb-4 line-clamp-2">
          {{ project.description }}
        </p>

        <!-- Budget Progress -->
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-ink-soft">{{ t('projects.card.budget') }}</span>
            <span class="font-medium">{{ formatCurrency(project.budget) }}</span>
          </div>

          <div class="flex justify-between text-sm">
            <span class="text-ink-soft">{{ t('projects.card.spent') }}</span>
            <span :class="[
              'font-medium',
              (project.stats?.total_spent || 0) > project.budget ? 'text-red-600' : 'text-ink'
            ]">
              {{ formatCurrency(project.stats?.total_spent || 0) }}
            </span>
          </div>

          <!-- Progress Bar -->
          <div class="w-full bg-surface-3 rounded-full h-2.5">
            <div
              :class="[
                'h-2.5 rounded-full transition-all',
                (project.stats?.percentage_spent || 0) > 100 ? 'bg-red-600' : 'bg-blue-600'
              ]"
              :style="{ width: Math.min(project.stats?.percentage_spent || 0, 100) + '%' }"
            ></div>
          </div>

          <div class="flex justify-between text-xs text-ink-soft">
            <span>{{ t('projects.card.percentUsed', { percent: (project.stats?.percentage_spent || 0).toFixed(1) }) }}</span>
            <span>
              {{ (project.stats?.remaining ?? 0) >= 0 ? t('projects.card.remaining') : t('projects.card.overrun') }}
              {{ formatCurrency(Math.abs(project.stats?.remaining ?? 0)) }}
            </span>
          </div>
        </div>

        <!-- Footer Stats -->
        <div class="mt-4 pt-4 border-t border-line flex justify-between items-center text-sm">
          <span class="text-ink-soft">
            {{ t(`projects.card.${(project.stats?.expense_count || 0) === 1 ? 'expenseCount_one' : 'expenseCount_other'}`, { n: project.stats?.expense_count || 0 }) }}
          </span>
          <div class="flex items-center gap-2">
            <!-- Shared member avatars -->
            <div v-if="project.shared_with?.length > 0" class="flex -space-x-1">
              <div
                v-for="user in project.shared_with.slice(0, 3)"
                :key="user.id"
                class="w-6 h-6 rounded-full bg-purple-100 dark:bg-purple-900 border-2 border-surface flex items-center justify-center text-xs font-medium text-purple-700 dark:text-purple-300"
                :title="user.name"
              >
                {{ user.name?.[0]?.toUpperCase() }}
              </div>
              <div v-if="project.shared_with.length > 3" class="w-6 h-6 rounded-full bg-surface-3 border-2 border-surface flex items-center justify-center text-xs text-ink-soft">
                +{{ project.shared_with.length - 3 }}
              </div>
            </div>
            <span v-if="isOverdue(project)" class="text-red-600 font-medium">
              {{ t('projects.card.overdue') }}
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

  </div>
</template>

<script setup>
defineOptions({ name: 'ProjectsView' })

import { ref, computed, onMounted, onActivated, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useProjectsStore } from '@/stores/projects'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useHighlight } from '@/composables/useHighlight'
import { formatCurrency as _formatCurrency, formatDate as _formatDate } from '@/utils/dateFormatter'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddProjectModal from '@/components/projects/AddProjectModal.vue'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const projectsStore = useProjectsStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

function isOwner(project) {
  if (project.user_id === authStore.user?.id) return true
  return project.members?.some(m => m.id === authStore.user?.id && m.role === 'owner') || false
}
const showAddModal = ref(false)
const currentPropertyId = ref(null)

const selectedStatus = ref('active')

const statuses = computed(() => [
  { value: 'active', label: t('projects.filters.active'), icon: '🔨' },
  { value: 'planned', label: t('projects.filters.planned'), icon: '📅' },
  { value: 'completed', label: t('projects.filters.completed'), icon: '✅' },
  { value: '', label: t('projects.filters.all'), icon: '📋' }
])

const defaultStats = { total_budget: 0, total_spent: 0, remaining: 0, percentage_spent: 0, expense_count: 0 }

function ensureStats(project) {
  if (!project.stats) project.stats = { ...defaultStats, total_budget: project.budget || 0, remaining: project.budget || 0 }
  return project
}

const filteredProjects = computed(() => {
  const all = projectsStore.projects.map(ensureStats)
  if (selectedStatus.value === '') return all
  if (selectedStatus.value === 'completed') {
    return all.filter(p => p.status === 'completed' || p.status === 'cancelled')
  }
  return all.filter(p => p.status === selectedStatus.value)
})


// Search-highlight wiring: if `?highlight=<id>` refers to a project hidden by
// the current status filter, relax the filter to "all" so the row is visible.
const { isHighlighted, registerRow, highlightId } = useHighlight({
  source: () => filteredProjects.value,
})
watch(
  () => [highlightId.value, projectsStore.projects.length],
  () => {
    if (highlightId.value == null) return
    const visible = filteredProjects.value.some(p => p.id === highlightId.value)
    if (!visible && projectsStore.projects.some(p => p.id === highlightId.value)) {
      selectedStatus.value = ''
    }
  },
  { immediate: true },
)

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function getStatusLabel(status) {
  const key = `projects.status.${status}`
  return t(key) === key ? status : t(key)
}

function getStatusColor(status) {
  const map = {
    planned: 'bg-blue-100 text-blue-700',
    active: 'bg-green-100 text-green-700',
    completed: 'bg-surface-2 text-ink-soft',
    cancelled: 'bg-red-100 text-red-700'
  }
  return map[status] || 'bg-surface-2 text-ink-soft'
}

function formatDateRange(start, end) {
  const s = _formatDate(start, settingsStore.dateSettings)
  const e = _formatDate(end, settingsStore.dateSettings)
  return `${s} - ${e}`
}

function isOverdue(project) {
  if (project.status === 'completed' || project.status === 'cancelled') return false
  return new Date(project.end_date) < new Date()
}

function viewProject(project) {
  router.push(`/projects/${project.id}`)
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      // `?property=<id>` (from global search) overrides the "current" property
      // so a project living under a non-current property is still loaded.
      const requested = Number(route.query.property)
      const fromQuery = Number.isFinite(requested)
        ? data.find(p => p.id === requested)
        : null
      const currentProp = fromQuery || data.find(p => p.is_current) || data[0]
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

onMounted(async () => {
  await fetchCurrentProperty()
  loadProjects()
})

// keep-alive suppresses remount; re-run property selection when a global-
// search deep-link arrives with a ?property= that differs from the current one.
onActivated(async () => {
  const requested = Number(route.query.property)
  if (Number.isFinite(requested) && requested !== currentPropertyId.value) {
    await fetchCurrentProperty()
    loadProjects()
  }
})
</script>
