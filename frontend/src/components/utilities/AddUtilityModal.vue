<template>
  <BaseModal :title="t('utilities.addUtilityModal.title')" @close="$emit('close')">
    <!-- PDF Contract Drop Zone -->
    <div class="mb-6">
      <div
        :class="[
          'border-2 border-dashed rounded-xl p-6 text-center transition-all cursor-pointer',
          isDragging
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-line hover:border-ink-faint',
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
          <span class="text-sm text-ink-soft">{{ t('utilities.addUtilityModal.extracting') }}</span>
        </div>

        <div v-else-if="uploadedFile" class="flex items-center justify-center gap-3">
          <svg class="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div class="text-left">
            <p class="text-sm font-medium text-ink">{{ uploadedFile.name }}</p>
            <p class="text-xs text-ink-muted">{{ t('utilities.addUtilityModal.extractedFromContract') }}</p>
          </div>
          <button
            type="button"
            @click.stop="clearUploadedFile"
            class="text-ink-faint hover:text-ink-soft"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-else class="flex flex-col items-center gap-2">
          <svg class="w-10 h-10 text-ink-faint" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <div>
            <p class="text-sm font-medium text-ink-soft">
              {{ t('utilities.addUtilityModal.dropZoneTitle') }}
            </p>
            <p class="text-xs text-ink-muted mt-1">
              {{ t('utilities.addUtilityModal.dropZoneSubtitle') }}
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
        <label class="block text-sm text-ink-soft mb-2">
          {{ t('utilities.addUtilityModal.typeLabel') }}
        </label>
        <div class="text-xs text-ink-faint mb-1.5 font-medium uppercase tracking-wider">{{ t('utilities.addUtilityModal.categoryMetered') }}</div>
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
                : 'border-line hover:border-line'
            ]"
          >
            <span :class="['text-2xl', type.iconClass]">{{ type.icon }}</span>
            <span class="text-sm font-medium text-ink">{{ type.label }}</span>
          </button>
        </div>
        <div class="text-xs text-ink-faint mb-1.5 font-medium uppercase tracking-wider">{{ t('utilities.addUtilityModal.categoryFixed') }}</div>
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
                : 'border-line hover:border-line'
            ]"
          >
            <span :class="['text-2xl', type.iconClass]">{{ type.icon }}</span>
            <span class="text-sm font-medium text-ink">{{ type.label }}</span>
          </button>
        </div>
      </div>

      <!-- Provider -->
      <Input
        v-model="form.provider"
        :label="isMetered ? t('utilities.addUtilityModal.providerLabel') : t('utilities.addUtilityModal.operatorLabel')"
        :placeholder="isMetered ? t('utilities.addUtilityModal.providerPlaceholder') : t('utilities.addUtilityModal.operatorPlaceholder')"
        required
      />

      <!-- Service Code -->
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
        :label="isMetered ? t('utilities.addUtilityModal.customerCodeLabel') : t('utilities.addUtilityModal.contractNumberLabel')"
        :placeholder="isMetered ? t('utilities.addUtilityModal.customerCodePlaceholder') : t('utilities.addUtilityModal.contractRefPlaceholder')"
      />

      <!-- Recurring Amount -->
      <Input
        v-if="!isMetered"
        v-model="form.recurring_amount"
        :label="t('utilities.addUtilityModal.recurringAmountLabel')"
        type="number"
        step="0.01"
        :placeholder="t('utilities.addUtilityModal.recurringAmountPlaceholder')"
        inputmode="decimal"
      />

      <!-- Billing Frequency -->
      <div v-if="!isMetered">
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('utilities.addUtilityModal.billingFrequencyLabel') }}
        </label>
        <div class="flex gap-2">
          <input
            v-model.number="form.billing_interval"
            type="number"
            min="1"
            max="365"
            inputmode="numeric"
            class="w-20 px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base text-center
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            v-model="form.billing_unit"
            class="flex-1 px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="day">{{ frequencyUnitLabel('day') }}</option>
            <option value="week">{{ frequencyUnitLabel('week') }}</option>
            <option value="month">{{ frequencyUnitLabel('month') }}</option>
            <option value="year">{{ frequencyUnitLabel('year') }}</option>
          </select>
        </div>
        <p class="text-xs text-ink-faint mt-1">{{ frequencyPreview }}</p>
      </div>

      <!-- Address -->
      <Input
        v-model="form.address"
        :label="isMetered ? t('utilities.addUtilityModal.addressMetered') : t('utilities.addUtilityModal.addressFixed')"
        :placeholder="t('utilities.addUtilityModal.addressPlaceholder')"
        autocomplete="street-address"
      />

      <!-- Power Capacity -->
      <Input
        v-if="form.type === 'electricity'"
        v-model="form.power_capacity"
        :label="t('utilities.addUtilityModal.powerLabel')"
        type="number"
        step="0.1"
        placeholder="3.0"
        inputmode="decimal"
      />

      <!-- Start Date -->
      <Input
        v-model="form.start_date"
        :label="t('utilities.addUtilityModal.startDateLabel')"
        type="date"
      />

      <!-- Customer Portal URL -->
      <Input
        v-model="form.customer_portal"
        :label="t('utilities.addUtilityModal.portalLabel')"
        type="url"
        placeholder="https://..."
        inputmode="url"
      />

      <!-- Allows Self Reading -->
      <div v-if="isMetered" class="flex items-center gap-3 p-3 bg-surface rounded-lg">
        <input
          type="checkbox"
          id="allows-self-reading"
          v-model="form.allows_self_reading"
          class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
        />
        <div>
          <label for="allows-self-reading" class="text-sm font-medium text-ink cursor-pointer">
            {{ t('utilities.addUtilityModal.allowsSelfReadingLabel') }}
          </label>
          <p class="text-xs text-ink-muted">
            {{ t('utilities.addUtilityModal.allowsSelfReadingHint') }}
          </p>
        </div>
      </div>

      <!-- Comparison Threshold -->
      <div v-if="isMetered" class="p-3 bg-surface rounded-lg">
        <div class="flex items-center justify-between">
          <div>
            <label for="comparison-threshold" class="text-sm font-medium text-ink">
              {{ t('utilities.addUtilityModal.thresholdLabel') }}
            </label>
            <p class="text-xs text-ink-muted">
              {{ t('utilities.addUtilityModal.thresholdHint') }}
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
              class="w-16 px-2 py-1 text-sm text-center border border-line rounded
                     bg-surface text-ink
                     focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span class="text-sm text-ink-muted">{{ consumptionUnitLabel }}</span>
          </div>
        </div>
      </div>

      <!-- Chi paga -->
      <div v-if="members.length > 1">
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('utilities.addUtilityModal.paidByLabel') }}
        </label>
        <select
          v-model="form.paid_by_member_id"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">{{ t('utilities.addUtilityModal.notSpecified') }}</option>
          <option
            v-for="member in members"
            :key="member.id"
            :value="member.id"
          >
            {{ member.name }}{{ member.role ? ` (${member.role})` : '' }}
          </option>
        </select>
        <p class="text-xs text-ink-faint mt-1">{{ t('utilities.addUtilityModal.paidByHint') }}</p>
      </div>

      <!-- Split Override -->
      <div v-if="members.length > 1" class="p-4 bg-surface rounded-lg space-y-3">
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('utilities.addUtilityModal.splitOverrideLabel') }}
          </label>
          <select
            v-model="form.split_override"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">{{ t('utilities.addUtilityModal.splitOverrideGlobal') }}</option>
            <option value="no_split">{{ t('utilities.addUtilityModal.splitOverrideNoSplit') }}</option>
            <option value="custom">{{ t('utilities.addUtilityModal.splitOverrideCustom') }}</option>
          </select>
          <p class="text-xs text-ink-faint mt-1">
            {{ splitOverrideHint }}
          </p>
        </div>

        <div v-if="form.split_override === 'custom'" class="space-y-2">
          <label class="block text-sm text-ink-soft">
            {{ t('utilities.addUtilityModal.splitWithLabel') }}
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
              class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
            />
            <label :for="'add-split-member-' + member.id" class="text-sm text-ink cursor-pointer">
              {{ member.name }}{{ member.role ? ` (${member.role})` : '' }}
            </label>
          </div>
        </div>
      </div>

      <!-- Billing flags -->
      <div class="space-y-2">
        <label class="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            v-model="form.is_domiciled"
            class="mt-0.5 w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
          />
          <div>
            <div class="text-sm text-ink">{{ t('utilities.addUtilityModal.domiciledLabel') }}</div>
            <div class="text-xs text-ink-muted">{{ t('utilities.addUtilityModal.domiciledHint') }}</div>
          </div>
        </label>
        <label v-if="form.type !== 'mutuo'" class="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            v-model="form.is_installment_based"
            class="mt-0.5 w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
          />
          <div>
            <div class="text-sm text-ink">{{ t('utilities.addUtilityModal.installmentsLabel') }}</div>
            <div class="text-xs text-ink-muted">{{ t('utilities.addUtilityModal.installmentsHint') }}</div>
          </div>
        </label>
      </div>

      <!-- Currency -->
      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('utilities.addUtilityModal.currencyLabel') }}
        </label>
        <select
          v-model="form.currency"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option v-for="opt in currencyOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p class="text-xs text-ink-muted mt-1">
          {{ t('utilities.addUtilityModal.currencyLockNotice') }}
        </p>
      </div>

      <!-- Notes -->
      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('utilities.addUtilityModal.notesLabel') }}
        </label>
        <textarea
          v-model="form.notes"
          rows="2"
          :placeholder="t('utilities.addUtilityModal.notesPlaceholder')"
          autocorrect="off"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          {{ t('utilities.addUtilityModal.cancel') }}
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? t('utilities.addUtilityModal.saving') : t('utilities.addUtilityModal.save') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
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
const { t } = useI18n()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()

