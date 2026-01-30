<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[60] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-md p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ isEditing ? 'Modifica Bolletta' : 'Nuova Bolletta' }}
        </h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- PDF Drop Zone (only for new bills) -->
      <div
        v-if="!isEditing"
        class="mb-6"
      >
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
                Trascina qui il PDF della bolletta
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                oppure clicca per selezionare
              </p>
            </div>
          </div>
        </div>

        <div v-if="pdfError" class="mt-2 text-sm text-red-600 dark:text-red-400">
          {{ pdfError }}
        </div>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Importo Totale -->
        <Input
          v-model="form.amount_total"
          label="Importo Totale *"
          type="number"
          step="0.01"
          min="0.01"
          placeholder="0.00"
          required
        />

        <!-- Periodo -->
        <div class="grid grid-cols-2 gap-4">
          <Input
            v-model="form.period_start"
            label="Inizio Periodo *"
            type="date"
            required
          />
          <Input
            v-model="form.period_end"
            label="Fine Periodo *"
            type="date"
            required
          />
        </div>

        <!-- Scadenza -->
        <Input
          v-model="form.due_date"
          label="Scadenza Pagamento *"
          type="date"
          required
        />

        <!-- Data Emissione -->
        <Input
          v-model="form.issue_date"
          label="Data Emissione *"
          type="date"
          required
        />

        <!-- Consumo -->
        <Input
          v-model="form.consumption_total"
          :label="'Consumo (' + getConsumptionUnit(utility.type) + ') *'"
          type="number"
          step="0.01"
          min="0"
          placeholder="0"
          required
        />

        <!-- Numero Bolletta -->
        <Input
          v-model="form.bill_number"
          label="Numero Bolletta"
          placeholder="Opzionale"
        />

        <!-- Tipo Lettura -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Tipo Lettura
          </label>
          <select
            v-model="form.reading_type"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="actual">Effettiva</option>
            <option value="estimated">Stimata</option>
          </select>
        </div>

        <!-- Provider Readings Section -->
        <div class="border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <svg class="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <span class="text-sm font-medium text-blue-700 dark:text-blue-300">
                Letture Fornitore (per confronto)
              </span>
            </div>
            <button
              type="button"
              @click="toggleProviderReadingsEdit"
              class="text-xs text-blue-600 dark:text-blue-400 hover:underline"
            >
              {{ isEditingProviderReadings ? 'Nascondi' : 'Modifica' }}
            </button>
          </div>

          <!-- Collapsed view when readings exist -->
          <div v-show="showCollapsedView">
            <div v-if="utility.type === 'electricity'" class="grid grid-cols-3 gap-2 text-center">
              <div class="bg-white dark:bg-gray-800 rounded p-2">
                <p class="text-xs text-red-600 dark:text-red-400 font-medium">F1</p>
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading_f1) }}</p>
              </div>
              <div class="bg-white dark:bg-gray-800 rounded p-2">
                <p class="text-xs text-yellow-600 dark:text-yellow-400 font-medium">F2</p>
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading_f2) }}</p>
              </div>
              <div class="bg-white dark:bg-gray-800 rounded p-2">
                <p class="text-xs text-green-600 dark:text-green-400 font-medium">F3</p>
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading_f3) }}</p>
              </div>
            </div>
            <div v-else class="text-center bg-white dark:bg-gray-800 rounded p-2">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading) }} mc</p>
            </div>
          </div>

          <!-- Expanded form for manual entry -->
          <div v-show="showExpandedForm" class="space-y-3">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Inserisci le letture riportate in bolletta per confrontarle con le tue autoletture
            </p>

            <!-- Electricity readings (F1/F2/F3) -->
            <div v-if="utility.type === 'electricity'" class="grid grid-cols-3 gap-2">
              <div>
                <label class="block text-xs text-red-600 dark:text-red-400 mb-1 font-medium">F1 (kWh)</label>
                <input
                  v-model="form.provider_reading_f1"
                  type="number"
                  step="0.01"
                  placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label class="block text-xs text-yellow-600 dark:text-yellow-400 mb-1 font-medium">F2 (kWh)</label>
                <input
                  v-model="form.provider_reading_f2"
                  type="number"
                  step="0.01"
                  placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label class="block text-xs text-green-600 dark:text-green-400 mb-1 font-medium">F3 (kWh)</label>
                <input
                  v-model="form.provider_reading_f3"
                  type="number"
                  step="0.01"
                  placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>

            <!-- Gas/Water single reading -->
            <div v-else-if="utility.type === 'gas' || utility.type === 'water'">
              <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Lettura Contatore (mc)</label>
              <input
                v-model="form.provider_reading"
                type="number"
                step="0.01"
                placeholder="0"
                class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                       focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
          </div>
        </div>

        <!-- Stato Pagamento -->
        <div class="flex items-center gap-3">
          <input
            type="checkbox"
            id="is-paid"
            v-model="form.is_paid"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <label for="is-paid" class="text-sm text-gray-900 dark:text-white cursor-pointer">
            Già pagata
          </label>
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
import { ref, computed, onMounted } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import { utilitiesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  utility: {
    type: Object,
    required: true
  },
  bill: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])
