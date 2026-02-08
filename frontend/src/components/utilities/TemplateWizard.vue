<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[70] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-6xl p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ isEditing ? 'Modifica Template' : 'Crea Template Estrazione' }}
        </h3>
        <button @click="handleClose" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Step Indicators -->
      <div class="flex items-center justify-center pb-10 pt-4">
        <div
          v-for="(stepInfo, index) in steps"
          :key="index"
          class="flex items-center"
        >
          <div class="flex flex-col items-center relative">
            <div
              :class="[
                'w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors z-10',
                step > index + 1
                  ? 'bg-green-500 text-white'
                  : step === index + 1
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
              ]"
            >
              <svg v-if="step > index + 1" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <span v-else>{{ index + 1 }}</span>
            </div>

            <span
              class="absolute -bottom-7 text-[10px] uppercase tracking-wider font-bold whitespace-nowrap"
              :class="step === index + 1 ? 'text-blue-500' : 'text-gray-400'"
            >
              {{ stepInfo }}
            </span>
          </div>

          <div
            v-if="index < steps.length - 1"
            :class="[
              'w-16 h-1 mx-2 mb-0',
              step > index + 1 ? 'bg-green-500' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          />
        </div>
      </div>

      <!-- Step 1: Upload PDF & Basic Info -->
      <div v-if="step === 1" class="space-y-4">
        <p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
          Carica un PDF di esempio del fornitore per creare le regole di estrazione.
        </p>

        <!-- Basic Info -->
        <Input
          v-model="template.name"
          label="Nome Template *"
          placeholder="Es: Fornitore Luce Bimestrale"
          required
        />

        <Input
          v-model="template.provider"
          label="Fornitore *"
          placeholder="Nome del fornitore"
          required
        />

        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Tipo Utenza *
          </label>
          <select
            v-model="template.utility_type"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="electricity">Luce</option>
            <option value="gas">Gas</option>
            <option value="water">Acqua</option>
            <option value="waste">Rifiuti</option>
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
          @click="triggerFileInput"
        >
          <input
            ref="fileInput"
            type="file"
            accept=".pdf"
            class="hidden"
            @change="handleFileSelect"
          />

          <div v-if="extracting" class="flex flex-col items-center gap-2">
            <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="text-sm text-gray-600 dark:text-gray-400">Analisi PDF in corso...</span>
          </div>

          <div v-else-if="uploadedFile" class="flex flex-col items-center gap-2">
            <svg class="w-10 h-10 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ uploadedFile.name }}</p>
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

      <!-- Step 2: Drag & Drop Field Mapping with PDF Textract View -->
      <div v-if="step === 2" class="space-y-4">
        <p class="text-gray-600 dark:text-gray-400 text-sm">
          Trascina i valori dal PDF ai campi corrispondenti sulla destra.
        </p>

        <div class="flex gap-4" style="height: 500px;">
          <!-- Left Panel: PDF Textract View -->
          <div class="flex-1 flex flex-col border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
            <PDFTextractView
              :pages="pdfAnalysis?.pages || []"
              :loading="extracting"
              :mapped-word-ids="mappedTokenIds"
              @drag-start="handleDragStart"
              @drag-end="handleDragEnd"
            />
          </div>

          <!-- Right Panel: Drop Zones for Fields -->
          <div class="w-72 flex flex-col">
            <h4 class="font-medium text-gray-900 dark:text-white text-sm mb-2">Campi da Mappare</h4>
            <div class="flex-1 space-y-2 overflow-y-auto pr-1">
              <div
                v-for="field in extractionFields"
                :key="field.key"
                :class="[
                  'border rounded-lg p-3 transition-all',
                  dragOverField === field.key
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : mappings[field.key]
                      ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/10'
                      : 'border-gray-200 dark:border-gray-700'
                ]"
                @dragover.prevent="dragOverField = field.key"
                @dragleave.prevent="dragOverField = null"
                @drop.prevent="handleFieldDrop($event, field.key)"
              >
                <div class="flex items-center justify-between mb-1">
                  <span class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ field.label }}
                    <span v-if="field.required" class="text-red-500">*</span>
                  </span>
                  <button
                    v-if="mappings[field.key]"
                    @click="clearMapping(field.key)"
                    class="text-gray-400 hover:text-red-500 transition-colors"
                    title="Rimuovi mappatura"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>

                <!-- Drop zone content -->
                <div v-if="mappings[field.key]" class="space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="px-2 py-1 rounded text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">
                      {{ mappings[field.key].token.text }}
                    </span>
                    <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <!-- Anchor/strategy feedback -->
                  <div v-if="mappings[field.key].globalSearch" class="text-[10px] text-blue-500 dark:text-blue-400 truncate" title="Ricerca globale nel documento">
                    Ricerca globale
                  </div>
                  <div v-else-if="mappings[field.key].anchorText" class="text-[10px] text-gray-400 dark:text-gray-500 truncate" :title="'Ancora: &quot;' + mappings[field.key].anchorText + '&quot; → ' + getDirectionLabel(mappings[field.key].anchorDirection)">
                    {{ mappings[field.key].anchorText }} → {{ getDirectionLabel(mappings[field.key].anchorDirection) }}
                  </div>

                  <!-- Context Editor (collapsible) -->
                  <div v-if="mappings[field.key].allNeighbors" class="mt-1">
                    <button @click="toggleContextEditor(field.key)"
                            class="text-xs text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1">
                      <svg class="w-3 h-3 transition-transform" :class="contextEditorOpen === field.key ? 'rotate-90' : ''"
                           fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                      </svg>
                      Contesto {{ getContextSummary(field.key) }}
                    </button>

                    <div v-if="contextEditorOpen === field.key" class="mt-1.5 space-y-1.5 text-xs">
                      <div v-for="dir in ['left', 'above', 'right']" :key="dir">
                        <div v-if="getNeighborWords(field.key, dir).length > 0"
                             class="flex flex-wrap items-center gap-1">
                          <span class="text-gray-400 dark:text-gray-500 w-8 text-[10px] flex-shrink-0">{{ directionLabels[dir] }}</span>
                          <button v-for="word in getNeighborWords(field.key, dir)" :key="word.id"
                                  @click="toggleContextWord(field.key, word, dir)"
                                  :class="[
                                    'px-1.5 py-0.5 rounded border text-[11px] transition-all cursor-pointer',
                                    isContextWordSelected(field.key, word.id)
                                      ? 'bg-blue-100 border-blue-400 text-blue-700 dark:bg-blue-900/40 dark:border-blue-500 dark:text-blue-300'
                                      : 'bg-gray-50 border-gray-300 text-gray-500 hover:border-blue-300 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:border-blue-500'
                                  ]">
                            {{ word.text }}
                          </button>
                        </div>
                      </div>

                      <!-- Pattern preview -->
                      <div class="mt-1 p-1.5 bg-gray-100 dark:bg-gray-800 rounded font-mono text-[10px] text-gray-600 dark:text-gray-400 break-all">
                        {{ mappings[field.key].pattern }}
                      </div>
                    </div>
                  </div>
                </div>
                <div
                  v-else
                  class="text-gray-400 dark:text-gray-500 text-sm py-2 text-center border-2 border-dashed border-gray-200 dark:border-gray-600 rounded"
                >
                  Trascina qui...
                </div>
              </div>
            </div>

            <!-- Test Pattern Button -->
            <div class="pt-3 mt-3 border-t border-gray-200 dark:border-gray-700">
              <Button
                @click="testAllPatterns"
                :disabled="testing || Object.keys(mappings).length === 0"
                variant="secondary"
                size="sm"
                class="w-full"
              >
                {{ testing ? 'Test in corso...' : 'Testa Pattern' }}
              </Button>
            </div>
          </div>
        </div>

        <!-- Test Results -->
        <div v-if="hasTestedPatterns && Object.keys(testResults).length > 0"
             class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 space-y-2">
          <h5 class="text-sm font-medium text-gray-900 dark:text-white mb-2">Risultati Test</h5>
          <div v-for="(result, fieldKey) in testResults" :key="fieldKey" class="flex flex-col gap-1 text-sm py-1 border-b border-gray-200 dark:border-gray-700 last:border-0">
            <div class="flex items-center gap-2">
              <svg v-if="result.success" class="w-4 h-4 text-green-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <svg v-else class="w-4 h-4 text-orange-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span class="text-gray-600 dark:text-gray-400">{{ getFieldLabel(fieldKey) }}:</span>
              <span :class="result.success ? 'text-green-600 dark:text-green-400 font-medium' : 'text-orange-600 dark:text-orange-400'">
                {{ result.success ? result.value : result.error }}
              </span>
            </div>
            <!-- Show position context if available -->
            <div v-if="result.hasPosition" class="text-xs text-gray-400 dark:text-gray-500 ml-6">
              Pag. {{ result.page + 1 }}, pos: ({{ Math.round(result.x) }}, {{ Math.round(result.y) }})
              <span v-if="result.contextLeft"> | ← "{{ result.contextLeft }}"</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Step 3: Review & Save -->
      <div v-if="step === 3" class="space-y-4">
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
            v-model="template.is_default"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <label for="is-default" class="text-sm text-gray-900 dark:text-white cursor-pointer">
            Usa come template predefinito per {{ template.provider }}
          </label>
        </div>
      </div>

      <!-- Error -->
      <div v-if="error" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <!-- Navigation Buttons -->
      <div class="flex gap-3 mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
        <Button
          v-if="step > 1"
          type="button"
          variant="secondary"
          @click="step--"
          class="flex-1"
        >
          Indietro
        </Button>
        <Button
          v-if="step === 1"
          type="button"
          variant="secondary"
          @click="handleClose"
          class="flex-1"
        >
          Annulla
        </Button>
        <Button
          v-if="step < 3"
          @click="nextStep"
          :disabled="!canProceed"
          class="flex-1"
        >
          Avanti
        </Button>
        <Button
          v-if="step === 3"
          @click="saveTemplate"
          :disabled="saving"
          class="flex-1"
        >
          {{ saving ? 'Salvataggio...' : 'Salva Template' }}
        </Button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { templatesAPI, pdfAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'
import PDFTextractView from './PDFTextractView.vue'
import { tokenizePDFText, getTokenColorClass } from '@/utils/tokenizer'
import { generatePatternForField, regeneratePatternWithContext, getNeighborWordsForToken, testPattern } from '@/utils/patternGenerator'

const props = defineProps({
  existingTemplate: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])

// Step management
const step = ref(1)
const steps = ['Info & PDF', 'Mappa Campi', 'Salva']

// File upload state
const fileInput = ref(null)
const isDraggingFile = ref(false)
const extracting = ref(false)
const extractError = ref(null)
const uploadedFile = ref(null)
const rawText = ref('')

// PDF Analysis data (from /pdf/analyze endpoint)
const pdfAnalysis = ref(null)
const analysisTimestamp = ref(null)

// Drag and drop state
const dragOverField = ref(null)
const draggingToken = ref(null)

// Field mappings: { fieldKey: { token, pattern, prefix, suffix } }
const mappings = reactive({})

// Context editor state
const contextEditorOpen = ref(null) // field.key of open editor, or null
const directionLabels = { left: '←', above: '↑', right: '→' }

// Testing state
const testing = ref(false)
const hasTestedPatterns = ref(false)
const testResults = ref({})

// Save state
const saving = ref(false)
const error = ref(null)

const isEditing = computed(() => !!props.existingTemplate)

const template = ref({
  name: '',
  provider: '',
  utility_type: 'electricity',
  is_default: false
})

// Fields available for extraction (dynamic based on utility type)
// consumption_total is always calculated from readings, never extracted from PDF
const baseExtractionFields = [
  { key: 'amount_total', label: 'Importo Totale', required: true },
  { key: 'bill_number', label: 'Numero Bolletta', required: false },
  { key: 'period_start', label: 'Data Inizio Periodo', required: false },
  { key: 'period_end', label: 'Data Fine Periodo', required: false },
  { key: 'due_date', label: 'Data Scadenza', required: false },
  { key: 'issue_date', label: 'Data Emissione', required: false }
]

const gasExtraFields = [
  { key: 'provider_reading', label: 'Lettura Contatore (mc)', required: false },
  { key: 'conversion_coefficient', label: 'Coeff. Conversione (C)', required: false },
  { key: 'estimated_date', label: 'Data Stima', required: false },
  { key: 'estimated_reading', label: 'Lettura Stimata (mc)', required: false },
  { key: 'previous_estimated_consumption', label: 'Consumi Precedenti Stimati (Smc)', required: false }
]

const waterExtraFields = [
  { key: 'provider_reading', label: 'Lettura Contatore (mc)', required: false }
]

const extractionFields = computed(() => {
  if (template.value.utility_type === 'gas') {
    return [...baseExtractionFields, ...gasExtraFields]
  }
  if (template.value.utility_type === 'water') {
    return [...baseExtractionFields, ...waterExtraFields]
  }
  return baseExtractionFields
})

// Computed: tokens that have been mapped
const mappedTokenIds = computed(() => {
  const ids = new Set()
  Object.values(mappings).forEach(m => {
    if (m && m.token) {
      ids.add(m.token.id)
    }
  })
  return ids
})

// All tokens from all pages (for pattern generation)
// Includes position data: page, x, y, width, height
const allTokens = computed(() => {
  if (!pdfAnalysis.value?.pages) return []
  const tokens = []
  pdfAnalysis.value.pages.forEach((page, pageIndex) => {
    (page.words || []).forEach((word, wordIndex) => {
      tokens.push({
        ...word,
        id: `${pageIndex + 1}-${word.lineIndex}-${word.wordIndex}-${wordIndex}`,
        // Ensure page index is set (backend uses 0-indexed 'page' field)
        page: word.page ?? pageIndex,
        pageIndex,
        // Position data should already be present from backend
        x: word.x ?? 0,
        y: word.y ?? 0,
        width: word.width ?? 0,
        height: word.height ?? 0
      })
    })
  })
  return tokens
})

// Can proceed to next step?
const canProceed = computed(() => {
  if (step.value === 1) {
    return template.value.name && template.value.provider && uploadedFile.value && pdfAnalysis.value
  }
  if (step.value === 2) {
    return Object.keys(mappings).length > 0
  }
  return true
})

// File handling
function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (file) {
    processFile(file)
  }
}