const currencyOptions = computed(() => {
  const base = settingsStore.formatSettings.currency || 'EUR'
  return [
    { value: '', label: t('utilities.addUtilityModal.currencyGlobalOption', { base }) },
    { value: 'EUR', label: 'EUR — Euro' },
    { value: 'USD', label: 'USD — Dollaro USA' },
    { value: 'GBP', label: 'GBP — Sterlina' },
    { value: 'CHF', label: 'CHF — Franco svizzero' },
    { value: 'JPY', label: 'JPY — Yen' },
    { value: 'CAD', label: 'CAD — Dollaro canadese' },
    { value: 'AUD', label: 'AUD — Dollaro australiano' },
    { value: 'SEK', label: 'SEK — Corona svedese' },
    { value: 'NOK', label: 'NOK — Corona norvegese' },
    { value: 'DKK', label: 'DKK — Corona danese' },
    { value: 'PLN', label: 'PLN — Złoty' },
  ]
})

const loading = ref(false)
const error = ref(null)
const fileInput = ref(null)
const isDragging = ref(false)
const pdfProcessing = ref(false)
const pdfError = ref(null)
const uploadedFile = ref(null)
const members = ref([])
const splitMemberIds = ref([])

function typeLabel(value) {
  const key = `utilities.utilityTypes.${value}`
  const label = t(key)
  return label === key ? value : label
}

