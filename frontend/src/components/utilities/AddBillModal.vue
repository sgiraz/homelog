<template>
  <BaseModal :title="modalTitle" @close="$emit('close')">

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
        <div v-if="isLocked" class="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 text-xs text-amber-800 dark:text-amber-200">
          <span>🔒</span>
          <span>{{ t('utilities.addBillModal.lockedNotice') }}</span>
        </div>

        <!-- Importo Totale -->
        <div v-if="isForeignCurrency">
          <label class="block text-sm text-ink-soft mb-1">
            {{ t('utilities.addBillModal.totalAmountForeign', { currency: utility.currency }) }}
          </label>
          <Input
            v-model="form.original_amount"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            :disabled="isLocked"
          />
          <div class="mt-1.5">
            <div v-if="rateLoading" class="text-xs text-gray-400">
              {{ t('utilities.addBillModal.rateConverting') }}
            </div>
            <div v-else-if="form.original_amount && convertedAmount != null" class="text-xs text-green-600 dark:text-green-400">
              {{ formatOriginal(form.original_amount, utility.currency) }} ≈ {{ formatCurrency(convertedAmount) }}
              <span class="text-gray-400">{{ t('utilities.addBillModal.rateInfo', { rate: exchangeRate?.toFixed(6) }) }}</span>
            </div>
            <div v-else-if="rateError" class="space-y-1">
              <p class="text-xs text-amber-600 dark:text-amber-400">
                {{ t('utilities.addBillModal.rateUnavailable') }}
              </p>
              <div class="flex items-center gap-2">
                <span class="text-xs text-gray-500">{{ t('utilities.addBillModal.currencyEquals', { currency: utility.currency }) }}</span>
                <input
                  v-model.number="manualRate"
                  type="number"
                  step="any"
                  min="0"
                  placeholder="0.00"
                  inputmode="decimal"
                  class="w-28 px-2 py-1 text-sm border border-line rounded
                         bg-surface text-ink
                         focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
                <span class="text-xs text-gray-500">{{ settingsStore.currency }}</span>
              </div>
              <div v-if="form.original_amount && manualConvertedAmount != null" class="text-xs text-green-600 dark:text-green-400">
                {{ formatOriginal(form.original_amount, utility.currency) }} ≈ {{ formatCurrency(manualConvertedAmount) }}
              </div>
            </div>
          </div>
        </div>
        <Input
          v-else
          v-model="form.amount_total"
          :label="t('utilities.addBillModal.totalAmount')"
          type="number"
          step="0.01"
          min="0"
          placeholder="0.00"
          :disabled="isLocked"
        />

        <!-- Periodo -->
        <div class="grid grid-cols-2 gap-4 w-full min-w-0 overflow-hidden">
          <Input
            v-model="form.period_start"
            :label="t('utilities.addBillModal.periodStart')"
            type="date"
            required
          />
          <Input
            v-model="form.period_end"
            :label="t('utilities.addBillModal.periodEnd')"
            type="date"
            required
          />
        </div>

        <!-- Scadenza + Emissione -->
        <div class="grid grid-cols-2 gap-4 w-full min-w-0 overflow-hidden">
          <Input
            v-model="form.due_date"
            :label="t('utilities.addBillModal.due')"
            type="date"
            required
          />
          <Input
            v-model="form.issue_date"
            :label="t('utilities.addBillModal.issue')"
            type="date"
            required
          />
        </div>

        <!-- Consumo (metered only) -->
        <Input
          v-if="isMetered"
          v-model="form.consumption_total"
          :label="t('utilities.addBillModal.consumption', { unit: consumptionUnit })"
          type="number"
          step="0.001"
          min="0"
          placeholder="0"
        />

        <!-- Numero Bolletta/Fattura -->
        <Input
          v-model="form.bill_number"
          :label="isMetered ? t('utilities.addBillModal.billNumber') : t('utilities.addBillModal.invoiceNumber')"
          :placeholder="isMetered ? t('utilities.addBillModal.billNumberPlaceholder') : t('utilities.addBillModal.invoiceNumberPlaceholder')"
          required
        />

        <!-- Autolettura di riferimento (metered only) -->
        <div v-if="isMetered" class="space-y-2">
          <label class="block text-sm font-medium text-ink-soft">
            {{ t('utilities.addBillModal.userReadingLabel') }}
          </label>
          <p class="text-xs text-ink-muted">
            {{ t('utilities.addBillModal.userReadingHint') }}
          </p>

          <select
            v-model="form.user_reading_id"
            class="w-full px-3 py-2 border border-line rounded-lg
                   bg-surface text-ink
                   focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            @change="inlineReadingValue = null"
          >
            <option :value="null">{{ t('utilities.addBillModal.noReading') }}</option>
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
              {{ t('utilities.addBillModal.inlineReadingHint') }}
            </p>
            <div class="flex gap-2 items-center">
              <input
                v-model="inlineReadingValue"
                type="number"
                step="0.001"
                :placeholder="form.provider_reading ? String(form.provider_reading) : '0'"
                class="flex-1 px-2 py-1.5 text-sm border border-line rounded
                       bg-surface text-ink
                       focus:outline-none focus:ring-1 focus:ring-amber-500"
              />
              <span class="text-xs text-ink-muted">{{ readingUnit }}</span>
            </div>
            <p class="text-xs text-ink-faint mt-1">
              {{ t('utilities.addBillModal.inlineReadingFooter') }}
            </p>
          </div>
        </div>

        <!-- Provider Readings Section (metered only) -->
        <ProviderReadingsSection
          v-if="isMetered"
          v-model:f1="form.provider_reading_f1"
          v-model:f2="form.provider_reading_f2"
          v-model:f3="form.provider_reading_f3"
          v-model:reading="form.provider_reading"
          v-model:conversion-coefficient="form.conversion_coefficient"
          v-model:previous-estimated-consumption="form.previous_estimated_consumption"
          :utility-type="utility.type"
          :previous-bill="previousBill"
          :previous-bill-has-estimate="previousBillHasEstimate"
        />

        <!-- Estimated consumption toggle (gas + water) -->
        <div v-if="isMetered && (utility.type === 'gas' || utility.type === 'water')" class="border-t border-line pt-3 mt-3">
          <div class="flex items-center gap-2">
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="form.has_estimated"
                class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-amber-500" />
              <span class="text-sm text-ink-soft">{{ t('utilities.addBillModal.estimatedToggle') }}</span>
            </label>
            <button type="button" @click="showEstimatedHelp = !showEstimatedHelp"
              class="w-5 h-5 inline-flex items-center justify-center rounded-full border border-line text-xs font-semibold text-ink-muted hover:bg-surface-2"
              :aria-expanded="showEstimatedHelp"
              :aria-label="t('utilities.addBillModal.estimatedHelpAria')">?</button>
          </div>
          <div v-if="showEstimatedHelp"
            class="mt-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 text-xs text-amber-900 dark:text-amber-100 space-y-1.5">
            <p>{{ t('utilities.addBillModal.estimatedHelp1') }}</p>
            <p>{{ t('utilities.addBillModal.estimatedHelp2') }}</p>
            <p class="text-amber-700 dark:text-amber-300">{{ t('utilities.addBillModal.estimatedHelp3') }}</p>
          </div>
          <div v-if="form.has_estimated" class="mt-3 space-y-3 pl-4 border-l-2 border-amber-300 dark:border-amber-600">
            <div>
              <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.addBillModal.estimatedDate') }}</label>
              <input v-model="form.estimated_date" type="date"
                class="w-full min-w-0 max-w-full box-border px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-amber-500" />
            </div>
            <div>
              <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.addBillModal.estimatedReading') }}</label>
              <input v-model="form.estimated_reading" type="number" step="0.001" placeholder="0"
                class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-amber-500" />
              <p v-if="calculatedEstimatedConsumption != null" class="text-xs text-amber-600 dark:text-amber-400 mt-1">
                {{ t('utilities.addBillModal.estimatedConsumption', { value: formatNumber(calculatedEstimatedConsumption), unit: utility.type === 'gas' ? 'Smc' : 'mc' }) }}
              </p>
            </div>
          </div>
        </div>

        <!-- Comunicazioni importanti -->
        <div>
          <label class="block text-sm text-ink-soft mb-1">
            {{ t('utilities.addBillModal.communicationsLabel') }}
          </label>
          <textarea
            v-model="form.communication_text"
            rows="3"
            :placeholder="isMetered ? t('utilities.addBillModal.communicationsPlaceholderBill') : t('utilities.addBillModal.communicationsPlaceholderInvoice')"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p class="text-xs text-ink-faint mt-1">
            {{ isMetered ? t('utilities.addBillModal.communicationsHintBill') : t('utilities.addBillModal.communicationsHintInvoice') }}
          </p>
        </div>

        <!-- Rate (installments) -->
        <InstallmentsSection
          v-if="isInstallmentBased"
          v-model="form.installments"
          :utility-id="utility.id"
          :bill-id="bill?.id"
          :is-editing="isEditing"
          :amount-total="form.amount_total"
          :default-due-date="form.due_date"
          @installment-updated="emit('installment-updated')"
          @error="submitError = $event"
        />

        <!-- Stato Pagamento -->
        <div v-if="!isInstallmentBased" class="flex items-center gap-3">
          <input type="checkbox" id="is-paid" v-model="form.is_paid"
            :disabled="isLocked"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed" />
          <label for="is-paid" class="text-sm cursor-pointer"
            :class="isLocked ? 'text-ink-faint cursor-not-allowed' : 'text-ink'">
            {{ isLocked ? t('utilities.addBillModal.alreadyPaidLocked') : t('utilities.addBillModal.alreadyPaid') }}
          </label>
        </div>

        <div v-if="submitError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ submitError }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            {{ t('utilities.addBillModal.cancel') }}
          </Button>
          <Button type="submit" :disabled="saving" class="flex-1">
            {{ saving ? t('utilities.addBillModal.saving') : t('utilities.addBillModal.save') }}
          </Button>
        </div>
      </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate, formatNumber as _formatNumber, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import { utilitiesAPI, exchangeAPI } from '@/api/client'
