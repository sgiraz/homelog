<template>
  <!-- Individual Comparisons (Accordion) -->
  <div class="space-y-3">
    <h4 class="font-semibold text-ink">{{ t('utilities.comparisonAccordion.title') }}</h4>
    <div
      v-for="comparison in comparisons"
      :key="comparison.bill_id"
      :class="[
        'rounded-lg border-2 overflow-hidden transition-all',
        getStatusClasses(comparison.status)
      ]"
    >
      <!-- Compact header — always visible, tappable -->
      <button
        @click="$emit('toggle-card', comparison.bill_id)"
        class="w-full flex items-center justify-between p-3 sm:p-4 text-left"
      >
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <Badge :variant="getStatusVariant(comparison.status)" class="flex-shrink-0">
            <component :is="getStatusIcon(comparison.status)" class="w-3.5 h-3.5" />
            <span>{{ getStatusLabel(comparison.status) }}</span>
          </Badge>
          <div class="min-w-0">
            <span class="text-sm font-medium text-ink">
              {{ comparison.bill_number || '#' + comparison.bill_id }}
            </span>
            <span class="text-xs text-ink-faint ml-1.5">
              {{ formatDate(comparison.period_end) }}
            </span>
          </div>
        </div>
        <!-- Chevron -->
        <svg
          :class="['w-4 h-4 text-ink-faint transition-transform flex-shrink-0 ml-2', expandedCards.has(comparison.bill_id) ? 'rotate-180' : '']"
          fill="none" stroke="currentColor" viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      <!-- Expandable detail -->
      <div v-show="expandedCards.has(comparison.bill_id)" class="px-3 sm:px-4 pb-3 sm:pb-4 space-y-3">
        <!-- Days difference and effective threshold info -->
        <div v-if="comparison.days_difference > 0" class="text-xs bg-info/10 text-info-soft px-3 py-2 rounded">
          <span class="font-medium">{{ t('utilities.comparisonAccordion.daysDifference', { n: comparison.days_difference }) }}</span> {{ t('utilities.comparisonAccordion.daysDifferenceMessage') }}
          {{ t('utilities.comparisonAccordion.effectiveThreshold') }} <span class="font-medium">{{ fmtNum(comparison.effective_threshold) }} {{ getUnit() }}</span>
        </div>

        <!-- Alert message if any -->
        <div v-if="comparison.alert_message && comparison.status !== 'ok'" class="text-sm">
          <span :class="comparison.status === 'alert' ? 'text-danger-soft' : 'text-warning-soft'">
            {{ comparison.alert_message }}
          </span>
        </div>

        <!-- Electricity comparison (F1/F2/F3) -->
        <div v-if="utilityType === 'electricity'" class="grid grid-cols-3 gap-2 sm:gap-3">
          <ReadingBandComparison
            v-for="band in ['F1', 'F2', 'F3']"
            :key="band"
            :band="band"
            :fmtNum="fmtNum"
            :providerValue="comparison['provider_' + band.toLowerCase()]"
            :userValue="comparison['user_' + band.toLowerCase()]"
            :difference="comparison['difference_' + band.toLowerCase()]"
            :threshold="comparison.effective_threshold"
            unit="kWh"
          />
        </div>

        <!-- Gas/Water comparison (single value) -->
        <div v-else class="grid grid-cols-2 gap-3">
          <div class="text-center p-3 bg-surface rounded-lg">
            <p class="text-xs text-ink-muted mb-1">{{ t('utilities.comparisonAccordion.providerLabel') }}</p>
            <p class="text-base font-semibold text-ink">
              {{ comparison.provider_reading != null ? fmtNum(comparison.provider_reading) : '-' }}
              <span class="text-xs font-normal text-ink-muted">{{ getUnit() }}</span>
            </p>
          </div>
          <div class="text-center p-3 bg-surface rounded-lg">
            <p class="text-xs text-ink-muted mb-1">{{ t('utilities.comparisonAccordion.selfLabel') }}</p>
            <p class="text-base font-semibold text-ink">
              {{ comparison.user_reading != null ? fmtNum(comparison.user_reading) : '-' }}
              <span class="text-xs font-normal text-ink-muted">{{ getUnit() }}</span>
            </p>
          </div>
          <div v-if="comparison.difference != null" class="col-span-2 text-center">
            <span :class="[
              'text-sm font-medium',
              comparison.status === 'alert' ? 'text-danger-soft' :
              comparison.status === 'warning' ? 'text-warning-soft' : 'text-positive-soft'
            ]">
              {{ t('utilities.comparisonAccordion.differencePrefix') }} {{ fmtDiff(comparison.difference) }} {{ getUnit() }}
            </span>
          </div>
        </div>

        <!-- Reading type indicator -->
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Badge :variant="comparison.reading_type === 'actual' ? 'positive' : 'warning'">
            {{ comparison.reading_type === 'actual' ? t('utilities.comparisonAccordion.actualReading') : t('utilities.comparisonAccordion.estimatedReading') }}
          </Badge>
          <span v-if="comparison.provider_reading_date">
            {{ t('utilities.comparisonAccordion.ofDate', { date: formatDate(comparison.provider_reading_date) }) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { h } from 'vue'
import { useI18n } from 'vue-i18n'
import Badge from '@/components/common/Badge.vue'

defineOptions({ name: 'ComparisonAccordion' })

const { t } = useI18n()

defineProps({
  comparisons: {
    type: Array,
    required: true
  },
  expandedCards: {
    type: Set,
    required: true
  },
  utilityType: {
    type: String,
    required: true
  },
  fmtNum: {
    type: Function,
    required: true
  },
  fmtDiff: {
    type: Function,
    required: true
  },
  formatDate: {
    type: Function,
    required: true
  },
  getUnit: {
    type: Function,
    required: true
  }
})

defineEmits(['toggle-card'])

function getStatusClasses(status) {
  switch (status) {
    case 'alert':
      return 'border-danger/30 bg-danger/10'
    case 'warning':
      return 'border-warning/30 bg-warning/10'
    case 'no_data':
      return 'border-line bg-surface'
    default:
      return 'border-positive/30 bg-positive/10'
  }
}

function getStatusVariant(status) {
  switch (status) {
    case 'alert':
      return 'danger'
    case 'warning':
      return 'warning'
    case 'no_data':
      return 'neutral'
    default:
      return 'positive'
  }
}

function getStatusLabel(status) {
  switch (status) {
    case 'alert': return t('utilities.comparisonAccordion.statusAlert')
    case 'warning': return t('utilities.comparisonAccordion.statusWarning')
    case 'no_data': return t('utilities.comparisonAccordion.statusNoData')
    default: return t('utilities.comparisonAccordion.statusOk')
  }
}

function getStatusIcon(status) {
  const icons = {
    alert: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z' })
        ])
      }
    },
    warning: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    },
    no_data: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    },
    ok: {
      render() {
        return h('svg', { fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
          h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' })
        ])
      }
    }
  }
  return icons[status] || icons.ok
}