function handleFileDrop(event) {
  isDraggingFile.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file && file.type === 'application/pdf') {
    processFile(file)
  } else {
    extractError.value = 'Per favore carica un file PDF'
  }
}

async function processFile(file) {
  if (file.type !== 'application/pdf') {
    extractError.value = 'Per favore carica un file PDF'
    return
  }

  extracting.value = true
  extractError.value = null

  try {
    // Use the new analyze endpoint for Textract-like view
    const { data } = await pdfAPI.analyzePDF(file)
    uploadedFile.value = file
    pdfAnalysis.value = data
    rawText.value = data.raw_text || ''

    // Extract timestamp from first page image URL for cleanup
    if (data.pages?.length > 0) {
      const match = data.pages[0].image_url.match(/template_page_(\d+)_/)
      if (match) {
        analysisTimestamp.value = match[1]
      }
    }

    if (!data.pages || data.pages.length === 0) {
      extractError.value = 'Impossibile analizzare il PDF. Verifica che sia un PDF valido.'
    }
  } catch (err) {
    extractError.value = err.response?.data?.error || 'Errore durante l\'analisi del PDF'
    console.error('PDF analysis error:', err)
  } finally {
    extracting.value = false
  }
}

// Drag and drop handlers
function handleDragStart(event, token) {
  draggingToken.value = token
}