import { useConsumptionCalculation } from '@/composables/useConsumptionCalculation'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'
import PDFUploadZone from '@/components/utilities/PDFUploadZone.vue'
import ProviderReadingsSection from '@/components/utilities/ProviderReadingsSection.vue'
import InstallmentsSection from '@/components/utilities/InstallmentsSection.vue'

const props = defineProps({
  utility: { type: Object, required: true },
  bill: { type: Object, default: null }
})

const emit = defineEmits(['close', 'saved', 'installment-updated'])
const { t } = useI18n()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const isMetered = computed(() => ['electricity', 'gas', 'water'].includes(props.utility?.type))
const isEditing = computed(() => !!props.bill)
const isInstallmentBased = computed(() => !!props.utility?.is_installment_based)
const isLocked = computed(() => !!props.bill?.is_locked)
const showEstimatedHelp = ref(false)

const modalTitle = computed(() => {
  if (isEditing.value) {
    return isMetered.value ? t('utilities.addBillModal.editBillTitle') : t('utilities.addBillModal.editInvoiceTitle')
  }
  return isMetered.value ? t('utilities.addBillModal.newBillTitle') : t('utilities.addBillModal.newInvoiceTitle')
})

const saving = ref(false)
const submitError = ref(null)

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

const isForeignCurrency = computed(() => {
  const u = props.utility?.currency
  if (!u) return false
  return u !== settingsStore.currency
})

