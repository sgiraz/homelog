<template>
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
    <Card class="p-3 sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <div class="min-w-0">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400">{{ t('dashboard.kpi.monthExpenses') }}</div>
          <div class="text-lg sm:text-2xl font-bold text-gray-900 dark:text-white truncate">
            {{ formatCurrency(monthTotal) }}
          </div>
        </div>
        <div class="w-9 h-9 sm:w-12 sm:h-12 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center shrink-0">
          <svg class="w-4 h-4 sm:w-6 sm:h-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      </div>
    </Card>

    <Card class="p-3 sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <div class="min-w-0">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400">{{ t('dashboard.kpi.periodCount') }}</div>
          <div class="text-lg sm:text-2xl font-bold text-gray-900 dark:text-white">
            {{ periodCount }}
          </div>
        </div>
        <div class="w-9 h-9 sm:w-12 sm:h-12 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center shrink-0">
          <svg class="w-4 h-4 sm:w-6 sm:h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
        </div>
      </div>
    </Card>

    <Card class="p-3 sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <div class="min-w-0">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400">{{ t('dashboard.kpi.dailyAverage') }}</div>
          <div class="text-lg sm:text-2xl font-bold text-gray-900 dark:text-white truncate">
            {{ formatCurrency(dailyAverage) }}
          </div>
        </div>
        <div class="w-9 h-9 sm:w-12 sm:h-12 bg-purple-100 dark:bg-purple-900 rounded-full flex items-center justify-center shrink-0">
          <svg class="w-4 h-4 sm:w-6 sm:h-6 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
          </svg>
        </div>
      </div>
    </Card>

    <Card class="p-3 sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <div class="min-w-0">
          <div class="text-xs sm:text-sm text-gray-600 dark:text-gray-400">{{ t('dashboard.kpi.yearExpenses') }}</div>
          <div class="text-lg sm:text-2xl font-bold text-gray-900 dark:text-white truncate">
            {{ formatCurrency(yearTotal) }}
          </div>
        </div>
        <div class="w-9 h-9 sm:w-12 sm:h-12 bg-orange-100 dark:bg-orange-900 rounded-full flex items-center justify-center shrink-0">
          <svg class="w-4 h-4 sm:w-6 sm:h-6 text-orange-600 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'KpiCards' })

import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'

const { t } = useI18n()

defineProps({
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
  }
})
</script>
