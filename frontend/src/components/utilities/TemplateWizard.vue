<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[70] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-6xl p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-ink">
          {{ isEditing ? t('utilities.templateWizard.editTitle') : t('utilities.templateWizard.createTitle') }}
        </h3>
        <button @click="handleClose" class="text-ink-muted hover:text-ink">
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
                  ? 'bg-positive text-white'
                  : step === index + 1
                    ? 'bg-blue-500 text-white'
                    : 'bg-surface-3 text-ink-soft'
              ]"
            >
              <svg v-if="step > index + 1" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <span v-else>{{ index + 1 }}</span>
            </div>

            <span
              class="absolute -bottom-7 text-[10px] uppercase tracking-wider font-bold whitespace-nowrap"
              :class="step === index + 1 ? 'text-blue-500' : 'text-ink-faint'"
            >
              {{ stepInfo }}
            </span>
          </div>

          <div
            v-if="index < steps.length - 1"
            :class="[
              'w-16 h-1 mx-2 mb-0',
              step > index + 1 ? 'bg-positive' : 'bg-surface-3'
            ]"
          />
        </div>
      </div>

      <!-- Step 1: Upload PDF & Basic Info -->
      <WizardStepInfo
        v-if="step === 1"
        :template="template"
        :utility-types="templateTypes"
        :pdf-file="uploadedFile"
        :pdf-analysis="pdfAnalysis"
        :extracting="extracting"
        :extract-error="extractError"
        @update:template="template = $event"
        @file-selected="processFile"
      />

      <!-- Step 2: Drag & Drop Field Mapping with PDF Textract View -->
      <WizardStepMapping
        v-if="step === 2"
        :pdf-analysis="pdfAnalysis"
        :extracting="extracting"
        :mapped-token-ids="mappedTokenIds"
        :extraction-fields="extractionFields"
        :drag-over-field="dragOverField"
        :mappings="mappings"
        :field-errors="fieldErrors"
        :suggesting="suggesting"
        :describe-rule="describeRule"
        @drag-start="handleDragStart"
        @drag-end="handleDragEnd"
        @field-drop="handleFieldDrop($event.event, $event.fieldKey)"
        @clear-mapping="clearMapping"
        @next-suggestion="nextSuggestion"
        @toggle-estimated-word="toggleEstimatedWord($event.fieldKey, $event.word)"
        @update:drag-over-field="dragOverField = $event"
      />

      <!-- Step 3: Review & Save -->
      <WizardStepReview
        v-if="step === 3"
        :template="template"
        :mappings="mappings"
        :extraction-fields="extractionFields"
        :get-field-label="getFieldLabel"
        :trying="trying"
        :try-result="tryResult"
        :try-error="tryError"
        @update:template="template = $event"
        @try-file="tryOnAnotherBill"
      />

      <!-- Error -->
      <div v-if="error" class="mt-4 text-danger-soft text-sm bg-danger/10 p-3 rounded-lg">
        {{ error }}
      </div>

      <!-- Navigation Buttons -->
      <div class="flex gap-3 mt-6 pt-4 border-t border-line">
        <Button
          v-if="step > 1"
          type="button"
          variant="secondary"
          @click="step--"
          class="flex-1"
        >
          {{ t('utilities.templateWizard.back') }}
        </Button>
        <Button
          v-if="step === 1"
          type="button"
          variant="secondary"
          @click="handleClose"
          class="flex-1"
        >
          {{ t('utilities.templateWizard.cancel') }}
        </Button>
        <Button
          v-if="step < 3"
          @click="nextStep"
          :disabled="!canProceed"
          class="flex-1"
        >
          {{ t('utilities.templateWizard.next') }}
        </Button>
        <Button
          v-if="step === 3"
          @click="saveTemplate"
          :disabled="saving"
          class="flex-1"
        >
          {{ saving ? t('utilities.templateWizard.saving') : t('utilities.templateWizard.save') }}
        </Button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { templatesAPI, pdfAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import { mergeExtendedDateTokens } from '@/utils/tokenizer'
import WizardStepInfo from './WizardStepInfo.vue'
import WizardStepMapping from './WizardStepMapping.vue'
import WizardStepReview from './WizardStepReview.vue'
import { apiErrorMessage } from '@/utils/apiError'

const props = defineProps({
  existingTemplate: {
    type: Object,
    default: null
  },
  propertyId: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])

const { t } = useI18n()

// Step management
const step = ref(1)
const steps = computed(() => [
  t('utilities.templateWizard.stepInfoLabel'),
  t('utilities.templateWizard.stepMappingLabel'),
  t('utilities.templateWizard.stepReviewLabel')
])

// File upload state
const extracting = ref(false)
const extractError = ref(null)
const uploadedFile = ref(null)

// PDF Analysis data (from /pdf/analyze endpoint)
const pdfAnalysis = ref(null)
const analysisTimestamp = ref(null)

// Drag and drop state
const dragOverField = ref(null)
const draggingToken = ref(null)

// Field mappings. A dropped value carries the rules the server suggested for
// it, all checked to read it back ({ token, suggestions, choice, cell,
// estimatedWords }); a rule of the template being edited is kept as { rule }.
const mappings = reactive({})
// Fields whose dropped value no rule reads back: the value, shown in red.
const fieldErrors = reactive({})
const suggesting = ref(null)

// Second-bill check on the review step
const trying = ref(false)
const tryResult = ref(null)
const tryError = ref(null)

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

// Fields a template may map, per service type: the server's catalogue
const fieldCatalogue = ref([])
const templateTypes = computed(() => fieldCatalogue.value.map(set => set.type))

const extractionFields = computed(() => {
  const set = fieldCatalogue.value.find(s => s.type === template.value.utility_type)
  return (set?.fields || []).map(f => ({
    key: f.key,
    label: t(`utilities.templateWizard.fields.${f.key}`),
    required: !!f.required,
    multiLine: !!f.multi_line
  }))
})

// A mapping the type does not list would be rejected on save (an old rule,
// or one left over from another type).
watch(extractionFields, fields => {
  if (!fields.length) return
  for (const key of Object.keys(mappings)) {
    if (!fields.some(f => f.key === key)) delete mappings[key]
  }
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

// File handling — the hidden input + drop zone live in WizardStepInfo,
// which emits `file-selected` with the raw File instance.
async function processFile(file) {
  if (file.type !== 'application/pdf') {
    extractError.value = t('utilities.templateWizard.uploadOnlyPdf')
    return
  }

  extracting.value = true
  extractError.value = null

  try {
    // Use the new analyze endpoint for Textract-like view
    const { data } = await pdfAPI.analyzePDF(file, props.existingTemplate?.id)
    uploadedFile.value = file
    // Merge split day/month/year tokens so an extended date ("01 aprile 2026")
    // is one draggable box; both the overlay and allTokens read these pages.
    if (Array.isArray(data.pages)) {
      data.pages = data.pages.map(p => ({ ...p, words: mergeExtendedDateTokens(p.words || []) }))
    }
    pdfAnalysis.value = data
    showExistingRuleValues(data.template_values || {})

    if (data.tag) {
      analysisTimestamp.value = data.tag
    } else if (data.pages?.length > 0) {
      const match = data.pages[0].image_url.match(/template_page_(\d+(?:_[a-f0-9]+)?)_/)
      if (match) {
        analysisTimestamp.value = match[1]
      }
    }

    if (!data.pages || data.pages.length === 0) {
      extractError.value = t('utilities.templateWizard.invalidPdf')
    }
  } catch (err) {
    extractError.value = apiErrorMessage(err, t('utilities.templateWizard.analyzeError'))
    console.error('PDF analysis error:', err)
  } finally {
    extracting.value = false
  }
}

// When editing, each untouched rule shows what it extracts from the uploaded
// PDF, pointing at the matching token so the overlay highlights it.
function showExistingRuleValues(values) {
  for (const mapping of Object.values(mappings)) {
    if (!mapping?.rule) continue
    const { rule } = mapping
    const text = values[rule.field] || ''
    const distance = tk => Math.hypot(tk.x - (rule.x ?? 0), tk.y - (rule.y ?? 0))
    const located = text && allTokens.value
      .filter(tk => tk.text === text && tk.page === (rule.page ?? 0))
      .sort((a, b) => distance(a) - distance(b))[0]
    mapping.token = located || { id: `existing-${rule.field}`, text, type: 'text' }
    mapping.notFound = !text
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

// The server proposes the rules that read the dropped value back from this
// same bill; none means the value cannot be mapped from where it was taken.
async function handleFieldDrop(event, fieldKey) {
  dragOverField.value = null
  let token
  try {
    token = JSON.parse(event.dataTransfer.getData('application/json'))
  } catch {
    return
  }

  suggesting.value = fieldKey
  delete fieldErrors[fieldKey]
  try {
    const { data } = await pdfAPI.suggestRule({
      tag: analysisTimestamp.value,
      field: fieldKey,
      word: { text: token.text, page: token.page, x: token.x, y: token.y, width: token.width, height: token.height }
    })
    if (!data.suggestions?.length) {
      delete mappings[fieldKey]
      fieldErrors[fieldKey] = token.text
      return
    }
    mappings[fieldKey] = { token, suggestions: data.suggestions, choice: 0, cell: data.cell, estimatedWords: [] }
    tryResult.value = null
  } catch (err) {
    error.value = apiErrorMessage(err, t('utilities.templateWizard.suggestError'))
  } finally {
    suggesting.value = null
  }
}

function clearMapping(fieldKey) {
  delete mappings[fieldKey]
  delete fieldErrors[fieldKey]
  tryResult.value = null
}

function nextSuggestion(fieldKey) {
  const mapping = mappings[fieldKey]
  mapping.choice = (mapping.choice + 1) % mapping.suggestions.length
  tryResult.value = null
}

function toggleEstimatedWord(fieldKey, word) {
  const words = mappings[fieldKey].estimatedWords
  const i = words.indexOf(word)
  if (i >= 0) words.splice(i, 1)
  else words.push(word)
  tryResult.value = null
}

// describeRule says in words how a rule finds its value.
function describeRule(rule) {
  if (rule.table) {
    const { column, cell, row_label: rowLabel, stop, occurrence } = rule.table
    const parts = [t('utilities.wizardStepMapping.ruleTable', { column })]
    if (occurrence) parts.push(t('utilities.wizardStepMapping.ruleOccurrence', { n: occurrence }))
    parts.push(cell === 'row'
      ? t('utilities.wizardStepMapping.ruleCellRow', { label: rowLabel })
      : t(cell === 'first' ? 'utilities.wizardStepMapping.ruleCellFirst' : 'utilities.wizardStepMapping.ruleCellLast'))
    if (stop) parts.push(t('utilities.wizardStepMapping.ruleStop', { stop }))
    return parts.join(', ')
  }
  if (rule.anchor_text) {
    return t('utilities.wizardStepMapping.ruleAnchor', { anchor: rule.anchor_text, direction: getDirectionLabel(rule.anchor_direction) })
  }
  return t('utilities.wizardStepMapping.positionOnly')
}

// templateRules is what the template saves: the chosen suggestion per field,
// the reading type with the words that mark an estimate.
function templateRules() {
  return Object.entries(mappings).map(([fieldKey, mapping]) => {
    if (!mapping.suggestions) return mapping.rule
    const rule = { ...mapping.suggestions[mapping.choice].rule, field: fieldKey }
    if (fieldKey === 'reading_type') rule.estimated_words = [...mapping.estimatedWords]
    return rule
  })
}

async function tryOnAnotherBill(file) {
  trying.value = true
  tryError.value = null
  tryResult.value = null
  try {
    const { data } = await pdfAPI.tryTemplate(file, templateRules())
    tryResult.value = data
  } catch (err) {
    tryError.value = apiErrorMessage(err, t('utilities.templateWizard.analyzeError'))
  } finally {
    trying.value = false
  }
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
  const key = `utilities.templateWizard.directions.${direction}`
  const label = t(key)
  return label === key ? direction : label
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
    const extractionRules = templateRules()

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
      await templatesAPI.createBillTemplate({ ...templateData, property_id: props.propertyId })
    }

    // Cleanup images after successful save
    await cleanupImages()

    emit('saved')
  } catch (err) {
    error.value = apiErrorMessage(err, t('utilities.templateWizard.saveError'))
  } finally {
    saving.value = false
  }
}

// Initialize with existing template data if editing
onMounted(() => {
  templatesAPI.listBillTemplateFields()
    .then(({ data }) => { fieldCatalogue.value = data })
    .catch(err => { error.value = apiErrorMessage(err, t('utilities.templateWizard.fieldsLoadError')) })

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
              rule,
              token: { id: `existing-${rule.field}`, text: '', type: 'text' },
              notFound: false
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
