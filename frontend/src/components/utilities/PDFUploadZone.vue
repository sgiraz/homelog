<template>
  <div>
    <!-- Template Selector -->
    <div v-if="availableTemplates.length > 0" class="mb-4">
      <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
        Template Estrazione
      </label>
      <select
        v-model="selectedTemplateId"
        class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
               bg-white dark:bg-gray-800 text-gray-900 dark:text-white
               focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
      >
        <option :value="null">Automatico (rileva dal fornitore)</option>
        <option
          v-for="tpl in availableTemplates"
          :key="tpl.id"
          :value="tpl.id"
        >
          {{ tpl.name }}{{ tpl.is_default ? ' (predefinito)' : '' }}
        </option>
      </select>
      <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
        Seleziona il template per estrarre i dati dal PDF
      </p>
    </div>

    <!-- PDF Drop Zone -->
    <div class="mb-6">
      <div
        :class="[
          'border-2 border-dashed rounded-xl p-6 text-center transition-all cursor-pointer',
          isDragging
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500',
          processing ? 'opacity-50 pointer-events-none' : ''
        ]"
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="handleDrop"
        @click="triggerFileInput"
      >
        <input
          ref="fileInput"
          type="file"
          accept=".pdf"
          class="hidden"
          @change="handleFileSelect"
        />

        <div v-if="processing" class="flex flex-col items-center gap-2">
          <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span class="text-sm text-gray-600 dark:text-gray-400">Estrazione dati dal PDF...</span>
        </div>

        <div v-else-if="uploadedFile" class="flex items-center justify-center gap-3">
          <svg class="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div class="text-left">
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ uploadedFile.name }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">Dati estratti automaticamente</p>
          </div>
          <button
            type="button"
            @click.stop="clearUploadedFile"
            class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-else class="flex flex-col items-center gap-2">
          <svg class="w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
          <div>
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
              Trascina qui il PDF della {{ isMetered ? 'bolletta' : 'fattura' }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              oppure clicca per selezionare
            </p>
          </div>
        </div>
      </div>

      <div v-if="error" class="mt-2 text-sm text-red-600 dark:text-red-400">
        {{ error }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { utilitiesAPI, templatesAPI } from '@/api/client'
import apiClient from '@/api/client'

defineOptions({ name: 'PDFUploadZone' })

const props = defineProps({
  utilityId: { type: [Number, String], required: true },
  utilityType: { type: String, required: true },
  isMetered: { type: Boolean, default: true },
  billingInterval: { type: Number, default: null },
  billingUnit: { type: String, default: null },
})

const emit = defineEmits(['extracted'])

const fileInput = ref(null)
const isDragging = ref(false)
const processing = ref(false)
const error = ref(null)
const uploadedFile = ref(null)

const availableTemplates = ref([])
const selectedTemplateId = ref(null)

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return ''
  return date.toISOString().split('T')[0]
}

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (file) processFile(file)
}

function handleDrop(event) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file && file.type === 'application/pdf') {
    processFile(file)
  } else {
    error.value = 'Per favore carica un file PDF'
  }
}

function inferMissingPeriodDate(data) {
  if (!props.billingInterval || !props.billingUnit) return data
  if (!data.period_start || data.period_end) return data

  const interval = props.billingInterval
  const unit = props.billingUnit
  if (!['day', 'week', 'month', 'year'].includes(unit)) return data

  const parts = data.period_start.split('-')
  const year = parseInt(parts[0]), month = parseInt(parts[1]) - 1, day = parseInt(parts[2])
  const end = new Date(Date.UTC(year, month, day))
  if (isNaN(end.getTime())) return data

  switch (unit) {
    case 'day': end.setUTCDate(end.getUTCDate() + interval); break
    case 'week': end.setUTCDate(end.getUTCDate() + interval * 7); break
    case 'month': end.setUTCMonth(end.getUTCMonth() + interval); break
    case 'year': end.setUTCFullYear(end.getUTCFullYear() + interval); break
  }
  end.setUTCDate(end.getUTCDate() - 1)
  return { ...data, period_end: end.toISOString().split('T')[0] }
}

async function processFile(file) {
  if (file.type !== 'application/pdf') {
    error.value = 'Per favore carica un file PDF'
    return
  }

  processing.value = true
  error.value = null

  try {
    const { data } = await utilitiesAPI.uploadBillPDF(props.utilityId, file, selectedTemplateId.value)
    uploadedFile.value = file

    if (data) {
      // Normalize date fields to YYYY-MM-DD
      const normalized = { ...data }
      for (const key of ['period_start', 'period_end', 'due_date', 'issue_date', 'provider_reading_date', 'estimated_date']) {
        if (normalized[key]) normalized[key] = formatDateForInput(normalized[key])
      }

      // Infer missing period date for fixed services
      const result = !props.isMetered ? inferMissingPeriodDate(normalized) : normalized
      emit('extracted', result)
    }
  } catch (err) {
    error.value = err.response?.data?.error || 'Errore durante l\'estrazione dei dati dal PDF'
    console.error('PDF extraction error:', err)
  } finally {
    processing.value = false
  }
}

function clearUploadedFile() {
  uploadedFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function loadTemplates() {
  try {
    const { data } = await templatesAPI.listBillTemplates()
    availableTemplates.value = data.filter(t => t.utility_type === props.utilityType)

    try {
      const { data: settings } = await apiClient.get('/settings')
      if (settings.default_templates) {
        const defaultTemplates = JSON.parse(settings.default_templates)
        const defaultId = defaultTemplates[props.utilityType]
        if (defaultId && availableTemplates.value.some(t => t.id === defaultId)) {
          selectedTemplateId.value = defaultId
        }
      }
    } catch { /* Ignore settings error */ }
  } catch (err) {
    console.error('Error loading templates:', err)
  }
}

onMounted(() => {
  loadTemplates()
})
</script>
