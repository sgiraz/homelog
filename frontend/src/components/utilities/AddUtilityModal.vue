<template>
  <BaseModal title="Nuovo Servizio" @close="$emit('close')">
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
      <!-- Tipo Servizio -->
      <div>
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">
          Tipo *
        </label>
        <!-- Category labels -->
        <div class="text-xs text-gray-400 dark:text-gray-500 mb-1.5 font-medium uppercase tracking-wider">Utenze</div>
        <div class="grid grid-cols-2 gap-2 mb-3">
          <button
            v-for="type in meteredTypes"
            :key="type.value"
            type="button"
            @click="selectType(type)"
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
        <div class="text-xs text-gray-400 dark:text-gray-500 mb-1.5 font-medium uppercase tracking-wider">Abbonamenti e Ricorrenti</div>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="type in fixedTypes"
            :key="type.value"
            type="button"
            @click="selectType(type)"
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
        :label="isMetered ? 'Fornitore *' : 'Operatore / Ente *'"
        :placeholder="isMetered ? 'Nome del fornitore' : 'es. WindTre, Allianz, Proprietario...'"
        required
      />

      <!-- Service Code (context-aware) -->
      <Input
        v-model="form.service_code"
        :label="serviceCodeLabel"
        :placeholder="serviceCodePlaceholder"
        autocorrect="off"
        autocapitalize="off"
      />

      <!-- Customer Code -->
      <Input
        v-model="form.customer_code"
        :label="isMetered ? 'Codice Cliente' : 'Numero Contratto'"
        :placeholder="isMetered ? 'Numero cliente' : 'Riferimento contratto'"
      />

      <!-- Recurring Amount (fixed services only) -->
      <Input
        v-if="!isMetered"
        v-model="form.recurring_amount"
        label="Importo ricorrente (€)"
        type="number"
        step="0.01"
        placeholder="es. 29.99"
        inputmode="decimal"
      />

      <!-- Billing Frequency (fixed services only) -->
      <div v-if="!isMetered">
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          Frequenza fatturazione
        </label>
        <div class="flex gap-2">
          <input
            v-model.number="form.billing_interval"
            type="number"
            min="1"
            max="365"
            inputmode="numeric"
            class="w-20 px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base text-center
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            v-model="form.billing_unit"
            class="flex-1 px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="day">{{ form.billing_interval === 1 ? 'Giorno' : 'Giorni' }}</option>
            <option value="week">{{ form.billing_interval === 1 ? 'Settimana' : 'Settimane' }}</option>
            <option value="month">{{ form.billing_interval === 1 ? 'Mese' : 'Mesi' }}</option>
            <option value="year">{{ form.billing_interval === 1 ? 'Anno' : 'Anni' }}</option>
          </select>
        </div>
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ frequencyPreview }}</p>
      </div>

      <!-- Address (optional) -->
      <Input
        v-model="form.address"
        :label="isMetered ? 'Indirizzo fornitura' : 'Indirizzo'"
        placeholder="Se diverso dall'indirizzo principale"
        autocomplete="street-address"
      />

      <!-- Power Capacity (only for electricity) -->
      <Input
        v-if="form.type === 'electricity'"
        v-model="form.power_capacity"
        label="Potenza (kW)"
        type="number"
        step="0.1"
        placeholder="3.0"
        inputmode="decimal"
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
        inputmode="url"
      />

      <!-- Allows Self Reading (metered only, not waste) -->
      <div v-if="isMetered && form.type !== 'waste'" class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
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

      <!-- Comparison Threshold (metered only, not waste) -->
      <div v-if="isMetered && form.type !== 'waste'" class="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <div class="flex items-center justify-between">
          <div>
            <label for="comparison-threshold" class="text-sm font-medium text-gray-900 dark:text-white">
              Soglia confronto letture
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Differenza per segnalare anomalie
            </p>
          </div>
          <div class="flex items-center gap-2">
            <input
              id="comparison-threshold"
              v-model.number="form.comparison_threshold"
              type="number"
              min="0.01"
              max="50"
              step="0.01"
              inputmode="decimal"
              class="w-16 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded
                     bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                     focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ consumptionUnitLabel }}</span>
          </div>
        </div>
      </div>

      <!-- Chi paga -->
      <div v-if="members.length > 1">
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          Chi paga
        </label>
        <select
          v-model="form.paid_by_member_id"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">Non specificato</option>
          <option
            v-for="member in members"
            :key="member.id"
            :value="member.id"
          >
            {{ member.name }}{{ member.role ? ` (${member.role})` : '' }}
          </option>
        </select>
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">Chi paga di default le bollette/fatture di questo servizio</p>
      </div>

      <!-- Split Override -->
      <div v-if="members.length > 1" class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg space-y-3">
        <div>
          <label class="block text-sm font-medium text-gray-900 dark:text-white mb-1">
            Divisione spese
          </label>
          <select
            v-model="form.split_override"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Usa impostazione globale</option>
            <option value="no_split">Mai dividere</option>
            <option value="custom">Dividi con membri specifici</option>
          </select>
          <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
            {{ splitOverrideHint }}
          </p>
        </div>

        <!-- Custom split member selection -->
        <div v-if="form.split_override === 'custom'" class="space-y-2">
          <label class="block text-sm text-gray-600 dark:text-gray-400">
            Dividi con
          </label>
          <div
            v-for="member in members"
            :key="'split-' + member.id"
            class="flex items-center gap-3"
          >
            <input
              type="checkbox"
              :id="'add-split-member-' + member.id"
              :value="member.id"
              v-model="splitMemberIds"
              class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
            />
            <label :for="'add-split-member-' + member.id" class="text-sm text-gray-900 dark:text-white cursor-pointer">
              {{ member.name }}{{ member.role ? ` (${member.role})` : '' }}
            </label>
          </div>
        </div>
      </div>

      <!-- Billing flags: domiciliation + installments -->
      <div class="space-y-2">
        <label class="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            v-model="form.is_domiciled"
            class="mt-0.5 w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <div>
            <div class="text-sm text-gray-900 dark:text-white">Domiciliata</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">I pagamenti vengono marcati automaticamente come saldati alla scadenza</div>
          </div>
        </label>
        <label class="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            v-model="form.is_installment_based"
            class="mt-0.5 w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <div>
            <div class="text-sm text-gray-900 dark:text-white">Bollette rateizzate</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">La bolletta è suddivisa in più rate con scadenze separate</div>
          </div>
        </label>
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
          autocorrect="off"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          Annulla
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? 'Salvataggio...' : 'Salva' }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import apiClient, { utilitiesAPI, membersAPI } from '@/api/client'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  defaultPropertyId: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['close', 'created'])
const utilitiesStore = useUtilitiesStore()

