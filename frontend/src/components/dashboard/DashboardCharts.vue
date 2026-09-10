<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <Card class="p-6">
      <!-- Back leads the title, like the other detail views: icon-only chevron
           with a 44pt target; the title already says where we are. -->
      <div class="flex items-center gap-1 mb-4 min-h-[44px]">
        <button
          v-if="isSubcategory"
          @click="emit('back')"
          class="flex items-center justify-center min-w-[44px] min-h-[44px] -ml-2.5 rounded-lg text-ink-soft hover:text-ink hover:bg-surface-2 active:bg-surface-3 transition-colors shrink-0"
          :aria-label="t('dashboard.charts.backToCategories')"
          :title="t('dashboard.charts.backToCategories')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h3 class="text-lg font-semibold text-ink truncate">{{ categoryChartTitle }}</h3>
      </div>
      <CategoryBars
        v-if="hasCategoryData"
        :rows="categoryRows"
        :formatCurrency="formatCurrency"
        :totalCount="categoryTotalCount"
        :expanded="categoriesExpanded"
        @slice-click="(index) => emit('slice-click', index)"
        @update:expanded="emit('update:categoriesExpanded', $event)"
      />
      <div v-else class="h-64 flex items-center justify-center text-ink-muted">
        {{ t('dashboard.charts.noData') }}
      </div>
    </Card>

    <Card class="p-6 flex flex-col">
      <div class="flex items-center mb-4 min-h-[44px]">
        <h3 class="text-lg font-semibold text-ink">{{ trendChartTitle }}</h3>
      </div>
      <!-- Always bars: every bucket is a period total, and a line would
           interpolate between sums that have nothing in between. -->
      <div v-if="hasTrendData" class="flex-1 min-h-64">
        <BarChart :chartData="trendBarChartData" :currency="currency" />
      </div>
      <div v-else class="flex-1 min-h-64 flex items-center justify-center text-ink-muted">
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
import CategoryBars from '@/components/charts/CategoryBars.vue'

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
  categoryRows: {
    type: Array,
    required: true
  },
  formatCurrency: {
    type: Function,
    required: true
  },
  categoryTotalCount: {
    type: Number,
    required: true
  },
  categoriesExpanded: {
    type: Boolean,
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
  }
})

const emit = defineEmits(['slice-click', 'back', 'update:categoriesExpanded'])
</script>