const meteredTypes = computed(() => [
  { value: 'electricity', label: typeLabel('electricity'), icon: '⚡', iconClass: 'text-yellow-500', metered: true },
  { value: 'gas', label: typeLabel('gas'), icon: '🔥', iconClass: 'text-orange-500', metered: true },
  { value: 'water', label: typeLabel('water'), icon: '💧', iconClass: 'text-blue-500', metered: true },
])

const fixedTypes = computed(() => [
  // waste (TARI) is billed on surface area, not on a meter — see Utility.IsMetered.
  { value: 'waste', label: typeLabel('waste'), icon: '♻️', iconClass: 'text-green-500', metered: false },
  { value: 'internet', label: typeLabel('internet'), icon: '🌐', iconClass: 'text-indigo-500', metered: false },
  { value: 'insurance', label: typeLabel('insurance'), icon: '🛡️', iconClass: 'text-emerald-500', metered: false },
  { value: 'affitto', label: typeLabel('affitto'), icon: '🏠', iconClass: 'text-purple-500', metered: false },
  { value: 'mutuo', label: typeLabel('mutuo'), icon: '🏦', iconClass: 'text-sky-500', metered: false },
])

const isMetered = computed(() => {
  const allTypes = [...meteredTypes.value, ...fixedTypes.value]
  const found = allTypes.find(tt => tt.value === form.value.type)
  return found ? found.metered : true
})

