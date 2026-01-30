<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[70] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-2xl p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ isEditing ? 'Modifica Template' : 'Crea Template Estrazione' }}
        </h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

<div class="flex items-center justify-center pb-10 pt-4"> <div
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
        {{ stepInfo || 'Step ' + (index + 1) }} 
      </span>
    </div>

    <div
      v-if="index < steps.length - 1"
      :class="[
        'w-12 h-1 mx-2 mb-0', /* keep the label center aligned with the circles */
        step > index + 1 ? 'bg-green-500' : 'bg-gray-200 dark:bg-gray-700'
      ]"
    />
  </div>
</div>

      <!-- Step 1: Upload PDF -->
      <div v-if="step === 1" class="space-y-4">
        <p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
          Carica un PDF di esempio del fornitore per creare le regole di estrazione.
        </p>

        <!-- Basic Info -->
        <Input
          v-model="template.name"
          label="Nome Template *"
          placeholder="Es: E.ON Luce Bimestrale"
          required
        />

        <Input
          v-model="template.provider"
          label="Fornitore *"
          placeholder="Es: E.ON, Enel, ETRA..."
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
            isDragging
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500',
            extracting ? 'opacity-50 pointer-events-none' : ''
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

          <div v-if="extracting" class="flex flex-col items-center gap-2">
            <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="text-sm text-gray-600 dark:text-gray-400">Estrazione testo...</span>
          </div>

          <div v-else-if="uploadedFile" class="flex flex-col items-center gap-2">
            <svg class="w-10 h-10 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ uploadedFile.name }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">Clicca per cambiare file</p>
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

      <!-- Step 2: Preview Text -->
      <div v-if="step === 2" class="space-y-4">
        <p class="text-gray-600 dark:text-gray-400 text-sm">
          Verifica il testo estratto dal PDF. Identifica i valori chiave da estrarre.
        </p>

        <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 max-h-96 overflow-y-auto">
          <pre class="text-xs text-gray-700 dark:text-gray-300 whitespace-pre-wrap font-mono">{{ rawText }}</pre>
        </div>

        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
          <p class="text-sm text-blue-800 dark:text-blue-300">
            <strong>Suggerimento:</strong> Cerca nel testo i valori che vuoi estrarre:
            importo totale, date, consumi, numero bolletta. Nel prossimo step definirai
            le regole per estrarli automaticamente.
          </p>
        </div>
      </div>

      <!-- Step 3: Define Rules -->
      <div v-if="step === 3" class="space-y-6">
        <p class="text-gray-600 dark:text-gray-400 text-sm">
          Definisci le regole di estrazione per ciascun campo. Usa pattern regex o testo di riferimento.
        </p>

        <!-- Extraction Rules -->
        <div class="space-y-4">
          <div v-for="field in extractionFields" :key="field.key" class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
              <label class="font-medium text-gray-900 dark:text-white">{{ field.label }}</label>
              <span v-if="field.required" class="text-xs text-red-500">*</span>
            </div>

            <div class="space-y-3">
              <Input
                v-model="rules[field.key].pattern"
                label="Pattern Regex"
                :placeholder="field.placeholder"
                class="text-sm"
              />

              <Input
                v-model="rules[field.key].prefix"
                label="Testo Prima (opzionale)"
                placeholder="Es: Totale da pagare"
                class="text-sm"
              />

              <Input
                v-model="rules[field.key].suffix"
                label="Testo Dopo (opzionale)"
                placeholder="Es: EUR"
                class="text-sm"
              />

              <!-- Test result -->
              <div v-if="testResults[field.key]" class="flex items-center gap-2 text-sm">
                <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <span class="text-gray-600 dark:text-gray-400">Valore trovato:</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ testResults[field.key] }}</span>
              </div>
              <div v-else-if="rules[field.key].pattern && hasTestedRules" class="flex items-center gap-2 text-sm text-orange-600 dark:text-orange-400">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span>Nessuna corrispondenza trovata</span>
              </div>
            </div>
          </div>
        </div>

        <Button @click="testRules" :disabled="testing" variant="secondary" class="w-full">
          {{ testing ? 'Test in corso...' : 'Testa Regole' }}
        </Button>
      </div>

      <!-- Step 4: Save -->
      <div v-if="step === 4" class="space-y-4">
        <div class="text-center py-4">
          <svg class="w-16 h-16 mx-auto text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <h4 class="text-lg font-bold text-gray-900 dark:text-white mt-4">Template Pronto</h4>
          <p class="text-gray-600 dark:text-gray-400 mt-2">
            Il template per <strong>{{ template.provider }}</strong> e pronto per essere salvato.
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
            <span class="text-gray-600 dark:text-gray-400">Regole definite:</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ definedRulesCount }}</span>
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
          @click="$emit('close')"
          class="flex-1"
        >
          Annulla
        </Button>
        <Button
          v-if="step < 4"
          @click="nextStep"
          :disabled="!canProceed"
          class="flex-1"
        >
          Avanti
        </Button>
        <Button
          v-if="step === 4"
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
import { ref, computed, onMounted } from 'vue'
import { templatesAPI, pdfAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  existingTemplate: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])

const step = ref(1)
const steps = ['Info & PDF', 'Anteprima', 'Regole', 'Salva']

const fileInput = ref(null)
const isDragging = ref(false)
const extracting = ref(false)
const extractError = ref(null)
const uploadedFile = ref(null)
const rawText = ref('')
const testing = ref(false)
const saving = ref(false)
const error = ref(null)
const hasTestedRules = ref(false)
const testResults = ref({})

const isEditing = computed(() => !!props.existingTemplate)

