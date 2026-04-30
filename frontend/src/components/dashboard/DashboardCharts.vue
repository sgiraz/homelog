<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <Card class="p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ categoryChartTitle }}</h3>
      <PieChart
        v-if="hasCategoryData"
        :chartData="categoryChartData"
        :currency="currency"
        :isSubcategory="isSubcategory"
        @slice-click="(index) => emit('slice-click', index)"
      />
      <div v-else class="h-64 flex items-center justify-center text-gray-500">
        {{ t('dashboard.charts.noData') }}
      </div>
    </Card>

    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ trendChartTitle }}</h3>
        <div class="flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-0.5">
          <button
            @click="emit('update:trendChartType', 'line')"
            :class="[
              'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
              trendChartType === 'line'
                ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
            ]"
            :title="t('dashboard.charts.lineTooltip')"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </button>
          <button
            @click="emit('update:trendChartType', 'bar')"
            :class="[
              'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
              trendChartType === 'bar'
                ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
            ]"
            :title="t('dashboard.charts.barTooltip')"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          </button>
        </div>
      </div>
      <template v-if="hasTrendData">
        <LineChart v-if="trendChartType === 'line'" :chartData="trendLineChartData" :currency="currency" />
        <BarChart v-else :chartData="trendBarChartData" :currency="currency" />
      </template>
      <div v-else class="h-64 flex items-center justify-center text-gray-500">
        {{ t('dashboard.charts.noData') }}
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'DashboardCharts' })

import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'
import BarChart from '@/components/charts/BarChart.vue'
import LineChart from '@/components/charts/LineChart.vue'
import PieChart from '@/components/charts/PieChart.vue'

const { t } = useI18n()

defineProps({
  categoryChartTitle: {
    type: String,
    required: true
  },
  hasCategoryData: {
    type: Boolean,
    required: true
  },
  categoryChartData: {
    type: Object,
    required: true
  },
  currency: {
    type: String,
    required: true
  },
  isSubcategory: {
    type: Boolean,
    required: true
  },
  trendChartTitle: {
    type: String,
    required: true
  },
  hasTrendData: {
    type: Boolean,
    required: true
  },
  trendBarChartData: {
    type: Object,
    required: true
  },
  trendLineChartData: {
    type: Object,
    required: true
  },
  trendChartType: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:trendChartType', 'slice-click'])
</script>
