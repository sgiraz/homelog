<template>
  <BaseModal :title="isEditing ? (isMetered ? 'Modifica Bolletta' : 'Modifica Fattura') : (isMetered ? 'Nuova Bolletta' : 'Nuova Fattura')" @close="$emit('close')">

      <!-- PDF Upload (only for new bills) -->
      <PDFUploadZone
        v-if="!isEditing"
        :utility-id="utility.id"
        :utility-type="utility.type"
        :is-metered="isMetered"
        :billing-interval="utility.billing_interval"
        :billing-unit="utility.billing_unit"
        :default-template-id="utility.default_bill_template_id"
        @extracted="onPDFExtracted"
      />

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Importo Totale -->
        <Input
          v-model="form.amount_total"
          label="Importo Totale"
          type="number"
          step="0.01"
          min="0"
          placeholder="0.00"
        />

        <!-- Periodo -->
        <div class="grid grid-cols-2 gap-4 w-full min-w-0 overflow-hidden">
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

        <!-- Scadenza + Emissione -->
        <div class="grid grid-cols-2 gap-4 w-full min-w-0 overflow-hidden">
          <Input
            v-model="form.due_date"
            label="Scadenza *"
            type="date"
            required
          />
          <Input
            v-model="form.issue_date"
            label="Emissione *"
            type="date"
            required
          />
        </div>

        <!-- Consumo (metered only) -->
        <Input
          v-if="isMetered"
          v-model="form.consumption_total"
          :label="'Consumo (' + consumptionUnit + ')'"
          type="number"
          step="0.001"
          min="0"
          placeholder="0"
        />

        <!-- Numero Bolletta/Fattura -->
        <Input
          v-model="form.bill_number"
          :label="isMetered ? 'Numero Bolletta *' : 'Numero Fattura *'"
          :placeholder="isMetered ? 'Es. 32455111' : 'Es. F2601900450'"
          required
        />

        <!-- Autolettura di riferimento (metered only) -->
        <div v-if="isMetered" class="space-y-2">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Autolettura di riferimento
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            Associa l'autolettura corrispondente alla lettura finale di questa bolletta
          </p>

          <select
            v-model="form.user_reading_id"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            @change="inlineReadingValue = null"
          >
            <option :value="null">-- Nessuna autolettura --</option>
            <option
              v-for="r in sortedReadings"
              :key="r.id"
              :value="r.id"
            >
              {{ formatReadingOption(r) }}
            </option>
          </select>

          <!-- Inline reading creation -->
          <div v-if="form.user_reading_id === null" class="mt-2 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg">
            <p class="text-xs text-amber-700 dark:text-amber-300 mb-2">
              Puoi inserire una lettura al volo (verrà creata automaticamente):
            </p>
            <div class="flex gap-2 items-center">
              <input
                v-model="inlineReadingValue"
                type="number"
                step="0.001"
                :placeholder="form.provider_reading ? String(form.provider_reading) : '0'"
                class="flex-1 px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded
                       bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                       focus:outline-none focus:ring-1 focus:ring-amber-500"
              />
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ readingUnit }}</span>
            </div>
            <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
              Lascia vuoto per usare la lettura del fornitore come riferimento
            </p>
          </div>
        </div>

        <!-- Provider Readings Section (metered only) -->
        <div v-if="isMetered" class="border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
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
              @click="isEditingProviderReadings = !isEditingProviderReadings"
              class="text-xs text-blue-600 dark:text-blue-400 hover:underline"
            >
              {{ isEditingProviderReadings ? 'Nascondi' : 'Modifica' }}
            </button>
          </div>

          <!-- Collapsed view when readings exist -->
          <div v-show="hasProviderReadings && !isEditingProviderReadings">
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
            <div v-else-if="utility.type === 'gas'" class="space-y-1">
              <div class="text-center bg-white dark:bg-gray-800 rounded p-2">
                <p class="text-xs text-gray-500 dark:text-gray-400">Lettura</p>
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading) }} mc</p>
              </div>
              <div v-if="form.conversion_coefficient" class="text-center bg-white dark:bg-gray-800 rounded p-2">
                <p class="text-xs text-gray-500 dark:text-gray-400">Coeff. C: {{ form.conversion_coefficient }}</p>
              </div>
            </div>
            <div v-else class="text-center bg-white dark:bg-gray-800 rounded p-2">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatNumber(form.provider_reading) }} mc</p>
            </div>
          </div>

          <!-- Expanded form for manual entry -->
          <div v-show="isEditingProviderReadings" class="space-y-3">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Inserisci le letture riportate in bolletta per confrontarle con le tue autoletture
            </p>

            <!-- Electricity readings (F1/F2/F3) -->
            <div v-if="utility.type === 'electricity'" class="grid grid-cols-3 gap-2">
              <div>
                <label class="block text-xs text-red-600 dark:text-red-400 mb-1 font-medium">F1 (kWh)</label>
                <input v-model="form.provider_reading_f1" type="number" step="0.001" placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
              </div>
              <div>
                <label class="block text-xs text-yellow-600 dark:text-yellow-400 mb-1 font-medium">F2 (kWh)</label>
                <input v-model="form.provider_reading_f2" type="number" step="0.001" placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
              </div>
              <div>
                <label class="block text-xs text-green-600 dark:text-green-400 mb-1 font-medium">F3 (kWh)</label>
                <input v-model="form.provider_reading_f3" type="number" step="0.001" placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
              </div>
            </div>

            <!-- Gas: reading + conversion coefficient -->
            <div v-else-if="utility.type === 'gas'" class="space-y-3">
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Lettura Contatore (mc)</label>
                <input v-model="form.provider_reading" type="number" step="0.001" placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
              </div>
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Coefficiente di Conversione (C)</label>
                <input v-model="form.conversion_coefficient" type="number" step="0.00000001" min="0" placeholder="1.00000000"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
              </div>
              <div v-if="previousBillHasEstimate && !previousBill?.estimated_reading">
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Consumi Precedenti Stimati (Smc)</label>
                <input v-model="form.previous_estimated_consumption" type="number" step="0.000001" placeholder="0"
                  class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">La bolletta precedente contiene una stima di {{ formatNumber(previousBill.estimated_consumption) }} Smc</p>
              </div>
            </div>

            <!-- Water single reading -->
            <div v-else-if="utility.type === 'water'">
              <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Lettura Contatore (mc)</label>
              <input v-model="form.provider_reading" type="number" step="0.001" placeholder="0"
                class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
            </div>
          </div>
        </div>

        <!-- Estimated consumption toggle (gas only) -->
        <div v-if="isMetered && utility.type === 'gas'" class="border-t border-gray-200 dark:border-gray-700 pt-3 mt-3">
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.has_estimated"
              class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-amber-500" />
            <span class="text-sm text-gray-700 dark:text-gray-300">Contiene lettura stimata</span>
          </label>
          <div v-if="form.has_estimated" class="mt-3 space-y-3 pl-4 border-l-2 border-amber-300 dark:border-amber-600">
            <div>
              <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Data stima</label>
              <input v-model="form.estimated_date" type="date"
                class="w-full min-w-0 max-w-full box-border px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-amber-500" />
            </div>
            <div>
              <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">Lettura Stimata (mc)</label>
              <input v-model="form.estimated_reading" type="number" step="0.001" placeholder="0"
                class="w-full px-2 py-1.5 text-sm border border-gray-200 dark:border-gray-700 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-amber-500" />
              <p v-if="calculatedEstimatedConsumption != null" class="text-xs text-amber-600 dark:text-amber-400 mt-1">
                Consumo stimato: {{ formatNumber(calculatedEstimatedConsumption) }} Smc
              </p>
            </div>
          </div>
        </div>

        <!-- Comunicazioni importanti -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Comunicazioni importanti
          </label>
          <textarea
            v-model="form.communication_text"
            rows="3"
            :placeholder="isMetered ? 'Eventuali comunicazioni rilevanti...' : 'Es. Modifica condizioni contrattuali, variazioni di prezzo...'"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
            Inserisci comunicazioni importanti contenute nella {{ isMetered ? 'bolletta' : 'fattura' }} (variazioni prezzo, scadenze recesso, ecc.)
          </p>
        </div>

        <!-- Stato Pagamento -->
        <div class="flex items-center gap-3">
          <input type="checkbox" id="is-paid" v-model="form.is_paid"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500" />
          <label for="is-paid" class="text-sm text-gray-900 dark:text-white cursor-pointer">
            Già pagata
          </label>
        </div>

        <div v-if="submitError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ submitError }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" :disabled="saving" class="flex-1">
            {{ saving ? 'Salvataggio...' : 'Salva' }}
          </Button>
        </div>
      </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate, formatNumber as _formatNumber } from '@/utils/dateFormatter'
