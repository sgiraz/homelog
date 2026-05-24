<template>
  <div class="border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
        <span class="text-sm font-medium text-blue-700 dark:text-blue-300">
          {{ t('utilities.providerReadings.title') }}
        </span>
      </div>
      <button
        type="button"
        @click="isEditing = !isEditing"
        class="text-xs text-blue-600 dark:text-blue-400 hover:underline"
      >
        {{ isEditing ? t('utilities.providerReadings.hide') : t('utilities.providerReadings.edit') }}
      </button>
    </div>

    <!-- Collapsed view when readings exist -->
    <div v-show="hasReadings && !isEditing">
      <div v-if="utilityType === 'electricity'" class="grid grid-cols-3 gap-2 text-center">
        <div class="bg-surface rounded p-2">
          <p class="text-xs text-red-600 dark:text-red-400 font-medium">F1</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(f1) }}</p>
        </div>
        <div class="bg-surface rounded p-2">
          <p class="text-xs text-yellow-600 dark:text-yellow-400 font-medium">F2</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(f2) }}</p>
        </div>
        <div class="bg-surface rounded p-2">
          <p class="text-xs text-green-600 dark:text-green-400 font-medium">F3</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(f3) }}</p>
        </div>
      </div>
      <div v-else-if="utilityType === 'gas'" class="space-y-1">
        <div class="text-center bg-surface rounded p-2">
          <p class="text-xs text-ink-muted">{{ t('utilities.providerReadings.readingLabel') }}</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(reading) }} mc</p>
        </div>
        <div v-if="conversionCoefficient" class="text-center bg-surface rounded p-2">
          <p class="text-xs text-ink-muted">{{ t('utilities.providerReadings.coefficientLabel', { value: conversionCoefficient }) }}</p>
        </div>
      </div>
      <div v-else class="text-center bg-surface rounded p-2">
        <p class="text-sm font-semibold text-ink">{{ fmt(reading) }} mc</p>
      </div>
    </div>

    <!-- Expanded form for manual entry -->
    <div v-show="isEditing" class="space-y-3">
      <p class="text-xs text-ink-muted">
        {{ t('utilities.providerReadings.instructions') }}
      </p>

      <!-- Electricity readings (F1/F2/F3) -->
      <div v-if="utilityType === 'electricity'" class="grid grid-cols-3 gap-2">
        <div>
          <label class="block text-xs text-red-600 dark:text-red-400 mb-1 font-medium">{{ t('utilities.providerReadings.f1Label') }}</label>
          <input v-model="f1" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-yellow-600 dark:text-yellow-400 mb-1 font-medium">{{ t('utilities.providerReadings.f2Label') }}</label>
          <input v-model="f2" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-green-600 dark:text-green-400 mb-1 font-medium">{{ t('utilities.providerReadings.f3Label') }}</label>
          <input v-model="f3" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
      </div>

      <!-- Gas: reading + conversion coefficient -->
      <div v-else-if="utilityType === 'gas'" class="space-y-3">
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.meterGas') }}</label>
          <input v-model="reading" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.conversionCoeff') }}</label>
          <input v-model="conversionCoefficient" type="number" step="0.00000001" min="0" placeholder="1.00000000"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div v-if="previousBillHasEstimate && !previousBill?.estimated_reading">
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.previousEstimatedSmc') }}</label>
          <input v-model="previousEstimatedConsumption" type="number" step="0.000001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
          <p class="text-xs text-ink-faint mt-0.5">{{ t('utilities.providerReadings.previousEstimateHintSmc', { value: fmt(previousBill.estimated_consumption) }) }}</p>
        </div>
      </div>

      <!-- Water single reading -->
      <div v-else-if="utilityType === 'water'" class="space-y-3">
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.meterWater') }}</label>
          <input v-model="reading" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div v-if="previousBillHasEstimate && !previousBill?.estimated_reading">
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.previousEstimatedMc') }}</label>
          <input v-model="previousEstimatedConsumption" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
          <p class="text-xs text-ink-faint mt-0.5">{{ t('utilities.providerReadings.previousEstimateHintMc', { value: fmt(previousBill.estimated_consumption) }) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { formatNumber as _formatNumber } from '@/utils/dateFormatter'

const props = defineProps({
  utilityType: { type: String, required: true },
  previousBill: { type: Object, default: null },
  previousBillHasEstimate: { type: Boolean, default: false }
})

const f1 = defineModel('f1')
const f2 = defineModel('f2')
const f3 = defineModel('f3')
const reading = defineModel('reading')
const conversionCoefficient = defineModel('conversionCoefficient')
const previousEstimatedConsumption = defineModel('previousEstimatedConsumption')

const { t } = useI18n()
const settingsStore = useSettingsStore()
const isEditing = ref(true)

const hasReadings = computed(() => {
  if (props.utilityType === 'electricity') {
    return f1.value != null || f2.value != null || f3.value != null
  }
  if (props.utilityType === 'gas') {
    return reading.value != null || conversionCoefficient.value != null
  }
  return reading.value != null
})

function fmt(value) {
  if (value == null) return '-'
  return _formatNumber(value, settingsStore.formatSettings)
}

onMounted(() => {
  if (hasReadings.value) isEditing.value = false
})
</script>
