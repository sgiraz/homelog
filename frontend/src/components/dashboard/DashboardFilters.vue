<template>
  <Card class="p-4">
    <!-- Mobile: filtri collassabili -->
    <div class="sm:hidden">
      <div class="flex items-center justify-between">
        <button
          @click="emit('update:filtersOpen', !filtersOpen)"
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
        <Button v-if="hasActiveFilters" @click="emit('reset')" variant="secondary" size="sm">
          Reset
        </Button>
      </div>
      <Transition name="filter-expand">
        <div v-if="filtersOpen" class="mt-3 space-y-3 border-t border-gray-100 dark:border-gray-700 pt-3">
          <div class="grid grid-cols-2 gap-2">
            <select
              :value="filters.categoryId"
              @change="emit('update:filters', { ...filters, categoryId: $event.target.value }); emit('apply')"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Tutte categorie</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.icon }} {{ cat.name }}</option>
            </select>
            <select
              :value="filters.projectId"
              @change="emit('update:filters', { ...filters, projectId: $event.target.value }); emit('apply')"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Tutti progetti</option>
              <option v-for="proj in projects" :key="proj.id" :value="proj.id">{{ proj.icon }} {{ proj.name }}</option>
            </select>
            <input
              :value="filters.from"
              @change="emit('update:filters', { ...filters, from: $event.target.value }); emit('apply')"
              type="date"
              class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              :value="filters.to"
              @change="emit('update:filters', { ...filters, to: $event.target.value }); emit('apply')"
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
          :value="filters.categoryId"
          @change="emit('update:filters', { ...filters, categoryId: $event.target.value }); emit('apply')"
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
          :value="filters.projectId"
          @change="emit('update:filters', { ...filters, projectId: $event.target.value }); emit('apply')"
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
          :value="filters.from"
          @change="emit('update:filters', { ...filters, from: $event.target.value }); emit('apply')"
          type="date"
          class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
        />
      </div>

      <div class="flex items-center gap-2">
        <label class="text-sm text-gray-600 dark:text-gray-400">A:</label>
        <input
          :value="filters.to"
          @change="emit('update:filters', { ...filters, to: $event.target.value }); emit('apply')"
          type="date"
          class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
        />
      </div>

      <Button @click="emit('reset')" variant="secondary" class="text-sm">
        Reset Filtri
      </Button>
    </div>
  </Card>
</template>

<script setup>
defineOptions({ name: 'DashboardFilters' })

import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

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
