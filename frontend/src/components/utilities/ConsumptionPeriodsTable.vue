<template>
  <!-- Consumption Periods Table -->
  <div v-if="consumptionAnalysis && consumptionAnalysis.length > 0" class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
    <h4 class="font-semibold text-gray-900 dark:text-white mb-3">Dettaglio Consumi per Periodo</h4>

    <div class="overflow-x-auto -mx-2">
      <table class="w-full text-xs sm:text-sm min-w-0">
        <thead>
          <tr class="border-b border-gray-300 dark:border-gray-600">
            <th class="text-left py-2 px-1.5 text-gray-600 dark:text-gray-400">Periodo</th>
            <th class="text-right py-2 px-1.5 text-gray-600 dark:text-gray-400">
              <span class="hidden sm:inline">Consumo </span>Effettivo
            </th>
            <th class="text-right py-2 px-1.5 text-gray-600 dark:text-gray-400">
              <span class="hidden sm:inline">Consumo </span>Fatturato
            </th>
            <th class="text-right py-2 px-1.5 text-gray-600 dark:text-gray-400 w-16 sm:w-auto">Diff.</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(period, idx) in consumptionAnalysis"
            :key="idx"
            class="border-b border-gray-200 dark:border-gray-700"
          >
            <td class="py-2 px-1.5 text-gray-900 dark:text-white whitespace-nowrap">
              {{ formatPeriodCompact(period.period_start, period.period_end) }}
            </td>
            <td class="py-2 px-1.5 text-right text-gray-900 dark:text-white whitespace-nowrap tabular-nums">
              {{ period.user_consumption != null ? fmtNum(period.user_consumption) : '-' }}
            </td>
            <td class="py-2 px-1.5 text-right text-gray-900 dark:text-white whitespace-nowrap tabular-nums">
              {{ period.provider_consumption != null ? fmtNum(period.provider_consumption) : '-' }}
            </td>
            <td :class="['py-2 px-1.5 text-right font-medium whitespace-nowrap tabular-nums', getDiffClass(period.difference)]">
              {{ period.difference != null ? fmtDiff(period.difference) : '-' }}
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="border-t-2 border-gray-400 dark:border-gray-500 font-semibold">
            <td class="py-2 px-1.5 text-gray-900 dark:text-white">TOTALE</td>
            <td class="py-2 px-1.5 text-right text-gray-900 dark:text-white whitespace-nowrap tabular-nums">
              {{ fmtNum(consumptionSummary?.total_user || 0) }} <span class="text-gray-400 font-normal">{{ getUnit() }}</span>
            </td>
            <td class="py-2 px-1.5 text-right text-gray-900 dark:text-white whitespace-nowrap tabular-nums">
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
</script>
