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
                  ? 'bg-green-500 text-white'
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
              step > index + 1 ? 'bg-green-500' : 'bg-surface-3'
            ]"
          />
        </div>
      </div>

      <!-- Step 1: Upload PDF & Basic Info -->
      <WizardStepInfo
        v-if="step === 1"
        :template="template"
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
        :testing="testing"
        :has-tested-patterns="hasTestedPatterns"
        :test-results="testResults"
        :context-editor-open="contextEditorOpen"
        :get-direction-label="getDirectionLabel"
        :get-context-summary="getContextSummary"
        :get-neighbor-words="getNeighborWords"
        :is-context-word-selected="isContextWordSelected"
        :get-field-label="getFieldLabel"
        :direction-labels="directionLabels"
        @drag-start="handleDragStart"
        @drag-end="handleDragEnd"
        @field-drop="handleFieldDrop($event.event, $event.fieldKey)"
        @clear-mapping="clearMapping"
        @toggle-context-editor="toggleContextEditor"
        @toggle-context-word="toggleContextWord($event.fieldKey, $event.word, $event.direction)"
        @test-patterns="testAllPatterns"
        @update:drag-over-field="dragOverField = $event"
      />

      <!-- Step 3: Review & Save -->
      <WizardStepReview
        v-if="step === 3"
        :template="template"
        :mappings="mappings"
        :extraction-fields="extractionFields"
        :get-field-label="getFieldLabel"
        @update:template="template = $event"
      />

      <!-- Error -->
      <div v-if="error" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
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
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { templatesAPI, pdfAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import { generatePatternForField, regeneratePatternWithContext, getNeighborWordsForToken, testPattern } from '@/utils/patternGenerator'
import { mergeExtendedDateTokens } from '@/utils/tokenizer'
import WizardStepInfo from './WizardStepInfo.vue'
import WizardStepMapping from './WizardStepMapping.vue'
import WizardStepReview from './WizardStepReview.vue'

const props = defineProps({
  existingTemplate: {
    type: Object,
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
const baseExtractionFields = computed(() => [
  { key: 'amount_total', label: t('utilities.templateWizard.fields.amount_total'), required: true },
  { key: 'bill_number', label: t('utilities.templateWizard.fields.bill_number'), required: false },
  { key: 'period_start', label: t('utilities.templateWizard.fields.period_start'), required: false },
  { key: 'period_end', label: t('utilities.templateWizard.fields.period_end'), required: false },
  { key: 'due_date', label: t('utilities.templateWizard.fields.due_date'), required: false },
  { key: 'issue_date', label: t('utilities.templateWizard.fields.issue_date'), required: false },
  { key: 'communication_text', label: t('utilities.templateWizard.fields.communication_text'), required: false, multiLine: true }
])

const gasExtraFields = computed(() => [
  { key: 'provider_reading', label: t('utilities.templateWizard.fields.provider_reading'), required: false },
  { key: 'conversion_coefficient', label: t('utilities.templateWizard.fields.conversion_coefficient'), required: false },
  { key: 'estimated_date', label: t('utilities.templateWizard.fields.estimated_date'), required: false },
  { key: 'estimated_reading', label: t('utilities.templateWizard.fields.estimated_reading'), required: false },
  { key: 'previous_estimated_consumption', label: t('utilities.templateWizard.fields.previous_estimated_consumption_smc'), required: false }
])

const waterExtraFields = computed(() => [
  { key: 'provider_reading', label: t('utilities.templateWizard.fields.provider_reading'), required: false },
  { key: 'estimated_date', label: t('utilities.templateWizard.fields.estimated_date'), required: false },
  { key: 'estimated_reading', label: t('utilities.templateWizard.fields.estimated_reading'), required: false },
  { key: 'previous_estimated_consumption', label: t('utilities.templateWizard.fields.previous_estimated_consumption_mc'), required: false }
])

// Metered services, per Utility.IsMetered — waste (TARI) is fixed-cost, so its
// period_end comes from the service's billing frequency like the other fixed
// services (see inferMissingPeriodDate in PDFUploadZone).
const isMeteredTemplateType = computed(() => {
  return ['electricity', 'gas', 'water'].includes(template.value.utility_type)
})

const extractionFields = computed(() => {
  let fields = baseExtractionFields.value
  // Fixed-cost services: period_end is derived from billing frequency, not extracted
  if (!isMeteredTemplateType.value) {
    fields = fields.filter(f => f.key !== 'period_end')
  }
  if (template.value.utility_type === 'gas') {
    return [...fields, ...gasExtraFields.value]
  }
  if (template.value.utility_type === 'water') {
    return [...fields, ...waterExtraFields.value]
  }
  return fields
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
    const { data } = await pdfAPI.analyzePDF(file)
    uploadedFile.value = file
    // Merge split day/month/year tokens so an extended date ("01 aprile 2026")
    // is one draggable box; both the overlay and allTokens read these pages.
    if (Array.isArray(data.pages)) {
      data.pages = data.pages.map(p => ({ ...p, words: mergeExtendedDateTokens(p.words || []) }))
    }
    pdfAnalysis.value = data
    rawText.value = data.raw_text || ''

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
    extractError.value = err.response?.data?.error || t('utilities.templateWizard.analyzeError')
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
    const fieldDef = extractionFields.value.find(f => f.key === fieldKey)

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
      anchorDirection: fieldDef?.multiLine ? 'below' : (patternInfo.anchorDirection || 'right_or_below'),
      globalSearch: patternInfo.globalSearch,
      // Multi-line flag for text block fields (communication_text)
      multiLine: fieldDef?.multiLine || false,
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
          global_search: mapping.globalSearch || false,
          multi_line: mapping.multiLine || false
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
    error.value = err.response?.data?.error || err.message || t('utilities.templateWizard.saveError')
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
              valuePattern: rule.value_pattern || '',
              multiLine: rule.multi_line || false
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