const exchangeRate = ref(null)
const convertedAmount = ref(null)
const rateLoading = ref(false)
const rateError = ref(false)
const manualRate = ref(null)
let fetchRateTimer = null

onUnmounted(() => {
  clearTimeout(fetchRateTimer)
})

const manualConvertedAmount = computed(() => {
  if (!manualRate.value || !form.value.original_amount) return null
  return parseFloat(form.value.original_amount) * manualRate.value
})

const finalConvertedAmount = computed(() => {
  if (!isForeignCurrency.value) return null
  if (convertedAmount.value != null) return convertedAmount.value
  if (manualConvertedAmount.value != null) return manualConvertedAmount.value
  return null
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}
function formatOriginal(value, currency) {
  return _formatCurrency(value, { ...settingsStore.formatSettings, currency })
}

async function fetchExchangeRate() {
  const amount = parseFloat(form.value.original_amount)
  if (!isForeignCurrency.value || !amount || amount <= 0) {
    exchangeRate.value = null
    convertedAmount.value = null
    return
  }
  rateLoading.value = true
  rateError.value = false
  try {
    const { data } = await exchangeAPI.getRate(props.utility.currency, settingsStore.currency, amount)
    exchangeRate.value = data.rate
    convertedAmount.value = data.result
  } catch {
    rateError.value = true
    exchangeRate.value = null
    convertedAmount.value = null
  } finally {
    rateLoading.value = false
  }
}

const form = ref({
  amount_total: null,
  original_amount: null,
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
  communication_text: '',
  installments: []
})

watch(() => form.value.original_amount, () => {
  if (!isForeignCurrency.value) return
  clearTimeout(fetchRateTimer)
  fetchRateTimer = setTimeout(fetchExchangeRate, 500)
})

watch(finalConvertedAmount, (val) => {
  if (!isForeignCurrency.value) return
  form.value.amount_total = val
})

const installmentsAmountMismatch = computed(() => {
  if (!isInstallmentBased.value || form.value.installments.length <= 1) return false
  const sum = (form.value.installments || []).reduce((s, i) => s + (parseFloat(i.amount) || 0), 0)
  return Math.abs(sum - (parseFloat(form.value.amount_total) || 0)) > 0.01
})

const { previousBill, previousBillHasEstimate, calculatedEstimatedConsumption } = useConsumptionCalculation(
  form,
  computed(() => props.utility),
  isEditing,
  props.bill
)

function formatNumber(value) {
  if (value == null) return '-'
  return _formatNumber(value, settingsStore.formatSettings)
}

function formatReadingOption(r) {
  const d = new Date(r.reading_date)
  const dateStr = _formatDate(d, settingsStore.dateSettings)
  const val = r.value != null ? _formatNumber(r.value, settingsStore.formatSettings) : '-'
  return t('utilities.addBillModal.readingOption', { date: dateStr, value: val, unit: readingUnit.value })
}

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return ''
  return date.toISOString().split('T')[0]
}

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