const utilitiesStore = useUtilitiesStore()

const loading = ref(false)
const error = ref(null)
const fileInput = ref(null)
const isDragging = ref(false)
const pdfProcessing = ref(false)
const pdfError = ref(null)
const uploadedFile = ref(null)

const isEditing = computed(() => !!props.bill)

// Stato per l'editing delle letture fornitore - inizializzato in onMounted
const isEditingProviderReadings = ref(true)

const form = ref({
  amount_total: null,
  period_start: '',
  period_end: '',
  due_date: '',
  issue_date: '',
  consumption_total: null,
  bill_number: '',
  reading_type: 'actual',
  is_paid: false,
  // Provider readings (letture rilevate)
  provider_reading_date: null,
  provider_reading_f1: null,
  provider_reading_f2: null,
  provider_reading_f3: null,
  provider_reading: null
})

// Track if we have provider readings
const hasProviderReadings = computed(() => {
  if (props.utility?.type === 'electricity') {
    return form.value.provider_reading_f1 || form.value.provider_reading_f2 || form.value.provider_reading_f3
  }
  return form.value.provider_reading != null
})

// Computed properties per la UI - logica centralizzata e chiara
const showCollapsedView = computed(() => {
  return hasProviderReadings.value && !isEditingProviderReadings.value
})

const showExpandedForm = computed(() => {
  return isEditingProviderReadings.value
})

// Toggle per il pulsante Modifica/Nascondi
function toggleProviderReadingsEdit() {
  isEditingProviderReadings.value = !isEditingProviderReadings.value
}

function getConsumptionUnit(type) {
  const units = {
    electricity: 'kWh',
    gas: 'Smc',
    water: 'mc',
    waste: ''
  }
  return units[type] || ''
}

function formatNumber(value) {
  if (value == null) return '-'
  return value.toLocaleString('it-IT', { maximumFractionDigits: 2 })
}

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toISOString().split('T')[0]
}

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
    const { data } = await utilitiesAPI.uploadBillPDF(props.utility.id, file)
    uploadedFile.value = file

    // Auto-fill form with extracted data (backend returns data directly)
    if (data) {
      if (data.amount_total) {
        form.value.amount_total = data.amount_total
      }
      if (data.consumption_total) {
        form.value.consumption_total = data.consumption_total
      }
      if (data.period_start) {
        form.value.period_start = formatDateForInput(data.period_start)
      }
      if (data.period_end) {
        form.value.period_end = formatDateForInput(data.period_end)
      }
      if (data.due_date) {
        form.value.due_date = formatDateForInput(data.due_date)
      }
      if (data.issue_date) {
        form.value.issue_date = formatDateForInput(data.issue_date)
      }
      if (data.bill_number) {
        form.value.bill_number = data.bill_number
      }
      if (data.reading_type) {
        form.value.reading_type = data.reading_type
      }
      // Provider readings (letture rilevate dal fornitore)
      if (data.provider_reading_date) {
        form.value.provider_reading_date = formatDateForInput(data.provider_reading_date)
      }
      if (data.provider_reading_f1 != null) {
        form.value.provider_reading_f1 = data.provider_reading_f1
      }
      if (data.provider_reading_f2 != null) {
        form.value.provider_reading_f2 = data.provider_reading_f2
      }
      if (data.provider_reading_f3 != null) {
        form.value.provider_reading_f3 = data.provider_reading_f3
      }
      if (data.provider_reading != null) {
        form.value.provider_reading = data.provider_reading
      }
    }
  } catch (err) {
    pdfError.value = err.response?.data?.error || 'Errore durante l\'estrazione dei dati dal PDF'
    console.error('PDF extraction error:', err)
  } finally {
    pdfProcessing.value = false
  }
}

