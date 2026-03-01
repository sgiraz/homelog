<template>
  <div :class="isMobile ? 'h-80' : 'h-64'">
    <Doughnut :data="chartData" :options="mergedOptions" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  ArcElement,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(ArcElement, Tooltip, Legend)

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
  },
  isSubcategory: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['slice-click'])

const isMobile = ref(window.innerWidth < 640)
function handleResize() { isMobile.value = window.innerWidth < 640 }
onMounted(() => window.addEventListener('resize', handleResize))
onUnmounted(() => window.removeEventListener('resize', handleResize))

const defaultOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  onClick: (event, elements) => {
    if (props.isSubcategory || elements.length === 0) return
    emit('slice-click', elements[0].index)
  },
  onHover: (event, elements) => {
    if (event.native?.target) {
      event.native.target.style.cursor =
        !props.isSubcategory && elements.length > 0 ? 'pointer' : 'default'
    }
  },
  plugins: {
    legend: {
      position: isMobile.value ? 'bottom' : 'right',
      labels: {
        usePointStyle: true,
        padding: isMobile.value ? 10 : 15,
        font: { size: isMobile.value ? 11 : 12 }
      },
      onClick: (event, legendItem) => {
        if (props.isSubcategory) return
        emit('slice-click', legendItem.index)
      }
    },
    tooltip: {
      callbacks: {
        label: (ctx) => {
          const value = new Intl.NumberFormat('it-IT', {
            style: 'currency',
            currency: props.currency
          }).format(ctx.parsed)
          const total = ctx.dataset.data.reduce((a, b) => a + b, 0)
          const pct = total > 0 ? ((ctx.parsed / total) * 100).toFixed(1) : '0'
          return ` ${value} (${pct}%)`
        }
      }
    }
  }
}))

const mergedOptions = computed(() => ({
  ...defaultOptions.value,
  ...props.chartOptions
}))
</script>