import { utilitiesAPI } from '@/api/client'
import { useConsumptionCalculation } from '@/composables/useConsumptionCalculation'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'
import PDFUploadZone from '@/components/utilities/PDFUploadZone.vue'

const props = defineProps({
  utility: { type: Object, required: true },
  bill: { type: Object, default: null }
})

const emit = defineEmits(['close', 'saved'])
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const isMetered = computed(() => ['electricity', 'gas', 'water', 'waste'].includes(props.utility?.type))
const isEditing = computed(() => !!props.bill)

const saving = ref(false)
const submitError = ref(null)

// Available readings for this utility
const availableReadings = ref([])
const inlineReadingValue = ref(null)

const readingUnit = computed(() => {
  const type = props.utility?.type
  if (type === 'electricity') return 'kWh'
  if (type === 'gas' || type === 'water') return 'mc'
  return ''
})

const consumptionUnit = computed(() => {
  const units = { electricity: 'kWh', gas: 'Smc', water: 'mc', waste: '' }
  return units[props.utility?.type] || ''
})

const sortedReadings = computed(() => {
  return [...availableReadings.value].sort((a, b) => new Date(b.reading_date) - new Date(a.reading_date))
})

// Provider readings edit state
const isEditingProviderReadings = ref(true)

const form = ref({
  amount_total: null,
  period_start: '',
  period_end: '',
  due_date: '',
  issue_date: '',
  consumption_total: null,
  bill_number: '',
  user_reading_id: null,
  reading_type: 'actual',
  is_paid: false,
  provider_reading_date: null,
  provider_reading_f1: null,
  provider_reading_f2: null,
  provider_reading_f3: null,
  provider_reading: null,
  conversion_coefficient: null,
  has_estimated: false,
  estimated_date: '',
  estimated_reading: null,
  previous_estimated_consumption: null,
  communication_text: ''
})

