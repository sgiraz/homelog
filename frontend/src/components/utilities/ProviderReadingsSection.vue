<template>
  <div class="border border-info/30 bg-info/10 rounded-lg p-4">
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-info-soft" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
        <span class="text-sm font-medium text-info-soft">
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
          <p class="text-xs text-danger-soft font-medium">F1</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(f1) }}</p>
        </div>
        <div class="bg-surface rounded p-2">
          <p class="text-xs text-warning-soft font-medium">F2</p>
          <p class="text-sm font-semibold text-ink">{{ fmt(f2) }}</p>
        </div>
        <div class="bg-surface rounded p-2">
          <p class="text-xs text-positive-soft font-medium">F3</p>
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
      <p class="mt-2 text-xs text-center text-ink-muted">
        <span v-if="readingDate">{{ formatDate(readingDate, settingsStore.dateSettings) }} · </span>
        {{ t(`utilities.providerReadings.${readingType === 'estimated' ? 'estimated' : 'actual'}`) }}
        <span v-if="meterSerial"> · {{ meterSerial }}</span>
      </p>
    </div>

    <!-- Expanded form for manual entry -->
    <div v-show="isEditing" class="space-y-3">
      <p class="text-xs text-ink-muted">
        {{ t('utilities.providerReadings.instructions') }}
      </p>

      <!-- Electricity readings (F1/F2/F3) -->
      <div v-if="utilityType === 'electricity'" class="grid grid-cols-3 gap-2">
        <div>
          <label class="block text-xs text-danger-soft mb-1 font-medium">{{ t('utilities.providerReadings.f1Label') }}</label>
          <input v-model="f1" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-warning-soft mb-1 font-medium">{{ t('utilities.providerReadings.f2Label') }}</label>
          <input v-model="f2" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-positive-soft mb-1 font-medium">{{ t('utilities.providerReadings.f3Label') }}</label>
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
      </div>

      <!-- Water single reading -->
      <div v-else-if="utilityType === 'water'" class="space-y-3">
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.meterWater') }}</label>
          <input v-model="reading" type="number" step="0.001" placeholder="0"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2">
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.readingDate') }}</label>
          <input v-model="readingDate" type="date"
            class="w-full min-w-0 px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.readingType') }}</label>
          <select v-model="readingType"
            class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500">
            <option value="actual">{{ t('utilities.providerReadings.actual') }}</option>
            <option value="estimated">{{ t('utilities.providerReadings.estimated') }}</option>
          </select>
        </div>
      </div>
      <div>
        <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.providerReadings.meterSerial') }}</label>
        <input v-model="meterSerial" type="text"
          class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-blue-500" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { formatDate, formatNumber as _formatNumber } from '@/utils/dateFormatter'

const props = defineProps({
  utilityType: { type: String, required: true }
})

const f1 = defineModel('f1')
const f2 = defineModel('f2')
const f3 = defineModel('f3')
const reading = defineModel('reading')
const conversionCoefficient = defineModel('conversionCoefficient')
const readingDate = defineModel('readingDate')
const readingType = defineModel('readingType')
const meterSerial = defineModel('meterSerial')

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