const serviceCodeLabel = computed(() => {
  const key = `utilities.addUtilityModal.serviceCodeLabels.${form.value.type}`
  const label = t(key)
  return label === key ? t('utilities.addUtilityModal.serviceCodeLabels.default') : label
})

const serviceCodePlaceholder = computed(() => {
  const key = `utilities.addUtilityModal.serviceCodePlaceholder.${form.value.type}`
  const label = t(key)
  return label === key ? '' : label
})

const consumptionUnitLabel = computed(() => {
  const units = { electricity: 'kWh', gas: 'Smc', water: 'm³' }
  return units[form.value.type] || ''
})

function frequencyUnitLabel(unit) {
  const suffix = (form.value.billing_interval === 1) ? '_one' : '_other'
  return t(`utilities.addUtilityModal.frequency.${unit}${suffix}`)
}

const frequencyPreview = computed(() => {
  const n = form.value.billing_interval || 1
  const u = form.value.billing_unit
  const word = frequencyUnitLabel(u).toLowerCase()
  return n === 1
    ? t('utilities.addUtilityModal.frequencyPreviewSingular', { unit: word })
    : t('utilities.addUtilityModal.frequencyPreviewPlural', { n, unit: word })
})

// Always set the type through this: the payload carries is_metered alongside
// type, so assigning form.type on its own submits the PREVIOUS type's metering
// — which is how a mutuo reached the API with is_metered=true.
function applyType(value) {
  const known = [...meteredTypes.value, ...fixedTypes.value].find(t => t.value === value)
  form.value.type = value
  form.value.is_metered = known?.metered ?? false
}

function selectType(type) {
  applyType(type.value)
  // A mortgage isn't "one bill split into installments" (installmentsHint) —
  // it already generates one bill per billing period on its own.
  if (type.value === 'mutuo') {
    form.value.is_installment_based = false
  }
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
  is_installment_based: false,
  currency: ''
})

const splitOverrideHint = computed(() => {
  switch (form.value.split_override) {
    case 'no_split': return t('utilities.addUtilityModal.splitOverrideHintNoSplit')
    case 'custom': return t('utilities.addUtilityModal.splitOverrideHintCustom')
    default: return t('utilities.addUtilityModal.splitOverrideHintGlobal')
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
    pdfError.value = t('utilities.addUtilityModal.uploadOnlyPdf')
  }
}

async function processFile(file) {
  if (file.type !== 'application/pdf') {
    pdfError.value = t('utilities.addUtilityModal.uploadOnlyPdf')
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
          applyType('electricity')
        } else if (/^\d+$/.test(data.service_code)) {
          applyType('gas')
        }
      }
      if (data.customer_code) form.value.customer_code = data.customer_code
      if (data.address) form.value.address = data.address
      if (data.power_capacity) {
        form.value.power_capacity = parseFloat(data.power_capacity.replace(',', '.'))
        applyType('electricity')
      }
    }
  } catch (err) {
    pdfError.value = err.response?.data?.error || t('utilities.addUtilityModal.extractError')
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
    error.value = t('utilities.addUtilityModal.validationRequired')
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
    window.$toast?.success(t('utilities.addUtilityModal.successToast'))
    emit('created')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('utilities.addUtilityModal.genericError')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchCurrentProperty()
})
</script>
