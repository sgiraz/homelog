<template>
  <div class="space-y-4">
    <div class="text-center py-4">
      <svg class="w-16 h-16 mx-auto text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <h4 class="text-lg font-bold text-gray-900 dark:text-white mt-4">Template Pronto</h4>
      <p class="text-gray-600 dark:text-gray-400 mt-2">
        Il template per <strong>{{ template.provider }}</strong> è pronto per essere salvato.
      </p>
    </div>

    <!-- Summary -->
    <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-2">
      <div class="flex justify-between">
        <span class="text-gray-600 dark:text-gray-400">Nome:</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ template.name }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-gray-600 dark:text-gray-400">Fornitore:</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ template.provider }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-gray-600 dark:text-gray-400">Tipo:</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ getUtilityTypeName(template.utility_type) }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-gray-600 dark:text-gray-400">Campi mappati:</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ Object.keys(mappings).length }}</span>
      </div>
    </div>

    <!-- Mapped Fields Summary -->
    <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
      <h5 class="text-sm font-medium text-gray-900 dark:text-white mb-3">Regole di Estrazione</h5>
      <div class="space-y-2">
        <div
          v-for="(mapping, fieldKey) in mappings"
          :key="fieldKey"
          class="flex items-center justify-between py-1 border-b border-gray-100 dark:border-gray-700 last:border-0"
        >
          <span class="text-sm text-gray-600 dark:text-gray-400">{{ getFieldLabel(fieldKey) }}</span>
          <span class="text-sm font-mono text-gray-900 dark:text-white">{{ mapping.token.text }}</span>
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
      <label for="is-default" class="text-sm text-gray-900 dark:text-white cursor-pointer">
        Usa come template predefinito per {{ template.provider }}
      </label>
    </div>
  </div>
</template>

<script setup>
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

function getUtilityTypeName(type) {
  const names = {
    electricity: 'Luce',
    gas: 'Gas',
    water: 'Acqua',
    waste: 'Rifiuti',
    internet: 'Internet',
    insurance: 'Assicurazione',
    affitto: 'Affitto',
    mutuo: 'Mutuo'
  }
  return names[type] || type
}
</script>