const template = ref({
  name: '',
  provider: '',
  utility_type: 'electricity',
  is_default: false
})

const extractionFields = [
  { key: 'amount_total', label: 'Importo Totale', placeholder: '(\\d+[.,]\\d{2})\\s*€?', required: true },
  { key: 'consumption_total', label: 'Consumo Totale', placeholder: '(\\d+[.,]?\\d*)\\s*(?:kWh|Smc|mc)', required: true },
  { key: 'period_start', label: 'Data Inizio Periodo', placeholder: '(\\d{2}[/.-]\\d{2}[/.-]\\d{4})', required: false },
  { key: 'period_end', label: 'Data Fine Periodo', placeholder: '(\\d{2}[/.-]\\d{2}[/.-]\\d{4})', required: false },
  { key: 'due_date', label: 'Data Scadenza', placeholder: '(\\d{2}[/.-]\\d{2}[/.-]\\d{4})', required: false },
  { key: 'issue_date', label: 'Data Emissione', placeholder: '(\\d{2}[/.-]\\d{2}[/.-]\\d{4})', required: false },
  { key: 'bill_number', label: 'Numero Bolletta', placeholder: '(\\d+)', required: false }
]

const rules = ref({
  amount_total: { pattern: '', prefix: '', suffix: '' },
  consumption_total: { pattern: '', prefix: '', suffix: '' },
  period_start: { pattern: '', prefix: '', suffix: '' },
  period_end: { pattern: '', prefix: '', suffix: '' },
  due_date: { pattern: '', prefix: '', suffix: '' },
  issue_date: { pattern: '', prefix: '', suffix: '' },
  bill_number: { pattern: '', prefix: '', suffix: '' }
})

const canProceed = computed(() => {
  if (step.value === 1) {
    return template.value.name && template.value.provider && uploadedFile.value && rawText.value
  }
  if (step.value === 2) {
    return rawText.value.length > 0
  }
  if (step.value === 3) {
    return rules.value.amount_total.pattern || rules.value.consumption_total.pattern
  }
  return true
})

const definedRulesCount = computed(() => {
  return Object.values(rules.value).filter(r => r.pattern).length
})

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (file) {
    processFile(file)
  }
}

function handleDrop(event) {
  isDragging.value = false
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
    const { data } = await pdfAPI.extractText(file)
    uploadedFile.value = file
    rawText.value = data.raw_text || ''

    if (!rawText.value) {
      extractError.value = 'Nessun testo estratto dal PDF. Potrebbe essere un PDF scansionato.'
    }
  } catch (err) {
    extractError.value = err.response?.data?.error || 'Errore durante l\'estrazione del testo'
    console.error('PDF extraction error:', err)
  } finally {
    extracting.value = false
  }
}

function nextStep() {
  if (canProceed.value && step.value < 4) {
    step.value++
  }
}

async function testRules() {
  testing.value = true
  testResults.value = {}
  hasTestedRules.value = true

  try {
    for (const field of extractionFields) {
      const rule = rules.value[field.key]
      if (rule.pattern) {
        const result = applyRule(rule, rawText.value)
        if (result) {
          testResults.value[field.key] = result
        }
      }
    }
  } finally {
    testing.value = false
  }
}

function applyRule(rule, text) {
  try {
    let searchText = text

    // If prefix is specified, find text after it
    if (rule.prefix) {
      const prefixIndex = searchText.indexOf(rule.prefix)
      if (prefixIndex >= 0) {
        searchText = searchText.substring(prefixIndex + rule.prefix.length)
      }
    }

    // If suffix is specified, find text before it
    if (rule.suffix) {
      const suffixIndex = searchText.indexOf(rule.suffix)
      if (suffixIndex >= 0) {
        searchText = searchText.substring(0, suffixIndex)
      }
    }

    // Apply regex pattern
    const regex = new RegExp(rule.pattern, 'i')
    const match = searchText.match(regex)

    if (match) {
      return match[1] || match[0]
    }
  } catch (e) {
    console.error('Regex error:', e)
  }

  return null
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

async function saveTemplate() {
  saving.value = true
  error.value = null

  try {
    // Build extraction rules array
    const extractionRules = []
    for (const field of extractionFields) {
      const rule = rules.value[field.key]
      if (rule.pattern) {
        extractionRules.push({
          field: field.key,
          pattern: rule.pattern,
          format: '' // Optional date format
        })
      }
    }

    const templateData = {
      name: template.value.name,
      provider: template.value.provider,
      utility_type: template.value.utility_type,
      is_default: template.value.is_default,
      extraction_rules: extractionRules // Send as array, backend will JSON encode
    }

    if (isEditing.value) {
      await templatesAPI.updateBillTemplate(props.existingTemplate.id, templateData)
    } else {
      await templatesAPI.createBillTemplate(templateData)
    }

    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (props.existingTemplate) {
    template.value = {
      name: props.existingTemplate.name,
      provider: props.existingTemplate.provider,
      utility_type: props.existingTemplate.utility_type,
      is_default: props.existingTemplate.is_default
    }

    // Parse existing rules
    if (props.existingTemplate.extraction_rules) {
      try {
        const existingRules = JSON.parse(props.existingTemplate.extraction_rules)
        for (const rule of existingRules) {
          if (rules.value[rule.field]) {
            rules.value[rule.field] = {
              pattern: rule.pattern || '',
              prefix: rule.prefix || '',
              suffix: rule.suffix || ''
            }
          }
        }
      } catch (e) {
        console.error('Error parsing existing rules:', e)
      }
    }
  }
})
</script>