// Sub-component for electricity band comparison
const ReadingBandComparison = {
  props: ['band', 'providerValue', 'userValue', 'difference', 'unit', 'threshold', 'fmtNum'],
  setup(props) {
    const { t: bt } = useI18n()
    // Route through the locale-aware formatter passed down from the parent.
    const formatNum = (v) => v != null ? props.fmtNum(v) : '-'

    const getBandColor = () => {
      switch (props.band) {
        case 'F1': return 'text-danger-soft'
        case 'F2': return 'text-warning-soft'
        case 'F3': return 'text-positive-soft'
        default: return 'text-ink-soft'
      }
    }

    const getDiffColor = () => {
      if (props.difference == null) return 'text-ink-muted'
      const abs = Math.abs(props.difference)
      const threshold = props.threshold || 2
      if (abs > threshold * 2) return 'text-danger-soft'
      if (abs > threshold) return 'text-warning-soft'
      return 'text-positive-soft'
    }

    return () => h('div', { class: 'text-center p-2 bg-surface rounded-lg' }, [
      h('p', { class: ['text-xs font-medium mb-1', getBandColor()] }, props.band),
      h('div', { class: 'text-xs text-ink-muted' }, [
        h('p', {}, [bt('utilities.comparisonAccordion.providerLabel') + ': ', h('span', { class: 'font-medium text-ink-soft' }, formatNum(props.providerValue))]),
        h('p', {}, [bt('utilities.comparisonAccordion.yourLabel') + ' ', h('span', { class: 'font-medium text-ink-soft' }, formatNum(props.userValue))]),
        props.difference != null && h('p', { class: getDiffColor() },
          (props.difference > 0 ? '+' : '') + formatNum(props.difference) + ' ' + props.unit
        )
      ])
    ])
  }
}
</script>