function handleDragEnd() {
  draggingToken.value = null
  dragOverField.value = null
}

function handleFieldDrop(event, fieldKey) {
  dragOverField.value = null

  try {
    const tokenData = JSON.parse(event.dataTransfer.getData('application/json'))

    // Generate pattern for this field (includes position data)
    const patternInfo = generatePatternForField(
      tokenData,
      fieldKey,
      allTokens.value,
      rawText.value
    )

    // Get neighbor words for context editor
    const neighbors = getNeighborWordsForToken(tokenData, allTokens.value)

    // Pre-select context words that match the auto-detected anchor
    const autoSelected = { left: [], above: [], right: [] }
    if (patternInfo.anchorText) {
      const anchorLower = patternInfo.anchorText.toLowerCase()
      autoSelected.left = neighbors.left.filter(w =>
        anchorLower.includes(w.text.toLowerCase())
      )
    }

    // Store the mapping with all position, context, and anchor data
    mappings[fieldKey] = {
      token: tokenData,
      pattern: patternInfo.pattern,
      prefix: patternInfo.prefix,
      suffix: patternInfo.suffix,
      valuePattern: patternInfo.valuePattern,
      // Position data for position-based extraction
      page: patternInfo.page,
      x: patternInfo.x,
      y: patternInfo.y,
      width: patternInfo.width,
      height: patternInfo.height,
      // Context for validation
      contextLeft: patternInfo.contextLeft,
      contextAbove: patternInfo.contextAbove,
      // Anchor-based extraction
      anchorText: patternInfo.anchorText,
      anchorDirection: patternInfo.anchorDirection,
      globalSearch: patternInfo.globalSearch,
      // Context editor data
      selectedContext: autoSelected,
      allNeighbors: neighbors
    }

    // Reset test results when mapping changes
    hasTestedPatterns.value = false
    testResults.value = {}
  } catch (e) {
    console.error('Error handling drop:', e)
  }
}