const hasProviderReadings = computed(() => {
  if (props.utility?.type === 'electricity') {
    return form.value.provider_reading_f1 || form.value.provider_reading_f2 || form.value.provider_reading_f3
  }
  if (props.utility?.type === 'gas') {
    return form.value.provider_reading != null || form.value.conversion_coefficient != null
  }
  return form.value.provider_reading != null
})

// Consumption calculation composable
const { previousBill, previousBillHasEstimate, calculatedEstimatedConsumption } = useConsumptionCalculation(
  form,
  computed(() => props.utility),
  isEditing,
  props.bill
)

// ── Helpers ──

function formatNumber(value) {
  if (value == null) return '-'
  return _formatNumber(value, settingsStore.formatSettings)
}

function formatReadingOption(r) {
  const d = new Date(r.reading_date)
  const dateStr = _formatDate(d, settingsStore.dateSettings)
  const val = r.value != null ? _formatNumber(r.value, settingsStore.formatSettings) : '-'
  return `${dateStr} — ${val} ${readingUnit.value}`
}

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return ''
  return date.toISOString().split('T')[0]
}

// ── PDF Extraction ──

function onPDFExtracted(data) {
  if (!data) return
  form.value.amount_total = data.amount_total
  if (data.consumption_total != null) form.value.consumption_total = data.consumption_total
  if (data.conversion_coefficient != null) form.value.conversion_coefficient = data.conversion_coefficient
  if (data.period_start) form.value.period_start = data.period_start
  if (data.period_end) form.value.period_end = data.period_end
  if (data.due_date) form.value.due_date = data.due_date
  if (data.issue_date) form.value.issue_date = data.issue_date
  if (data.bill_number) form.value.bill_number = data.bill_number
  if (data.reading_type) form.value.reading_type = data.reading_type
  if (data.provider_reading_date) form.value.provider_reading_date = data.provider_reading_date
  if (data.provider_reading_f1 != null) form.value.provider_reading_f1 = data.provider_reading_f1
  if (data.provider_reading_f2 != null) form.value.provider_reading_f2 = data.provider_reading_f2
  if (data.provider_reading_f3 != null) form.value.provider_reading_f3 = data.provider_reading_f3
  if (data.provider_reading != null) form.value.provider_reading = data.provider_reading
  if (data.estimated_date) { form.value.estimated_date = data.estimated_date; form.value.has_estimated = true }
  if (data.estimated_reading != null) { form.value.estimated_reading = data.estimated_reading; form.value.has_estimated = true }
  if (data.previous_estimated_consumption != null && previousBillHasEstimate.value) {
    form.value.previous_estimated_consumption = data.previous_estimated_consumption
  }
  if (data.communication_text) form.value.communication_text = data.communication_text
}

// ── Readings ──

async function fetchReadings() {
  try {
    const { data } = await utilitiesAPI.getReadings(props.utility.id)
    availableReadings.value = data || []
  } catch (err) {
    console.error('Error loading readings:', err)
  }
}

// ── Submit ──

