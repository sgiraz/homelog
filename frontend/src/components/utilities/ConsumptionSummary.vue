<template>
  <!-- Cumulative Summary Alert (only for overcharges) -->
  <div
    v-if="consumptionSummary?.has_cumulative_alert"
    :class="[
      'p-4 rounded-lg border-2',
      consumptionSummary.cumulative_alert_level === 'alert'
        ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20'
        : 'border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-900/20'
    ]"
  >
    <div class="flex items-start gap-3">
      <svg class="w-6 h-6 flex-shrink-0" :class="consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-500' : 'text-yellow-500'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      <div>
        <h4 :class="['font-semibold', consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-700 dark:text-red-300' : 'text-yellow-700 dark:text-yellow-300']">
          {{ consumptionSummary.cumulative_alert_level === 'alert' ? t('utilities.consumptionSummary.overcharge') : t('utilities.consumptionSummary.warning') }}
        </h4>
        <p :class="['text-sm mt-1', consumptionSummary.cumulative_alert_level === 'alert' ? 'text-red-600 dark:text-red-400' : 'text-yellow-600 dark:text-yellow-400']">
          {{ consumptionSummary.cumulative_message }}
        </p>
      </div>
    </div>
  </div>

  <!-- Informational message (provider charged less - not a problem) -->
  <div
    v-else-if="consumptionSummary?.cumulative_message && !consumptionSummary?.has_cumulative_alert"
    class="p-3 rounded-lg border border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20"
  >
    <div class="flex items-start gap-2">
      <svg class="w-5 h-5 flex-shrink-0 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p class="text-sm text-blue-700 dark:text-blue-300">
        {{ consumptionSummary.cumulative_message }}
      </p>
    </div>
  </div>

  <!-- Consumption Summary Card -->
  <div v-if="consumptionSummary && (consumptionSummary.total_user > 0 || consumptionSummary.total_provider > 0)" class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
    <h4 class="font-semibold text-gray-900 dark:text-white mb-3">{{ t('utilities.consumptionSummary.title') }}</h4>

    <div v-if="utilityType === 'electricity'" class="space-y-3">
      <div class="grid grid-cols-4 gap-2 sm:gap-3 text-sm">
        <div></div>
        <div class="text-center font-medium text-red-600 dark:text-red-400">F1</div>
        <div class="text-center font-medium text-yellow-600 dark:text-yellow-400">F2</div>
        <div class="text-center font-medium text-green-600 dark:text-green-400">F3</div>

        <div class="text-gray-600 dark:text-gray-400 text-xs sm:text-sm">{{ t('utilities.consumptionSummary.selfReadings') }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_user_f1) }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_user_f2) }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_user_f3) }}</div>

        <div class="text-gray-600 dark:text-gray-400 text-xs sm:text-sm">{{ t('utilities.consumptionSummary.provider') }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_provider_f1) }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_provider_f2) }}</div>
        <div class="text-center text-gray-900 dark:text-white text-xs sm:text-sm">{{ fmtNum(consumptionSummary.total_provider_f3) }}</div>

        <div class="text-gray-600 dark:text-gray-400 font-medium text-xs sm:text-sm">{{ t('utilities.consumptionSummary.difference') }}</div>
        <div :class="['text-center font-medium text-xs sm:text-sm', getDiffClass(consumptionSummary.cumulative_difference_f1)]">
          {{ fmtDiff(consumptionSummary.cumulative_difference_f1) }}
        </div>
        <div :class="['text-center font-medium text-xs sm:text-sm', getDiffClass(consumptionSummary.cumulative_difference_f2)]">
          {{ fmtDiff(consumptionSummary.cumulative_difference_f2) }}
        </div>
        <div :class="['text-center font-medium text-xs sm:text-sm', getDiffClass(consumptionSummary.cumulative_difference_f3)]">
          {{ fmtDiff(consumptionSummary.cumulative_difference_f3) }}
        </div>
      </div>

      <div :class="[
        'p-3 rounded grid grid-cols-2 sm:grid-cols-4 gap-2 text-sm',
        consumptionSummary.cumulative_difference > 1 ? 'bg-red-100 dark:bg-red-900/30' : 'bg-white dark:bg-gray-700'
      ]">
        <div class="text-gray-600 dark:text-gray-400 text-xs sm:text-sm sm:text-center">{{ t('utilities.consumptionSummary.total') }}</div>
        <div class="text-gray-900 dark:text-white text-xs sm:text-sm sm:text-center" v-html="totalSelfHtml"></div>
        <div class="text-gray-900 dark:text-white text-xs sm:text-sm sm:text-center" v-html="totalProviderHtml"></div>
        <div :class="['font-semibold text-xs sm:text-sm sm:text-center', getDiffClass(consumptionSummary.cumulative_difference)]">
          {{ consumptionSummary.cumulative_difference > 0 ? t('utilities.consumptionSummary.overchargeShort') : t('utilities.consumptionSummary.diffShort') }} {{ fmtDiff(consumptionSummary.cumulative_difference) }} kWh
        </div>
      </div>
    </div>

    <div v-else class="grid grid-cols-3 gap-3 text-sm">
      <div class="text-center p-3 bg-white dark:bg-gray-700 rounded">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">{{ t('utilities.consumptionSummary.selfReadings') }}</p>
        <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ fmtNum(consumptionSummary.total_user) }} {{ getUnit() }}</p>
      </div>
      <div class="text-center p-3 bg-white dark:bg-gray-700 rounded">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">{{ t('utilities.consumptionSummary.billed') }}</p>
        <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ fmtNum(consumptionSummary.total_provider) }} {{ getUnit() }}</p>
      </div>
      <div :class="[
        'text-center p-3 rounded',
        consumptionSummary.cumulative_difference > 1 ? 'bg-red-100 dark:bg-red-900/30' : 'bg-white dark:bg-gray-700'
      ]">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">
          {{ consumptionSummary.cumulative_difference > 0 ? t('utilities.consumptionSummary.overchargeShort') : t('utilities.consumptionSummary.difference') }}
        </p>
        <p :class="['text-lg font-semibold', getDiffClass(consumptionSummary.cumulative_difference)]">
          {{ fmtDiff(consumptionSummary.cumulative_difference) }} {{ getUnit() }}
        </p>
      </div>
    </div>

    <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
      {{ t('utilities.consumptionSummary.period', { start: formatDate(consumptionSummary.first_period), end: formatDate(consumptionSummary.last_period) }) }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ConsumptionSummary' })

const props = defineProps({
  consumptionSummary: {
    type: Object,
    default: null
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

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]))
}

const totalSelfHtml = computed(() => {
  const value = `<strong>${escapeHtml(props.fmtNum(props.consumptionSummary?.total_user))}</strong>`
  return t('utilities.consumptionSummary.totalSelfWithUnit', { value })
})

const totalProviderHtml = computed(() => {
  const value = `<strong>${escapeHtml(props.fmtNum(props.consumptionSummary?.total_provider))}</strong>`
  return t('utilities.consumptionSummary.totalProviderWithUnit', { value })
})
</script>