function clearMapping(fieldKey) {
  if (contextEditorOpen.value === fieldKey) {
    contextEditorOpen.value = null
  }
  delete mappings[fieldKey]
  hasTestedPatterns.value = false
  testResults.value = {}
}

// Test all patterns
async function testAllPatterns() {
  testing.value = true
  testResults.value = {}
  hasTestedPatterns.value = true

  try {
    for (const [fieldKey, mapping] of Object.entries(mappings)) {
      if (mapping && mapping.pattern) {
        // Test with regex (for compatibility check)
        const regexResult = testPattern(mapping.pattern, rawText.value)

        // Build result with position info
        const result = {
          ...regexResult,
          // Position-based extraction info
          hasPosition: mapping.x > 0 || mapping.y > 0,
          page: mapping.page ?? 0,
          x: mapping.x ?? 0,
          y: mapping.y ?? 0,
          contextLeft: mapping.contextLeft || '',
          contextAbove: mapping.contextAbove || ''
        }

        // If regex found a value but it differs from the dragged token,
        // and we have position data, the position-based match should be more accurate
        if (result.success && mapping.token && result.value !== mapping.token.text) {
          if (result.hasPosition) {
            // Position-based extraction should use the exact value at that position
            result.value = mapping.token.text
            result.note = 'Valore estratto per posizione'
          }
        }

        testResults.value[fieldKey] = result
      }
    }
  } finally {
    testing.value = false
  }
}

