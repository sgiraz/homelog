<template>
  <div class="h-64">
    <Bar :data="chartData" :options="mergedOptions" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

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

const defaultOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx) => new Intl.NumberFormat('it-IT', {
          style: 'currency',
          currency: props.currency
        }).format(ctx.parsed.y)
      }
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        callback: (value) => new Intl.NumberFormat('it-IT', {
          style: 'currency',
          currency: props.currency,
          maximumFractionDigits: 0
        }).format(value)
      }
    }
  }
}

const mergedOptions = computed(() => ({
  ...defaultOptions,
  ...props.chartOptions
}))
</script>
