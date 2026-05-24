<template>
  <div class="space-y-4">
    <div class="text-center py-4">
      <svg class="w-16 h-16 mx-auto text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <h4 class="text-lg font-bold text-ink mt-4">{{ t('utilities.wizardStepReview.ready') }}</h4>
      <i18n-t keypath="utilities.wizardStepReview.readyDescription" tag="p" class="text-ink-soft mt-2">
        <template #provider><strong>{{ template.provider }}</strong></template>
      </i18n-t>
    </div>

    <!-- Summary -->
    <div class="bg-surface rounded-lg p-4 space-y-2">
      <div class="flex justify-between">
        <span class="text-ink-soft">{{ t('utilities.wizardStepReview.summaryName') }}</span>
        <span class="font-medium text-ink">{{ template.name }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-ink-soft">{{ t('utilities.wizardStepReview.summaryProvider') }}</span>
        <span class="font-medium text-ink">{{ template.provider }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-ink-soft">{{ t('utilities.wizardStepReview.summaryType') }}</span>
        <span class="font-medium text-ink">{{ getUtilityTypeName(template.utility_type) }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-ink-soft">{{ t('utilities.wizardStepReview.summaryFieldsMapped') }}</span>
        <span class="font-medium text-ink">{{ Object.keys(mappings).length }}</span>
      </div>
    </div>

    <!-- Mapped Fields Summary -->
    <div class="border border-line rounded-lg p-4">
      <h5 class="text-sm font-medium text-ink mb-3">{{ t('utilities.wizardStepReview.rulesTitle') }}</h5>
      <div class="space-y-2">
        <div
          v-for="(mapping, fieldKey) in mappings"
          :key="fieldKey"
          class="flex items-center justify-between py-1 border-b border-line last:border-0"
        >
          <span class="text-sm text-ink-soft">{{ getFieldLabel(fieldKey) }}</span>
          <span class="text-sm font-mono text-ink">{{ mapping.token.text }}</span>
        </div>
      </div>
    </div>

    <div class="flex items-center gap-3 pt-2">
      <input
        type="checkbox"
        id="is-default"
        :checked="template.is_default"
        @change="emit('update:template', { ...template, is_default: $event.target.checked })"
        class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
      />
      <label for="is-default" class="text-sm text-ink cursor-pointer">
        {{ t('utilities.wizardStepReview.useAsDefault', { provider: template.provider }) }}
      </label>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'WizardStepReview' })

defineProps({
  template: {
    type: Object,
    required: true
  },
  mappings: {
    type: Object,
    required: true
  },
  extractionFields: {
    type: Array,
    required: true
  },
  getFieldLabel: {
    type: Function,
    required: true
  }
})

const emit = defineEmits(['update:template'])

const { t } = useI18n()

function getUtilityTypeName(type) {
  const key = `utilities.utilityTypes.${type}`
  const label = t(key)
  return label === key ? type : label
}
</script>