// Context editor functions
function toggleContextEditor(fieldKey) {
  contextEditorOpen.value = contextEditorOpen.value === fieldKey ? null : fieldKey
}

function getNeighborWords(fieldKey, direction) {
  const mapping = mappings[fieldKey]
  if (!mapping?.allNeighbors) return []
  return mapping.allNeighbors[direction] || []
}

function isContextWordSelected(fieldKey, wordId) {
  const mapping = mappings[fieldKey]
  if (!mapping?.selectedContext) return false
  const { left, above, right } = mapping.selectedContext
  return [...(left || []), ...(above || []), ...(right || [])].some(w => w.id === wordId)
}

function toggleContextWord(fieldKey, word, direction) {
  const mapping = mappings[fieldKey]
  if (!mapping?.selectedContext) return

  const dirArr = mapping.selectedContext[direction] || []
  const idx = dirArr.findIndex(w => w.id === word.id)

  if (idx >= 0) {
    // Remove
    dirArr.splice(idx, 1)
  } else {
    // Add, maintaining spatial order (by x for left/right, by y for above)
    dirArr.push(word)
    if (direction === 'left' || direction === 'right') {
      dirArr.sort((a, b) => a.x - b.x)
    } else {
      dirArr.sort((a, b) => a.y - b.y)
    }
  }

  // Regenerate the pattern with the updated context
  const patternInfo = regeneratePatternWithContext(
    mapping.token,
    fieldKey,
    allTokens.value,
    rawText.value,
    mapping.selectedContext
  )

  // Update the mapping in-place
  mapping.pattern = patternInfo.pattern
  mapping.prefix = patternInfo.prefix
  mapping.suffix = patternInfo.suffix
  mapping.anchorText = patternInfo.anchorText
  mapping.anchorDirection = patternInfo.anchorDirection

  // Reset test results
  hasTestedPatterns.value = false
  testResults.value = {}
}

