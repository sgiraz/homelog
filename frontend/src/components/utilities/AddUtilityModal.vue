<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">Nuova Utenza</h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- PDF Contract Drop Zone -->
      <div class="mb-6">
        <div
          :class="[
            'border-2 border-dashed rounded-xl p-6 text-center transition-all cursor-pointer',
            isDragging
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500',
            pdfProcessing ? 'opacity-50 pointer-events-none' : ''
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

          <div v-if="pdfProcessing" class="flex flex-col items-center gap-2">
            <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="text-sm text-gray-600 dark:text-gray-400">Estrazione dati dal contratto...</span>
          </div>

          <div v-else-if="uploadedFile" class="flex items-center justify-center gap-3">
            <svg class="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div class="text-left">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ uploadedFile.name }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">Dati estratti dal contratto</p>
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
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <div>
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
                Trascina qui il PDF del contratto
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                oppure clicca per selezionare (opzionale)
              </p>
            </div>
          </div>
        </div>

        <div v-if="pdfError" class="mt-2 text-sm text-red-600 dark:text-red-400">
          {{ pdfError }}
        </div>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Tipo Utenza -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">
            Tipo *
          </label>
          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="type in utilityTypes"
              :key="type.value"
              type="button"
              @click="form.type = type.value"
              :class="[
                'p-3 rounded-lg border-2 transition-colors flex flex-col items-center gap-2',
                form.type === type.value
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
              ]"
            >
              <span :class="['text-2xl', type.iconClass]">{{ type.icon }}</span>
              <span class="text-sm font-medium text-gray-900 dark:text-white">{{ type.label }}</span>
            </button>
          </div>
        </div>

        <!-- Provider -->
        <Input
          v-model="form.provider"
          label="Fornitore *"
          placeholder="Es: Enel, Eni, Acea..."
          required
        />

        <!-- Service Code (POD/PDR) -->
        <Input
          v-model="form.service_code"
          :label="form.type === 'electricity' ? 'POD' : form.type === 'gas' ? 'PDR' : 'Codice Servizio'"
          :placeholder="form.type === 'electricity' ? 'IT001E...' : form.type === 'gas' ? 'IT001...' : ''"
        />

        <!-- Customer Code -->
        <Input
          v-model="form.customer_code"
          label="Codice Cliente"
          placeholder="Numero cliente"
        />

        <!-- Address (optional) -->
        <Input
          v-model="form.address"
          label="Indirizzo fornitura"
          placeholder="Se diverso dall'indirizzo principale"
        />

        <!-- Power Capacity (only for electricity) -->
        <Input
          v-if="form.type === 'electricity'"
          v-model="form.power_capacity"
          label="Potenza (kW)"
          type="number"
          step="0.1"
          placeholder="3.0"
        />

        <!-- Start Date -->
        <Input
          v-model="form.start_date"
          label="Data inizio contratto"
          type="date"
        />

        <!-- Customer Portal URL -->
        <Input
          v-model="form.customer_portal"
          label="Area clienti (URL)"
          type="url"
          placeholder="https://..."
        />

        <!-- Allows Self Reading (only for electricity/gas/water) -->
        <div v-if="form.type !== 'waste'" class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <input
            type="checkbox"
            id="allows-self-reading"
            v-model="form.allows_self_reading"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <div>
            <label for="allows-self-reading" class="text-sm font-medium text-gray-900 dark:text-white cursor-pointer">
              Il fornitore accetta autolettura
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Attiva se puoi comunicare le tue letture al fornitore
            </p>
          </div>
        </div>

        <!-- Comparison Threshold (only for electricity/gas/water) -->
        <div v-if="form.type !== 'waste'" class="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div class="flex items-center justify-between">
            <div>
              <label for="comparison-threshold" class="text-sm font-medium text-gray-900 dark:text-white">
                Soglia confronto letture
              </label>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Differenza % per segnalare anomalie
              </p>
            </div>
            <div class="flex items-center gap-2">
              <input
                id="comparison-threshold"
                v-model.number="form.comparison_threshold"
                type="number"
                min="1"
                max="50"
                step="1"
                class="w-16 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded
                       bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                       focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <span class="text-sm text-gray-500 dark:text-gray-400">%</span>
            </div>
          </div>
        </div>

        <!-- Notes -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Note
          </label>
          <textarea
            v-model="form.notes"
            rows="2"
            placeholder="Note aggiuntive..."
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" :disabled="loading" class="flex-1">
            {{ loading ? 'Salvataggio...' : 'Salva' }}
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import apiClient, { utilitiesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const emit = defineEmits(['close', 'created'])
const utilitiesStore = useUtilitiesStore()

const loading = ref(false)
const error = ref(null)
const fileInput = ref(null)
const isDragging = ref(false)
const pdfProcessing = ref(false)
const pdfError = ref(null)
const uploadedFile = ref(null)

const utilityTypes = [
  { value: 'electricity', label: 'Luce', icon: '\u26A1', iconClass: 'text-yellow-500' },
  { value: 'gas', label: 'Gas', icon: '\uD83D\uDD25', iconClass: 'text-orange-500' },
  { value: 'water', label: 'Acqua', icon: '\uD83D\uDCA7', iconClass: 'text-blue-500' },
  { value: 'waste', label: 'Rifiuti', icon: '\u267B\uFE0F', iconClass: 'text-green-500' }
]

const form = ref({
  type: 'electricity',
  provider: '',
  service_code: '',
  customer_code: '',
  address: '',
  power_capacity: null,
  start_date: '',
  customer_portal: '',
  notes: '',
  property_id: null,
  allows_self_reading: true,  // Default to true for most providers
  comparison_threshold: 5     // Default 5% threshold
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
    pdfError.value = 'Per favore carica un file PDF'
  }
}

async function processFile(file) {
  if (file.type !== 'application/pdf') {
    pdfError.value = 'Per favore carica un file PDF'
    return
  }

  pdfProcessing.value = true
  pdfError.value = null

  try {
    const { data } = await utilitiesAPI.uploadContractPDF(file)
    uploadedFile.value = file

    // Auto-fill form with extracted data (backend returns data directly)
    if (data) {
      if (data.provider) {
        form.value.provider = data.provider
      }
      // service_code contains POD or PDR
      if (data.service_code) {
        form.value.service_code = data.service_code
        // Detect type from code format
        if (data.service_code.startsWith('IT') && data.service_code.includes('E')) {
          form.value.type = 'electricity'
        } else if (/^\d+$/.test(data.service_code)) {
          form.value.type = 'gas'
        }
      }
      if (data.customer_code) {
        form.value.customer_code = data.customer_code
      }
      if (data.address) {
        form.value.address = data.address
      }
      if (data.power_capacity) {
        form.value.power_capacity = parseFloat(data.power_capacity.replace(',', '.'))
        form.value.type = 'electricity'
      }
    }
  } catch (err) {
    pdfError.value = err.response?.data?.error || 'Errore durante l\'estrazione dei dati dal contratto'
    console.error('PDF extraction error:', err)
  } finally {
    pdfProcessing.value = false
  }
}

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toISOString().split('T')[0]
}

function clearUploadedFile() {
  uploadedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      form.value.property_id = currentProp.id
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function handleSubmit() {
  if (!form.value.provider || !form.value.type) {
    error.value = 'Tipo e fornitore sono obbligatori'
    return
  }

  loading.value = true
  error.value = null

  try {
    const utilityData = {
      ...form.value,
      power_capacity: form.value.power_capacity ? parseFloat(form.value.power_capacity) : 0,
      start_date: form.value.start_date ? new Date(form.value.start_date).toISOString() : new Date().toISOString()
    }

    await utilitiesStore.createUtility(utilityData)
    emit('created')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchCurrentProperty()
})
</script>
