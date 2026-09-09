<template>
  <div class="h-64">
    <Line :data="chartData" :options="mergedOptions" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { useSettingsStore } from '@/stores/settings'
import { useChartTheme } from '@/composables/useChartTheme'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const settingsStore = useSettingsStore()
const theme = useChartTheme()

const props = defineProps({
  chartData: {
    type: Object,
    required: true
  },
  chartOptions: {
    type: Object,
    default: () => ({})
  },
  currency: {
    type: String,
    default: 'EUR'
  }
})

const defaultOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx) => _formatCurrency(ctx.parsed.y, settingsStore.formatSettings)
      }
    }
  },
  scales: {
    x: {
      ticks: { color: theme.value.tick },
      grid: { display: false },
      border: { color: theme.value.grid }
    },
    y: {
      beginAtZero: true,
      ticks: {
        color: theme.value.tick,
        callback: (value) => _formatCurrency(value, settingsStore.formatSettings, { maximumFractionDigits: 0 })
      },
      grid: { color: theme.value.grid },
      border: { display: false }
    }
  },
  elements: {
    line: {
      tension: 0.3
    },
    point: {
      radius: 4,
      hoverRadius: 6
    }
  }
}))

const mergedOptions = computed(() => ({
  ...defaultOptions.value,
  ...props.chartOptions
}))
</script>
