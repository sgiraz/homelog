<template>
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
    <!-- The headline: what the household spent in the window on screen. -->
    <Card class="p-4 sm:p-6">
      <div class="text-xs sm:text-sm text-ink-soft">{{ t('dashboard.kpi.periodTotal') }}</div>
      <!-- Currency never truncates: the compact form shows on narrow screens,
           the full amount stays in the title and for assistive tech. -->
      <div class="mt-1 text-2xl sm:text-3xl font-bold text-ink tabular-nums" :title="totalFull">
        <span aria-hidden="true" class="sm:hidden">{{ totalCompact }}</span>
        <span aria-hidden="true" class="hidden sm:inline">{{ totalFull }}</span>
        <span class="sr-only">{{ totalFull }}</span>
      </div>
      <div class="mt-2 flex items-center gap-2 text-xs sm:text-sm">
        <span v-if="delta !== null" :class="delta > 0 ? 'text-accent-soft' : 'text-positive'">
          {{ deltaLabel }}
        </span>
        <span class="text-ink-muted">{{ countLabel }}</span>
      </div>
    </Card>

    <Card class="p-4 sm:p-6">
      <div class="text-xs sm:text-sm text-ink-soft">{{ t('dashboard.kpi.dailyAverage') }}</div>
      <div class="mt-1 text-2xl sm:text-3xl font-bold text-ink tabular-nums" :title="averageFull">
        <span aria-hidden="true" class="sm:hidden">{{ averageCompact }}</span>
        <span aria-hidden="true" class="hidden sm:inline">{{ averageFull }}</span>
        <span class="sr-only">{{ averageFull }}</span>
      </div>
      <div class="mt-2 text-xs sm:text-sm text-ink-muted">
        {{ daysLabel }}
      </div>
    </Card>

    <Card class="p-4 sm:p-6">
      <div class="text-xs sm:text-sm text-ink-soft">{{ t('dashboard.kpi.topCategory') }}</div>
      <!-- A category name is text, not a figure: it wraps to a second line
           rather than being cut, and sits a size below the numbers. -->
      <div
        v-if="topCategory"
        class="mt-1 text-xl sm:text-2xl font-bold text-ink leading-tight line-clamp-2"
        :title="topCategory.label"
      >
        {{ topCategory.label }}
      </div>
      <div v-else class="mt-1 text-2xl sm:text-3xl font-bold text-ink-muted">—</div>
      <div v-if="topCategory" class="mt-2 text-xs sm:text-sm text-ink-muted tabular-nums">
        {{ formatCurrency(topCategory.amount) }} · {{ Math.round(topCategory.share) }}%
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
  periodTotal: {
    type: Number,
    required: true
  },
  previousTotal: {
    type: Number,
    default: null
  },
  count: {
    type: Number,
    required: true
  },
  days: {
    type: Number,
    required: true
  },
  /** Biggest category of the period: { label, amount, share } or null. */
  topCategory: {
    type: Object,
    default: null
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

// Plurals follow the repo convention: the suffix is picked here, the message
// takes {n}. vue-i18n's pipe syntax is off limits.
const countLabel = computed(() =>
  t(`dashboard.kpi.expenseCount_${props.count === 1 ? 'one' : 'other'}`, { n: props.count })
)
const daysLabel = computed(() =>
  t(`dashboard.kpi.overDays_${props.days === 1 ? 'one' : 'other'}`, { n: props.days })
)

const totalFull = computed(() => props.formatCurrency(props.periodTotal))
const totalCompact = computed(() => props.formatCurrencyCompact(props.periodTotal))

const dailyAverage = computed(() => (props.days > 0 ? props.periodTotal / props.days : 0))
const averageFull = computed(() => props.formatCurrency(dailyAverage.value))
const averageCompact = computed(() => props.formatCurrencyCompact(dailyAverage.value))

// No previous window, or one with nothing in it, means there is nothing to
// compare against: "+100%" against zero would be noise, not information.
const delta = computed(() => {
  if (!props.previousTotal) return null
  return ((props.periodTotal - props.previousTotal) / props.previousTotal) * 100
})

const deltaLabel = computed(() => {
  const value = delta.value
  if (value === null) return ''
  const sign = value > 0 ? '+' : '−'
  return t('dashboard.kpi.deltaVsPrevious', {
    delta: `${sign}${Math.abs(Math.round(value))}%`
  })
})
</script>
