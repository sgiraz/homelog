<template>
  <div class="space-y-4">
    <p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
      Carica un PDF di esempio del fornitore per creare le regole di estrazione.
    </p>

    <!-- Basic Info -->
    <Input
      :model-value="template.name"
      @update:model-value="emit('update:template', { ...template, name: $event })"
      label="Nome Template *"
      placeholder="Es: Fornitore Luce Bimestrale"
      required
    />

    <Input
      :model-value="template.provider"
      @update:model-value="emit('update:template', { ...template, provider: $event })"
      label="Fornitore *"
      placeholder="Nome del fornitore"
      required
    />

    <div>
      <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
        Tipo Utenza *
      </label>
      <select
        :value="template.utility_type"
        @change="emit('update:template', { ...template, utility_type: $event.target.value })"
        class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
               bg-white dark:bg-gray-800 text-gray-900 dark:text-white
               focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <option value="electricity">Luce</option>
        <option value="gas">Gas</option>
        <option value="water">Acqua</option>
        <option value="waste">Rifiuti</option>
        <option value="internet">Internet</option>
        <option value="insurance">Assicurazione</option>
        <option value="affitto">Affitto</option>
        <option value="mutuo">Mutuo</option>
      </select>
    </div>

    <!-- PDF Upload -->
    <div
      :class="[
        'border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer',
        isDraggingFile
          ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
          : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500',
        extracting ? 'opacity-50 pointer-events-none' : ''
      ]"
      @dragover.prevent="isDraggingFile = true"
      @dragleave.prevent="isDraggingFile = false"
      @drop.prevent="handleFileDrop"
      @click="emit('trigger-file-input')"
    >
      <div v-if="extracting" class="flex flex-col items-center gap-2">
        <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span class="text-sm text-gray-600 dark:text-gray-400">Analisi PDF in corso...</span>
      </div>

      <div v-else-if="pdfFile" class="flex flex-col items-center gap-2">
        <svg class="w-10 h-10 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ pdfFile.name }}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ pdfAnalysis?.page_count || 0 }} pagine estratte - Clicca per cambiare file
        </p>
      </div>

      <div v-else class="flex flex-col items-center gap-2">
        <svg class="w-12 h-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
        </svg>
        <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
          Trascina qui un PDF di esempio
        </p>
        <p class="text-xs text-gray-500 dark:text-gray-400">oppure clicca per selezionare</p>
      </div>
    </div>

    <div v-if="extractError" class="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
      {{ extractError }}
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Input from '@/components/common/Input.vue'

defineOptions({ name: 'WizardStepInfo' })

defineProps({
  template: {
    type: Object,
    required: true
  },
  pdfFile: {
    type: Object,
    default: null
  },
  pdfAnalysis: {
    type: Object,
    default: null
  },
  extracting: {
    type: Boolean,
    default: false
  },
  extractError: {
    type: String,
    default: null
  }
})

const emit = defineEmits(['update:template', 'trigger-file-input', 'file-dropped'])

const isDraggingFile = ref(false)

function handleFileDrop(event) {
  isDraggingFile.value = false
  emit('file-dropped', event)
}
</script>
