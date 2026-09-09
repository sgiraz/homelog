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
import { useSettingsStore } from '@/stores/settings'
import { useChartTheme } from '@/composables/useChartTheme'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'

ChartJS.register(ArcElement, Tooltip, Legend)

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
  // Surface-coloured gap keeps neighbouring slices readable.
  elements: {
    arc: {
      borderColor: theme.value.surface,
      borderWidth: 2
    }
  },
  plugins: {
    legend: {
      position: isMobile.value ? 'bottom' : 'right',
      labels: {
        usePointStyle: true,
        padding: isMobile.value ? 10 : 15,
        font: { size: isMobile.value ? 11 : 12 },
        color: theme.value.tick,
        // Share in the label: arcs alone are unreadable with a dominant category.
        generateLabels: (chart) => {
          const { labels, datasets } = chart.data
          const values = datasets[0]?.data ?? []
          const total = values.reduce((a, b) => a + b, 0)
          return labels.map((label, i) => {
            const share = total > 0 ? (values[i] / total) * 100 : 0
            // A non-zero slice must never read "0%".
            const pct = share > 0 && share < 1 ? '<1' : Math.round(share)
            return {
              text: `${label} — ${pct}%`,
              // Chart.js v4 paints legend text with the item's own fontColor;
              // labels.color only feeds the default generateLabels, so a custom
              // one has to carry it or the text falls back to black.
              fontColor: theme.value.tick,
              fillStyle: datasets[0].backgroundColor[i],
              strokeStyle: datasets[0].backgroundColor[i],
              pointStyle: 'circle',
              hidden: false,
              index: i
            }
          })
        }
      },
      onClick: (event, legendItem) => {
        if (props.isSubcategory) return
        emit('slice-click', legendItem.index)
      }
    },
    tooltip: {
      callbacks: {
        label: (ctx) => {
          const value = _formatCurrency(ctx.parsed, settingsStore.formatSettings)
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
