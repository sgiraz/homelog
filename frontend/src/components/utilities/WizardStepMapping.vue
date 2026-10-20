<template>
  <div class="space-y-4">
    <p class="text-ink-soft text-sm">
      {{ t('utilities.wizardStepMapping.intro') }}
    </p>

    <div class="flex gap-4" style="height: 500px;">
      <!-- Left Panel: PDF Textract View -->
      <div class="flex-1 flex flex-col border border-line rounded-lg overflow-hidden">
        <PDFTextractView
          :pages="pdfAnalysis?.pages || []"
          :loading="extracting"
          :mapped-word-ids="mappedTokenIds"
          @drag-start="emit('drag-start', $event)"
          @drag-end="emit('drag-end')"
        />
      </div>

      <!-- Right Panel: Drop Zones for Fields -->
      <div class="w-72 flex flex-col">
        <h4 class="font-medium text-ink text-sm mb-2">{{ t('utilities.wizardStepMapping.fieldsTitle') }}</h4>
        <div class="flex-1 space-y-2 overflow-y-auto pr-1">
          <div
            v-for="field in extractionFields"
            :key="field.key"
            :class="[
              'border rounded-lg p-3 transition-all',
              dragOverField === field.key
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : fieldErrors[field.key]
                  ? 'border-danger/40 bg-danger/10'
                  : mappings[field.key]
                    ? 'border-positive/30 bg-positive/10'
                    : 'border-line'
            ]"
            @dragover.prevent="emit('update:dragOverField', field.key)"
            @dragleave.prevent="emit('update:dragOverField', null)"
            @drop.prevent="emit('field-drop', { event: $event, fieldKey: field.key })"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-ink">
                {{ field.label }}
                <span v-if="field.required" class="text-danger-soft">*</span>
              </span>
              <button
                v-if="mappings[field.key]"
                @click="emit('clear-mapping', field.key)"
                class="text-ink-faint hover:text-danger-soft transition-colors"
                :title="t('utilities.wizardStepMapping.removeMapping')"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div v-if="suggesting === field.key" class="text-xs text-ink-muted py-2">
              {{ t('utilities.wizardStepMapping.suggesting') }}
            </div>

            <div v-else-if="fieldErrors[field.key]" class="text-xs text-danger-soft py-1">
              {{ t('utilities.wizardStepMapping.noRule', { value: fieldErrors[field.key] }) }}
            </div>

            <!-- Drop zone content -->
            <div v-else-if="mappings[field.key]" class="space-y-1">
              <Badge v-if="mappings[field.key].notFound" variant="warning" class="text-sm">
                {{ t('utilities.wizardStepMapping.notFoundInPdf') }}
              </Badge>
              <Badge v-else variant="positive" class="text-sm">
                {{ mappingValue(field.key) }}
              </Badge>
              <div class="text-[11px] text-ink-muted">
                {{ describeRule(mappingRule(field.key)) }}
              </div>
              <button
                v-if="(mappings[field.key].suggestions?.length || 0) > 1"
                @click="emit('next-suggestion', field.key)"
                class="text-xs text-accent-soft hover:underline"
              >
                {{ t('utilities.wizardStepMapping.otherRule', { n: mappings[field.key].choice + 1, total: mappings[field.key].suggestions.length }) }}
              </button>

              <!-- Words that mark an estimated reading -->
              <div v-if="field.key === 'reading_type' && mappings[field.key].cell" class="pt-1 space-y-1">
                <p class="text-[11px] text-ink-soft">{{ t('utilities.wizardStepMapping.estimatedHint') }}</p>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="word in cellWords(mappings[field.key].cell)"
                    :key="word"
                    @click="emit('toggle-estimated-word', { fieldKey: field.key, word })"
                    :class="[
                      'px-1.5 py-0.5 rounded border text-[11px] transition-all',
                      mappings[field.key].estimatedWords.includes(word)
                        ? 'bg-warning/15 border-warning/50 text-warning-soft'
                        : 'bg-surface border-line text-ink-muted hover:border-ink-faint'
                    ]"
                  >
                    {{ word }}
                  </button>
                </div>
                <p class="text-[11px]" :class="mappings[field.key].estimatedWords.length ? 'text-warning-soft' : 'text-ink-muted'">
                  {{ mappings[field.key].estimatedWords.length ? t('utilities.wizardStepMapping.readsAsEstimated') : t('utilities.wizardStepMapping.readsAsActual') }}
                </p>
              </div>
            </div>
            <div
              v-else
              class="text-ink-faint text-sm py-2 text-center border-2 border-dashed border-line rounded"
            >
              {{ field.multiLine ? t('utilities.wizardStepMapping.dropMultiLine') : t('utilities.wizardStepMapping.dropHere') }}
            </div>
            <div v-if="field.multiLine && mappings[field.key]" class="text-[10px] text-warning-soft mt-1">
              {{ t('utilities.wizardStepMapping.multiLineNotice') }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import Badge from '@/components/common/Badge.vue'
import PDFTextractView from './PDFTextractView.vue'

defineOptions({ name: 'WizardStepMapping' })

const props = defineProps({
  pdfAnalysis: { type: Object, default: null },
  extracting: { type: Boolean, default: false },
  mappedTokenIds: { type: [Set, Array], required: true },
  extractionFields: { type: Array, required: true },
  dragOverField: { type: String, default: null },
  mappings: { type: Object, required: true },
  fieldErrors: { type: Object, required: true },
  suggesting: { type: String, default: null },
  describeRule: { type: Function, required: true }
})

const emit = defineEmits([
  'drag-start',
  'drag-end',
  'field-drop',
  'clear-mapping',
  'next-suggestion',
  'toggle-estimated-word',
  'update:dragOverField'
])

const { t } = useI18n()

function mappingRule(fieldKey) {
  const m = props.mappings[fieldKey]
  return m.suggestions ? m.suggestions[m.choice].rule : m.rule
}

function mappingValue(fieldKey) {
  const m = props.mappings[fieldKey]
  return m.suggestions ? m.suggestions[m.choice].value : m.token.text
}

function cellWords(cell) {
  return [...new Set(cell.split(/\s+/).filter(w => /\p{L}{2,}/u.test(w)))]
}
</script>