const loading = ref(false)
const error = ref(null)
const fileInput = ref(null)
const isDragging = ref(false)
const pdfProcessing = ref(false)
const pdfError = ref(null)
const uploadedFile = ref(null)
const members = ref([])
const splitMemberIds = ref([])

const meteredTypes = [
  { value: 'electricity', label: 'Luce', icon: '\u26A1', iconClass: 'text-yellow-500', metered: true },
  { value: 'gas', label: 'Gas', icon: '\uD83D\uDD25', iconClass: 'text-orange-500', metered: true },
  { value: 'water', label: 'Acqua', icon: '\uD83D\uDCA7', iconClass: 'text-blue-500', metered: true },
  { value: 'waste', label: 'Rifiuti', icon: '\u267B\uFE0F', iconClass: 'text-green-500', metered: true },
]

const fixedTypes = [
  { value: 'internet', label: 'Internet', icon: '\uD83C\uDF10', iconClass: 'text-indigo-500', metered: false },
  { value: 'insurance', label: 'Assicurazione', icon: '\uD83D\uDEE1\uFE0F', iconClass: 'text-emerald-500', metered: false },
  { value: 'affitto', label: 'Affitto', icon: '\uD83C\uDFE0', iconClass: 'text-purple-500', metered: false },
  { value: 'mutuo', label: 'Mutuo', icon: '\uD83C\uDFE6', iconClass: 'text-sky-500', metered: false },
]

const isMetered = computed(() => {
  const allTypes = [...meteredTypes, ...fixedTypes]
  const found = allTypes.find(t => t.value === form.value.type)
  return found ? found.metered : true
})