function clearUploadedFile() {
  uploadedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

async function handleSubmit() {
  if (!form.value.amount_total || !form.value.consumption_total) {
    error.value = 'Importo e consumo sono obbligatori'
    return
  }

  loading.value = true
  error.value = null

  try {
    const billData = {
      amount_total: parseFloat(form.value.amount_total),
      period_start: new Date(form.value.period_start).toISOString(),
      period_end: new Date(form.value.period_end).toISOString(),
      due_date: new Date(form.value.due_date).toISOString(),
      issue_date: new Date(form.value.issue_date).toISOString(),
      consumption_total: parseFloat(form.value.consumption_total),
      bill_number: form.value.bill_number,
      reading_type: form.value.reading_type,
      is_paid: form.value.is_paid,
      paid_date: form.value.is_paid ? new Date().toISOString() : null,
      // Provider readings (letture rilevate dal fornitore)
      provider_reading_date: form.value.provider_reading_date ? new Date(form.value.provider_reading_date).toISOString() : null,
      provider_reading_f1: form.value.provider_reading_f1 ? parseFloat(form.value.provider_reading_f1) : null,
      provider_reading_f2: form.value.provider_reading_f2 ? parseFloat(form.value.provider_reading_f2) : null,
      provider_reading_f3: form.value.provider_reading_f3 ? parseFloat(form.value.provider_reading_f3) : null,
      provider_reading: form.value.provider_reading ? parseFloat(form.value.provider_reading) : null
    }

    if (isEditing.value) {
      await utilitiesStore.updateBillFull(props.utility.id, props.bill.id, billData)
    } else {
      await utilitiesStore.addBill(props.utility.id, billData)
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (props.bill) {
    form.value = {
      amount_total: props.bill.amount_total,
      period_start: formatDateForInput(props.bill.period_start),
      period_end: formatDateForInput(props.bill.period_end),
      due_date: formatDateForInput(props.bill.due_date),
      issue_date: formatDateForInput(props.bill.issue_date),
      consumption_total: props.bill.consumption_total,
      bill_number: props.bill.bill_number || '',
      reading_type: props.bill.reading_type || 'actual',
      is_paid: props.bill.is_paid || false,
      // Provider readings
      provider_reading_date: formatDateForInput(props.bill.provider_reading_date),
      provider_reading_f1: props.bill.provider_reading_f1,
      provider_reading_f2: props.bill.provider_reading_f2,
      provider_reading_f3: props.bill.provider_reading_f3,
      provider_reading: props.bill.provider_reading
    }

    // Se stiamo editando una bolletta con letture esistenti, mostra la collapsed view
    const hasExistingReadings = props.utility?.type === 'electricity'
      ? (props.bill.provider_reading_f1 || props.bill.provider_reading_f2 || props.bill.provider_reading_f3)
      : props.bill.provider_reading != null

    if (hasExistingReadings) {
      isEditingProviderReadings.value = false
    }
  }
  // Per nuove bollette, isEditingProviderReadings rimane true (default)
})
</script>