async function fetchReadings() {
  try {
    const { data } = await utilitiesAPI.getReadings(props.utility.id)
    availableReadings.value = data || []
  } catch (err) {
    console.error('Error loading readings:', err)
  }
}

async function handleSubmit() {
  if (isMetered.value && (form.value.consumption_total == null || form.value.consumption_total === '')) {
    submitError.value = t('utilities.addBillModal.consumptionRequired')
    return
  }
  if (isInstallmentBased.value && !isEditing.value && installmentsAmountMismatch.value) {
    submitError.value = t('utilities.addBillModal.installmentsMismatch')
    return
  }
  if (isForeignCurrency.value) {
    if (!form.value.original_amount) {
      submitError.value = t('utilities.addBillModal.foreignAmountRequired', { currency: props.utility.currency })
      return
    }
    if (finalConvertedAmount.value == null) {
      submitError.value = t('utilities.addBillModal.rateMissing')
      return
    }
  }

  saving.value = true
  submitError.value = null

  try {
    let resolvedReadingId = form.value.user_reading_id
    if (resolvedReadingId === null && inlineReadingValue.value != null && inlineReadingValue.value !== '') {
      const readingDate = form.value.period_end
        ? new Date(form.value.period_end).toISOString()
        : new Date().toISOString()
      const { data: newReading } = await utilitiesAPI.addReading(props.utility.id, {
        reading_date: readingDate,
        value: parseFloat(inlineReadingValue.value),
        notes: t('utilities.addBillModal.inlineReadingNote', { number: form.value.bill_number })
      })
      resolvedReadingId = newReading.id
    }

    const billData = {
      amount_total: parseFloat(form.value.amount_total) || 0,
      original_amount: isForeignCurrency.value ? parseFloat(form.value.original_amount) : undefined,
      original_currency: isForeignCurrency.value ? props.utility.currency : undefined,
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

    if (isInstallmentBased.value && !isEditing.value && form.value.installments.length > 0) {
      const first = form.value.installments[0]
      if (first.due_date) {
        billData.due_date = new Date(first.due_date).toISOString()
      }
      billData.installments = form.value.installments.map(inst => ({
        number: inst.number,
        due_date: inst.due_date ? new Date(inst.due_date).toISOString() : new Date(form.value.due_date).toISOString(),
        amount: parseFloat(inst.amount) || 0,
        is_paid: !!inst.is_paid,
        paid_at: null
      }))
    }

    if (isEditing.value) {
      await utilitiesStore.updateBillFull(props.utility.id, props.bill.id, billData)
    } else {
      await utilitiesStore.addBill(props.utility.id, billData)
    }
    emit('saved')
  } catch (err) {
    submitError.value = err.response?.data?.error || err.message || t('utilities.addBillModal.genericError')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (props.bill) {
    form.value = {
      amount_total: props.bill.amount_total,
      original_amount: props.bill.original_amount ?? null,
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
      communication_text: '',
      installments: (props.bill.installments || []).map(inst => ({
        id: inst.id,
        number: inst.number,
        due_date: formatDateForInput(inst.due_date),
        amount: inst.amount,
        is_paid: !!inst.is_paid,
        is_locked: !!inst.is_locked
      }))
    }

    try {
      const { data: comms } = await utilitiesAPI.getCommunications(props.utility.id)
      const billComm = comms.find(c => c.bill_id === props.bill.id)
      if (billComm) form.value.communication_text = billComm.content || ''
    } catch { /* non-critical */ }
  }

  if (!props.bill && isInstallmentBased.value && form.value.installments.length === 0) {
    form.value.installments.push({ number: 1, due_date: form.value.due_date || '', amount: 0, is_paid: false })
  }

  await fetchReadings()

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
