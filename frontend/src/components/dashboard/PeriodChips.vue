<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <button
      v-for="range in RANGES"
      :key="range.days"
      type="button"
      class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
      :class="activeDays === range.days
        ? 'bg-accent/10 text-accent-soft'
        : 'text-ink-soft hover:bg-surface-2'"
      @click="emit('select', range.days)"
    >
      {{ t(`dashboard.period.${range.key}`) }}
    </button>
    <!-- Shown only once the dates stop matching a preset: it is a state, not a
         choice, so it reads as a label and opens the filters instead. -->
    <button
      v-if="activeDays === null"
      type="button"
      class="px-3 py-1.5 rounded-lg text-sm font-medium bg-accent/10 text-accent-soft"
      @click="emit('open-filters')"
    >
      {{ t('dashboard.period.custom') }}
    </button>
  </div>
</template>

<script setup>
defineOptions({ name: 'PeriodChips' })

import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// Windows counted in days, so "last 30 days" always means 30 days — a calendar
// month would make the number jump between February and March.
const RANGES = [
  { key: 'days7', days: 7 },
  { key: 'days30', days: 30 },
  { key: 'days90', days: 90 },
  { key: 'year', days: 365 }
]

defineProps({
  /** Length in days of the active window, or null when it matches no preset. */
  activeDays: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['select', 'open-filters'])
</script>