async function handleSubmit() {
  if (isMetered.value && (form.value.consumption_total == null || form.value.consumption_total === '')) {
    submitError.value = 'Il consumo è obbligatorio'
    return
  }

  saving.value = true
  submitError.value = null

  try {
    // Create inline reading if needed
    let resolvedReadingId = form.value.user_reading_id
    if (resolvedReadingId === null && inlineReadingValue.value != null && inlineReadingValue.value !== '') {
      const readingDate = form.value.period_end
        ? new Date(form.value.period_end).toISOString()
        : new Date().toISOString()
      const { data: newReading } = await utilitiesAPI.addReading(props.utility.id, {
        reading_date: readingDate,
        value: parseFloat(inlineReadingValue.value),
        notes: `Creata in fase di inserimento bolletta ${form.value.bill_number}`
      })
      resolvedReadingId = newReading.id
    }

    const billData = {
      amount_total: parseFloat(form.value.amount_total) || 0,
      period_start: new Date(form.value.period_start).toISOString(),
      period_end: new Date(form.value.period_end).toISOString(),
      due_date: new Date(form.value.due_date).toISOString(),
      issue_date: new Date(form.value.issue_date).toISOString(),
      consumption_total: isMetered.value ? (parseFloat(form.value.consumption_total) || 0) : 0,
      conversion_coefficient: props.utility.type === 'gas' && form.value.conversion_coefficient
        ? parseFloat(form.value.conversion_coefficient) : null,
      bill_number: form.value.bill_number,
      user_reading_id: resolvedReadingId || null,
      reading_type: form.value.reading_type,
      is_paid: form.value.is_paid,
      paid_date: form.value.is_paid ? new Date().toISOString() : null,
      provider_reading_date: form.value.provider_reading_date ? new Date(form.value.provider_reading_date).toISOString() : null,
      provider_reading_f1: form.value.provider_reading_f1 ? parseFloat(form.value.provider_reading_f1) : null,
      provider_reading_f2: form.value.provider_reading_f2 ? parseFloat(form.value.provider_reading_f2) : null,
      provider_reading_f3: form.value.provider_reading_f3 ? parseFloat(form.value.provider_reading_f3) : null,
      provider_reading: form.value.provider_reading ? parseFloat(form.value.provider_reading) : null,
      estimated_date: form.value.has_estimated && form.value.estimated_date
        ? new Date(form.value.estimated_date).toISOString() : null,
      estimated_reading: form.value.has_estimated && form.value.estimated_reading != null
        ? parseFloat(form.value.estimated_reading) : null,
      estimated_consumption: form.value.has_estimated && calculatedEstimatedConsumption.value != null
        ? calculatedEstimatedConsumption.value : null,
      communication_text: form.value.communication_text || ''
    }

    if (isEditing.value) {
      await utilitiesStore.updateBillFull(props.utility.id, props.bill.id, billData)
    } else {
      await utilitiesStore.addBill(props.utility.id, billData)
    }
    emit('saved')
  } catch (err) {
    submitError.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    saving.value = false
  }
}

// ── Init ──

onMounted(async () => {
  if (props.bill) {
    form.value = {
      amount_total: props.bill.amount_total,
      period_start: formatDateForInput(props.bill.period_start),
      period_end: formatDateForInput(props.bill.period_end),
      due_date: formatDateForInput(props.bill.due_date),
      issue_date: formatDateForInput(props.bill.issue_date),
      consumption_total: props.bill.consumption_total,
      bill_number: props.bill.bill_number || '',
      user_reading_id: props.bill.user_reading_id || null,
      reading_type: props.bill.reading_type || 'actual',
      is_paid: props.bill.is_paid || false,
      provider_reading_date: formatDateForInput(props.bill.provider_reading_date),
      provider_reading_f1: props.bill.provider_reading_f1,
      provider_reading_f2: props.bill.provider_reading_f2,
      provider_reading_f3: props.bill.provider_reading_f3,
      provider_reading: props.bill.provider_reading,
      conversion_coefficient: props.bill.conversion_coefficient || null,
      has_estimated: props.bill.estimated_reading != null || props.bill.estimated_consumption != null,
      estimated_date: formatDateForInput(props.bill.estimated_date),
      estimated_reading: props.bill.estimated_reading,
      previous_estimated_consumption: null,
      communication_text: ''
    }

    // Load existing communication
    try {
      const { data: comms } = await utilitiesAPI.getCommunications(props.utility.id)
      const billComm = comms.find(c => c.bill_id === props.bill.id)
      if (billComm) form.value.communication_text = billComm.content || ''
    } catch { /* non-critical */ }

    // Collapse provider readings if they exist
    const hasExisting = props.utility?.type === 'electricity'
      ? (props.bill.provider_reading_f1 || props.bill.provider_reading_f2 || props.bill.provider_reading_f3)
      : props.bill.provider_reading != null
    if (hasExisting) isEditingProviderReadings.value = false
  }

  await fetchReadings()

  // Auto-suggest closest reading for new bills
  if (!props.bill && sortedReadings.value.length > 0 && form.value.period_end) {
    const periodEnd = new Date(form.value.period_end)
    let closest = null
    let closestDiff = Infinity
    for (const r of availableReadings.value) {
      const diff = Math.abs(new Date(r.reading_date) - periodEnd)
      if (diff < closestDiff) { closestDiff = diff; closest = r }
    }
    if (closest && closestDiff <= 45 * 24 * 60 * 60 * 1000) {
      form.value.user_reading_id = closest.id
    }
  }
})
</script>
