<template>
  <div v-if="consumptionAnalysis && consumptionAnalysis.length > 0" class="p-4 bg-surface rounded-lg">
    <h4 class="font-semibold text-ink mb-3">{{ t('utilities.consumptionPeriods.title') }}</h4>

    <div class="overflow-x-auto -mx-2">
      <table class="w-full text-xs sm:text-sm min-w-0">
        <thead>
          <tr class="border-b border-line">
            <th class="text-left py-2 px-1.5 text-ink-soft">{{ t('utilities.consumptionPeriods.period') }}</th>
            <th class="text-right py-2 px-1.5 text-ink-soft">
              <span class="hidden sm:inline">{{ t('utilities.consumptionPeriods.consumptionPrefix') }} </span>{{ t('utilities.consumptionPeriods.actual') }}
            </th>
            <th class="text-right py-2 px-1.5 text-ink-soft">
              <span class="hidden sm:inline">{{ t('utilities.consumptionPeriods.consumptionPrefix') }} </span>{{ t('utilities.consumptionPeriods.billed') }}
            </th>
            <th class="text-right py-2 px-1.5 text-ink-soft w-16 sm:w-auto">{{ t('utilities.consumptionPeriods.diff') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(period, idx) in consumptionAnalysis"
            :key="idx"
            class="border-b border-line"
          >
            <td class="py-2 px-1.5 text-ink whitespace-nowrap">
              {{ formatPeriodCompact(period.period_start, period.period_end) }}
            </td>
            <td class="py-2 px-1.5 text-right text-ink whitespace-nowrap tabular-nums">
              {{ period.user_consumption != null ? fmtNum(period.user_consumption) : '-' }}
            </td>
            <td class="py-2 px-1.5 text-right text-ink whitespace-nowrap tabular-nums">
              {{ period.provider_consumption != null ? fmtNum(period.provider_consumption) : '-' }}
            </td>
            <td :class="['py-2 px-1.5 text-right font-medium whitespace-nowrap tabular-nums', getDiffClass(period.difference)]">
              {{ period.difference != null ? fmtDiff(period.difference) : '-' }}
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="border-t-2 border-ink-faint font-semibold">
            <td class="py-2 px-1.5 text-ink">{{ t('utilities.consumptionPeriods.total') }}</td>
            <td class="py-2 px-1.5 text-right text-ink whitespace-nowrap tabular-nums">
              {{ fmtNum(consumptionSummary?.total_user || 0) }} <span class="text-gray-400 font-normal">{{ getUnit() }}</span>
            </td>
            <td class="py-2 px-1.5 text-right text-ink whitespace-nowrap tabular-nums">
              {{ fmtNum(consumptionSummary?.total_provider || 0) }} <span class="text-gray-400 font-normal">{{ getUnit() }}</span>
            </td>
            <td :class="['py-2 px-1.5 text-right whitespace-nowrap tabular-nums', getDiffClass(consumptionSummary?.cumulative_difference)]">
              {{ fmtDiff(consumptionSummary?.cumulative_difference || 0) }}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ConsumptionPeriodsTable' })

defineProps({
  consumptionAnalysis: {
    type: Array,
    required: true
  },
  utilityType: {
    type: String,
    required: true
  },
  consumptionSummary: {
    type: Object,
    default: null
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
  formatPeriodCompact: {
    type: Function,
    required: true
  },
  getDiffClass: {
    type: Function,
    required: true
  },
  getUnit: {
    type: Function,
    required: true
  }
})

const { t } = useI18n()
</script>
