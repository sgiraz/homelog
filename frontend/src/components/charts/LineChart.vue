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
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const settingsStore = useSettingsStore()

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
    y: {
      beginAtZero: true,
      ticks: {
        callback: (value) => _formatCurrency(value, settingsStore.formatSettings, { maximumFractionDigits: 0 })
      }
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