const serviceCodeLabel = computed(() => {
  const labels = { electricity: 'POD', gas: 'PDR', internet: 'Numero linea', affitto: 'Riferimento contratto', mutuo: 'Numero mutuo' }
  return labels[form.value.type] || 'Codice Servizio'
})

const serviceCodePlaceholder = computed(() => {
  const placeholders = { electricity: 'IT001E...', gas: 'IT001...', internet: '04XXXXXXXX', affitto: '', mutuo: '' }
  return placeholders[form.value.type] || ''
})

const consumptionUnitLabel = computed(() => {
  const units = { electricity: 'kWh', gas: 'Smc', water: 'm\u00B3' }
  return units[form.value.type] || ''
})

const frequencyPreview = computed(() => {
  const n = form.value.billing_interval || 1
  const u = form.value.billing_unit
  const unitLabels = {
    day: n === 1 ? 'giorno' : 'giorni',
    week: n === 1 ? 'settimana' : 'settimane',
    month: n === 1 ? 'mese' : 'mesi',
    year: n === 1 ? 'anno' : 'anni'
  }
  return n === 1 ? `Ogni ${unitLabels[u]}` : `Ogni ${n} ${unitLabels[u]}`
})

function selectType(type) {
  form.value.type = type.value
  form.value.is_metered = type.metered
}

const form = ref({
  type: 'electricity',
  is_metered: true,
  provider: '',
  service_code: '',
  customer_code: '',
  address: '',
  power_capacity: null,
  recurring_amount: null,
  billing_interval: 1,
  billing_unit: 'month',
  paid_by_member_id: null,
  split_override: '',
  start_date: '',
  customer_portal: '',
  notes: '',
  property_id: null,
  allows_self_reading: true,
  comparison_threshold: 5,
  is_domiciled: false,
  is_installment_based: false
})

const splitOverrideHint = computed(() => {
  switch (form.value.split_override) {
    case 'no_split': return 'Le spese di questo servizio non verranno mai divise'
    case 'custom': return 'Le spese verranno divise con i membri selezionati sotto'
    default: return 'Segue le impostazioni di divisione della famiglia'
  }
})

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

    if (data) {
      if (data.provider) form.value.provider = data.provider
      if (data.service_code) {
        form.value.service_code = data.service_code
        if (data.service_code.startsWith('IT') && data.service_code.includes('E')) {
          form.value.type = 'electricity'
        } else if (/^\d+$/.test(data.service_code)) {
          form.value.type = 'gas'
        }
      }
      if (data.customer_code) form.value.customer_code = data.customer_code
      if (data.address) form.value.address = data.address
      if (data.power_capacity) {
        form.value.power_capacity = parseFloat(data.power_capacity.replace(',', '.'))
        form.value.type = 'electricity'
      }
    }
  } catch (err) {
    pdfError.value = err.response?.data?.error || 'Errore durante l\'estrazione dei dati dal contratto'
  } finally {
    pdfProcessing.value = false
  }
}

function clearUploadedFile() {
  uploadedFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function fetchCurrentProperty() {
  if (props.defaultPropertyId) {
    form.value.property_id = props.defaultPropertyId
  } else {
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

  // Fetch household members for payer selection
  if (form.value.property_id) {
    try {
      const { data } = await membersAPI.list(form.value.property_id)
      members.value = data || []
    } catch (err) {
      console.error('Error fetching members:', err)
    }
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
      recurring_amount: form.value.recurring_amount ? parseFloat(form.value.recurring_amount) : undefined,
      billing_interval: form.value.billing_interval || 1,
      billing_unit: form.value.billing_unit || 'month',
      paid_by_member_id: form.value.paid_by_member_id || undefined,
      split_override: form.value.split_override,
      split_member_ids: form.value.split_override === 'custom' ? JSON.stringify(splitMemberIds.value) : '',
      start_date: form.value.start_date ? new Date(form.value.start_date).toISOString() : new Date().toISOString()
    }

    await utilitiesStore.createUtility(utilityData)
    window.$toast?.success('Servizio aggiunto con successo!')
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