function getContextSummary(fieldKey) {
  const mapping = mappings[fieldKey]
  if (!mapping?.selectedContext) return ''

  const left = mapping.selectedContext.left || []
  const above = mapping.selectedContext.above || []

  if (left.length > 0) {
    return '\u2190 ' + left.map(w => w.text).join(' ')
  }
  if (above.length > 0) {
    return '\u2191 ' + above.map(w => w.text).join(' ')
  }
  return ''
}

// Navigation
function nextStep() {
  if (canProceed.value && step.value < 3) {
    step.value++
  }
}

// Helper functions
function getFieldLabel(fieldKey) {
  const field = extractionFields.value.find(f => f.key === fieldKey)
  return field ? field.label : fieldKey
}

function getDirectionLabel(direction) {
  const labels = { right: 'destra', below: 'sotto', right_or_below: 'destra/sotto' }
  return labels[direction] || direction
}

function getUtilityTypeName(type) {
  const names = {
    electricity: 'Luce',
    gas: 'Gas',
    water: 'Acqua',
    waste: 'Rifiuti'
  }
  return names[type] || type
}

// Cleanup temporary images
async function cleanupImages() {
  if (analysisTimestamp.value) {
    try {
      await pdfAPI.cleanupImages(analysisTimestamp.value)
    } catch (e) {
      console.error('Cleanup error:', e)
    }
  }
}

// Handle close with cleanup
function handleClose() {
  cleanupImages()
  emit('close')
}

// Save template
async function saveTemplate() {
  saving.value = true
  error.value = null

  try {
    // Build extraction rules array from mappings (includes position data)
    const extractionRules = []
    for (const [fieldKey, mapping] of Object.entries(mappings)) {
      if (mapping && mapping.pattern) {
        extractionRules.push({
          field: fieldKey,
          pattern: mapping.pattern,
          prefix: mapping.prefix || '',
          suffix: mapping.suffix || '',
          value_pattern: mapping.valuePattern || '',
          format: '',
          // Position data for position-based extraction
          page: mapping.page ?? 0,
          x: mapping.x ?? 0,
          y: mapping.y ?? 0,
          width: mapping.width ?? 0,
          height: mapping.height ?? 0,
          // Context for validation
          context_left: mapping.contextLeft || '',
          context_above: mapping.contextAbove || '',
          // Anchor-based extraction
          anchor_text: mapping.anchorText || '',
          anchor_direction: mapping.anchorDirection || 'right_or_below',
          global_search: mapping.globalSearch || false
        })
      }
    }

    const templateData = {
      name: template.value.name,
      provider: template.value.provider,
      utility_type: template.value.utility_type,
      is_default: template.value.is_default,
      extraction_rules: extractionRules
    }

    if (isEditing.value) {
      await templatesAPI.updateBillTemplate(props.existingTemplate.id, templateData)
    } else {
      await templatesAPI.createBillTemplate(templateData)
    }

    // Cleanup images after successful save
    await cleanupImages()

    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    saving.value = false
  }
}

// Initialize with existing template data if editing
onMounted(() => {
  if (props.existingTemplate) {
    template.value = {
      name: props.existingTemplate.name,
      provider: props.existingTemplate.provider,
      utility_type: props.existingTemplate.utility_type,
      is_default: props.existingTemplate.is_default
    }

    // Parse existing rules and convert to mappings
    if (props.existingTemplate.extraction_rules) {
      try {
        const existingRules = JSON.parse(props.existingTemplate.extraction_rules)
        for (const rule of existingRules) {
          if (rule.field) {
            mappings[rule.field] = {
              token: {
                id: `existing-${rule.field}`,
                text: rule.value_pattern || rule.pattern || '...',
                type: 'text'
              },
              pattern: rule.pattern || '',
              prefix: rule.prefix || '',
              suffix: rule.suffix || '',
              valuePattern: rule.value_pattern || ''
            }
          }
        }
      } catch (e) {
        console.error('Error parsing existing rules:', e)
      }
    }
  }
})

// Cleanup on unmount
onUnmounted(() => {
  cleanupImages()
})
</script>
