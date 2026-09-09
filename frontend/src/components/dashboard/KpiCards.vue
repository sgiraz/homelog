<template>
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
    <Card v-for="kpi in kpis" :key="kpi.key" class="p-3 sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <div class="min-w-0">
          <div class="text-xs sm:text-sm text-ink-soft">{{ t(`dashboard.kpi.${kpi.key}`) }}</div>
          <!-- Compact form on narrow screens; full amount in title + sr-only. -->
          <div class="text-base sm:text-2xl font-bold text-ink tabular-nums" :title="kpi.full">
            <span aria-hidden="true" class="sm:hidden">{{ kpi.compact }}</span>
            <span aria-hidden="true" class="hidden sm:inline">{{ kpi.full }}</span>
            <span class="sr-only">{{ kpi.full }}</span>
          </div>
        </div>
        <div
          class="w-9 h-9 sm:w-12 sm:h-12 rounded-full flex items-center justify-center shrink-0"
          :class="kpi.primary ? 'bg-accent/10' : 'bg-surface-2'"
        >
          <svg
            class="w-4 h-4 sm:w-6 sm:h-6"
            :class="kpi.primary ? 'text-accent-soft' : 'text-ink-muted'"
            fill="none" stroke="currentColor" viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="kpi.icon" />
          </svg>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'KpiCards' })

import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'

const { t } = useI18n()

const props = defineProps({
  monthTotal: {
    type: Number,
    required: true
  },
  periodCount: {
    type: Number,
    required: true
  },
  dailyAverage: {
    type: Number,
    required: true
  },
  yearTotal: {
    type: Number,
    required: true
  },
  formatCurrency: {
    type: Function,
    required: true
  },
  formatCurrencyCompact: {
    type: Function,
    required: true
  }
})

const ICONS = {
  monthExpenses: 'M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  periodCount: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  dailyAverage: 'M13 7h8m0 0v8m0-8l-8 8-4-4-6 6',
  yearExpenses: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z'
}

// Only the headline KPI wears the accent; the rest stay neutral.
const kpis = computed(() => [
  {
    key: 'monthExpenses',
    full: props.formatCurrency(props.monthTotal),
    compact: props.formatCurrencyCompact(props.monthTotal),
    icon: ICONS.monthExpenses,
    primary: true
  },
  {
    key: 'periodCount',
    full: String(props.periodCount),
    compact: String(props.periodCount),
    icon: ICONS.periodCount,
    primary: false
  },
  {
    key: 'dailyAverage',
    full: props.formatCurrency(props.dailyAverage),
    compact: props.formatCurrencyCompact(props.dailyAverage),
    icon: ICONS.dailyAverage,
    primary: false
  },
  {
    key: 'yearExpenses',
    full: props.formatCurrency(props.yearTotal),
    compact: props.formatCurrencyCompact(props.yearTotal),
    icon: ICONS.yearExpenses,
    primary: false
  }
])
</script>
