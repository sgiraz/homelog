<template>
  <Card class="p-4">
    <!-- Mobile: filtri collassabili -->
    <div class="sm:hidden">
      <div class="flex items-center justify-between">
        <button
          @click="emit('update:filtersOpen', !filtersOpen)"
          class="flex items-center gap-2 text-sm font-medium text-ink-soft"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
          </svg>
          {{ t('dashboard.filters.filtersLabel') }}
          <span
            v-if="activeFiltersCount > 0"
            class="inline-flex items-center justify-center w-5 h-5 text-xs font-bold text-white bg-blue-600 rounded-full"
          >
            {{ activeFiltersCount }}
          </span>
        </button>
        <Button v-if="hasActiveFilters" @click="emit('reset')" variant="secondary" size="sm">
          {{ t('dashboard.filters.reset') }}
        </Button>
      </div>
      <Transition name="filter-expand">
        <div v-if="filtersOpen" class="mt-3 space-y-3 border-t border-line pt-3">
          <div class="grid grid-cols-2 gap-2">
            <select
              :value="filters.categoryId"
              @change="emit('update:filters', { ...filters, categoryId: $event.target.value }); emit('apply')"
              class="px-3 py-2 border border-line rounded-lg
                     bg-surface text-ink text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">{{ t('dashboard.filters.allCategoriesLong') }}</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.icon }} {{ cat.name }}</option>
            </select>
            <select
              :value="filters.projectId"
              @change="emit('update:filters', { ...filters, projectId: $event.target.value }); emit('apply')"
              class="px-3 py-2 border border-line rounded-lg
                     bg-surface text-ink text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">{{ t('dashboard.filters.allProjectsLong') }}</option>
              <option v-for="proj in projects" :key="proj.id" :value="proj.id">{{ proj.icon }} {{ proj.name }}</option>
            </select>
            <input
              :value="filters.from"
              @change="emit('update:filters', { ...filters, from: $event.target.value }); emit('apply')"
              type="date"
              class="px-3 py-2 border border-line rounded-lg
                     bg-surface text-ink text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              :value="filters.to"
              @change="emit('update:filters', { ...filters, to: $event.target.value }); emit('apply')"
              type="date"
              class="px-3 py-2 border border-line rounded-lg
                     bg-surface text-ink text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
      </Transition>
    </div>

    <!-- Desktop: filtri sempre visibili -->
    <div class="hidden sm:flex flex-wrap items-center gap-4">
      <div class="flex items-center gap-2">
        <label class="text-sm text-ink-soft">{{ t('dashboard.filters.categoryLabel') }}</label>
        <select
          :value="filters.categoryId"
          @change="emit('update:filters', { ...filters, categoryId: $event.target.value }); emit('apply')"
          class="px-3 py-2 border border-line rounded-lg
                 bg-surface text-ink text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">{{ t('dashboard.filters.allCategoriesShort') }}</option>
          <option v-for="cat in categories" :key="cat.id" :value="cat.id">
            {{ cat.icon }} {{ cat.name }}
          </option>
        </select>
      </div>

      <div class="flex items-center gap-2">
        <label class="text-sm text-ink-soft">{{ t('dashboard.filters.projectLabel') }}</label>
        <select
          :value="filters.projectId"
          @change="emit('update:filters', { ...filters, projectId: $event.target.value }); emit('apply')"
          class="px-3 py-2 border border-line rounded-lg
                 bg-surface text-ink text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">{{ t('dashboard.filters.allProjectsShort') }}</option>
          <option v-for="proj in projects" :key="proj.id" :value="proj.id">
            {{ proj.icon }} {{ proj.name }}
          </option>
        </select>
      </div>

      <div class="flex items-center gap-2">
        <label class="text-sm text-ink-soft">{{ t('dashboard.filters.fromLabel') }}</label>
        <input
          :value="filters.from"
          @change="emit('update:filters', { ...filters, from: $event.target.value }); emit('apply')"
          type="date"
          class="px-3 py-2 border border-line rounded-lg
                 bg-surface text-ink text-sm"
        />
      </div>

      <div class="flex items-center gap-2">
        <label class="text-sm text-ink-soft">{{ t('dashboard.filters.toLabel') }}</label>
        <input
          :value="filters.to"
          @change="emit('update:filters', { ...filters, to: $event.target.value }); emit('apply')"
          type="date"
          class="px-3 py-2 border border-line rounded-lg
                 bg-surface text-ink text-sm"
        />
      </div>

      <Button @click="emit('reset')" variant="secondary" class="text-sm">
        {{ t('dashboard.filters.resetFilters') }}
      </Button>
    </div>
  </Card>
</template>

<script setup>
defineOptions({ name: 'DashboardFilters' })

import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()

defineProps({
  filters: {
    type: Object,
    required: true
  },
  categories: {
    type: Array,
    required: true
  },
  projects: {
    type: Array,
    required: true
  },
  filtersOpen: {
    type: Boolean,
    required: true
  },
  hasActiveFilters: {
    type: Boolean,
    required: true
  },
  activeFiltersCount: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['update:filters', 'update:filtersOpen', 'apply', 'reset'])
</script>
