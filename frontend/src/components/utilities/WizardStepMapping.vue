<template>
  <div class="space-y-4">
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
          @drag-start="emit('drag-start', $event)"
          @drag-end="emit('drag-end')"
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
            @dragover.prevent="emit('update:dragOverField', field.key)"
            @dragleave.prevent="emit('update:dragOverField', null)"
            @drop.prevent="emit('field-drop', { event: $event, fieldKey: field.key })"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-gray-900 dark:text-white">
                {{ field.label }}
                <span v-if="field.required" class="text-red-500">*</span>
              </span>
              <button
                v-if="mappings[field.key]"
                @click="emit('clear-mapping', field.key)"
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
                <button @click="emit('toggle-context-editor', field.key)"
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
                              @click="emit('toggle-context-word', { fieldKey: field.key, word, direction: dir })"
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
              {{ field.multiLine ? 'Trascina una parola dalla sezione comunicazioni...' : 'Trascina qui...' }}
            </div>
            <div v-if="field.multiLine && mappings[field.key]" class="text-[10px] text-amber-600 dark:text-amber-400 mt-1">
              Estrazione multi-riga: il testo sotto l'ancora verrà catturato automaticamente
            </div>
          </div>
        </div>

        <!-- Test Pattern Button -->
        <div class="pt-3 mt-3 border-t border-gray-200 dark:border-gray-700">
          <Button
            @click="emit('test-patterns')"
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
</template>

<script setup>
import PDFTextractView from './PDFTextractView.vue'
import Button from '@/components/common/Button.vue'

defineOptions({ name: 'WizardStepMapping' })

defineProps({
  pdfAnalysis: {
    type: Object,
    default: null
  },
  extracting: {
    type: Boolean,
    default: false
  },
  mappedTokenIds: {
    type: [Set, Array],
    required: true
  },
  extractionFields: {
    type: Array,
    required: true
  },
  dragOverField: {
    type: String,
    default: null
  },
  mappings: {
    type: Object,
    required: true
  },
  testing: {
    type: Boolean,
    default: false
  },
  hasTestedPatterns: {
    type: Boolean,
    default: false
  },
  testResults: {
    type: Object,
    required: true
  },
  contextEditorOpen: {
    type: String,
    default: null
  },
  getDirectionLabel: {
    type: Function,
    required: true
  },
  getContextSummary: {
    type: Function,
    required: true
  },
  getNeighborWords: {
    type: Function,
    required: true
  },
  isContextWordSelected: {
    type: Function,
    required: true
  },
  getFieldLabel: {
    type: Function,
    required: true
  },
  directionLabels: {
    type: Object,
    required: true
  }
})

const emit = defineEmits([
  'drag-start',
  'drag-end',
  'field-drop',
  'clear-mapping',
  'toggle-context-editor',
  'toggle-context-word',
  'test-patterns',
  'update:dragOverField'
])
</script>
