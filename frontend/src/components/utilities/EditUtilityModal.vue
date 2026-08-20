<template>
  <BaseModal :title="t('utilities.editUtilityModal.title')" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Tipo Servizio (read-only badge) -->
      <div>
        <label class="block text-sm text-ink-soft mb-2">
          {{ t('utilities.editUtilityModal.typeLabel') }}
        </label>
        <div class="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg bg-surface-2 border border-line">
          <span class="text-2xl">{{ typeInfo.icon }}</span>
          <span class="text-sm font-medium text-ink">{{ typeInfo.label }}</span>
        </div>
      </div>

      <!-- Attivo / Disattivo toggle -->
      <div class="flex items-center gap-3 p-3 bg-surface rounded-lg">
        <button
          type="button"
          @click="form.is_active = !form.is_active"
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
            form.is_active ? 'bg-blue-600' : 'bg-surface-3'
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
              form.is_active ? 'translate-x-6' : 'translate-x-1'
            ]"
          />
        </button>
        <div>
          <span class="text-sm font-medium text-ink">
            {{ form.is_active ? t('utilities.editUtilityModal.active') : t('utilities.editUtilityModal.inactive') }}
          </span>
          <p class="text-xs text-ink-muted">
            {{ form.is_active ? t('utilities.editUtilityModal.activeHint') : t('utilities.editUtilityModal.inactiveHint') }}
          </p>
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

      <!-- End Date -->
      <Input
        v-model="form.end_date"
        :label="t('utilities.editUtilityModal.endDateLabel')"
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
          id="edit-allows-self-reading"
          v-model="form.allows_self_reading"
          class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
        />
        <div>
          <label for="edit-allows-self-reading" class="text-sm font-medium text-ink cursor-pointer">
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
            <label for="edit-comparison-threshold" class="text-sm font-medium text-ink">
              {{ t('utilities.addUtilityModal.thresholdLabel') }}
            </label>
            <p class="text-xs text-ink-muted">
              {{ t('utilities.addUtilityModal.thresholdHint') }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <input
              id="edit-comparison-threshold"
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

      <!-- Threshold per Day -->
      <div v-if="isMetered" class="p-3 bg-surface rounded-lg">
        <div class="flex items-center justify-between">
          <div>
            <label for="edit-threshold-per-day" class="text-sm font-medium text-ink">
              {{ t('utilities.editUtilityModal.thresholdPerDayLabel') }}
            </label>
            <p class="text-xs text-ink-muted">
              {{ t('utilities.editUtilityModal.thresholdPerDayHint') }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <input
              id="edit-threshold-per-day"
              v-model.number="form.threshold_per_day"
              type="number"
              min="0"
              step="0.1"
              inputmode="decimal"
              class="w-20 px-2 py-1 text-sm text-center border border-line rounded
                     bg-surface text-ink
                     focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span class="text-sm text-ink-muted">{{ t('utilities.editUtilityModal.thresholdPerDayUnit', { unit: consumptionUnitLabel }) }}</span>
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
              :id="'split-member-' + member.id"
              :value="member.id"
              v-model="splitMemberIds"
              class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
            />
            <label :for="'split-member-' + member.id" class="text-sm text-ink cursor-pointer">
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
          :disabled="isCurrencyLocked"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500
                 disabled:opacity-60 disabled:cursor-not-allowed"
        >
          <option v-for="opt in currencyOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p v-if="isCurrencyLocked" class="text-xs text-amber-600 dark:text-amber-400 mt-1">
          {{ t('utilities.editUtilityModal.currencyLocked') }}
        </p>
        <p v-else class="text-xs text-ink-muted mt-1">
          {{ t('utilities.addUtilityModal.currencyLockNotice') }}
        </p>
      </div>

      <!-- Default Bill Template -->
      <div v-if="billTemplates.length > 0">
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('utilities.editUtilityModal.billTemplateLabel') }}
        </label>
        <select
          v-model="form.default_bill_template_id"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">{{ t('utilities.editUtilityModal.billTemplateNone') }}</option>
          <option
            v-for="tpl in billTemplates"
            :key="tpl.id"
            :value="tpl.id"
          >
            {{ t('utilities.editUtilityModal.billTemplateOption', { name: tpl.name, provider: tpl.provider }) }}
          </option>
        </select>
        <p class="text-xs text-ink-faint mt-1">
          {{ t('utilities.editUtilityModal.billTemplateHint') }}
        </p>
      </div>

      <!-- Default Category -->
      <div v-if="form.default_category_id !== undefined">
        <input type="hidden" v-model="form.default_category_id" />
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
          {{ loading ? t('utilities.addUtilityModal.saving') : t('utilities.editUtilityModal.saveChanges') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { currencies, currencyOptionLabel } from '@/utils/currencies'
import { utilitiesAPI, membersAPI, templatesAPI } from '@/api/client'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  utility: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated'])
const { t } = useI18n()
const settingsStore = useSettingsStore()

const isCurrencyLocked = computed(() => !!props.utility.is_currency_locked)

const currencyOptions = computed(() => {
  const base = settingsStore.formatSettings.currency || 'EUR'
  return [
    { value: '', label: t('utilities.addUtilityModal.currencyGlobalOption', { base }) },
    ...currencies.map(c => ({ value: c.code, label: currencyOptionLabel(c.code, t) })),
  ]
})

const loading = ref(false)
const error = ref(null)
const members = ref([])
const billTemplates = ref([])
const splitMemberIds = ref([])

function typeLabel(value) {
  const key = `utilities.utilityTypes.${value}`
  const label = t(key)
  return label === key ? value : label
}

const allTypes = computed(() => [
  { value: 'electricity', label: typeLabel('electricity'), icon: '⚡', metered: true },
  { value: 'gas', label: typeLabel('gas'), icon: '🔥', metered: true },
  { value: 'water', label: typeLabel('water'), icon: '💧', metered: true },
  // waste (TARI) is billed on surface area, not on a meter — see Utility.IsMetered.
  { value: 'waste', label: typeLabel('waste'), icon: '♻️', metered: false },
  { value: 'internet', label: typeLabel('internet'), icon: '🌐', metered: false },
  { value: 'insurance', label: typeLabel('insurance'), icon: '🛡️', metered: false },
  { value: 'affitto', label: typeLabel('affitto'), icon: '🏠', metered: false },
  { value: 'mutuo', label: typeLabel('mutuo'), icon: '🏦', metered: false },
])

const typeInfo = computed(() => {
  return allTypes.value.find(tt => tt.value === form.value.type) || { label: form.value.type, icon: '' }
})

const isMetered = computed(() => {
  const found = allTypes.value.find(tt => tt.value === form.value.type)
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

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  if (dateStr.includes('T')) {
    return dateStr.split('T')[0]
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
    return dateStr
  }
  const d = new Date(dateStr)
  if (!isNaN(d.getTime())) {
    return d.toISOString().split('T')[0]
  }
  return ''
}

const form = ref({
  type: props.utility.type || 'electricity',
  provider: props.utility.provider || '',
  service_code: props.utility.service_code || '',
  customer_code: props.utility.customer_code || '',
  address: props.utility.address || '',
  power_capacity: props.utility.power_capacity || null,
  recurring_amount: props.utility.recurring_amount || null,
  billing_interval: props.utility.billing_interval || 1,
  billing_unit: props.utility.billing_unit || 'month',
  paid_by_member_id: props.utility.paid_by_member_id || null,
  start_date: formatDateForInput(props.utility.start_date),
  end_date: formatDateForInput(props.utility.end_date),
  customer_portal: props.utility.customer_portal || '',
  notes: props.utility.notes || '',
  is_active: props.utility.is_active !== undefined ? props.utility.is_active : true,
  allows_self_reading: props.utility.allows_self_reading !== undefined ? props.utility.allows_self_reading : true,
  comparison_threshold: props.utility.comparison_threshold || 5,
  threshold_per_day: props.utility.threshold_per_day || null,
  default_category_id: props.utility.default_category_id || null,
  default_bill_template_id: props.utility.default_bill_template_id || null,
  split_override: props.utility.split_override || '',
  is_domiciled: props.utility.is_domiciled || false,
  is_installment_based: props.utility.is_installment_based || false,
  currency: props.utility.currency || '',
})

if (props.utility.split_member_ids) {
  try {
    splitMemberIds.value = JSON.parse(props.utility.split_member_ids)
  } catch {
    splitMemberIds.value = []
  }
}

const splitOverrideHint = computed(() => {
  switch (form.value.split_override) {
    case 'no_split': return t('utilities.addUtilityModal.splitOverrideHintNoSplit')
    case 'custom': return t('utilities.addUtilityModal.splitOverrideHintCustom')
    default: return t('utilities.addUtilityModal.splitOverrideHintGlobal')
  }
})

async function fetchMembers() {
  const propertyId = props.utility.property_id
  if (!propertyId) return

  try {
    const { data } = await membersAPI.list(propertyId)
    members.value = data || []
  } catch (err) {
    console.error('Error fetching members:', err)
  }
}

async function fetchBillTemplates() {
  try {
    const { data } = await templatesAPI.listBillTemplates()
    billTemplates.value = (data || []).filter(tpl => tpl.utility_type === props.utility.type)
  } catch (err) {
    console.error('Error fetching bill templates:', err)
  }
}

async function handleSubmit() {
  if (!form.value.provider) {
    error.value = t('utilities.editUtilityModal.providerRequired')
    return
  }

  loading.value = true
  error.value = null

  try {
    const updateData = {
      provider: form.value.provider,
      service_code: form.value.service_code,
      customer_code: form.value.customer_code,
      address: form.value.address,
      power_capacity: form.value.power_capacity ? parseFloat(form.value.power_capacity) : 0,
      recurring_amount: form.value.recurring_amount ? parseFloat(form.value.recurring_amount) : undefined,
      billing_interval: form.value.billing_interval || 1,
      billing_unit: form.value.billing_unit || 'month',
      paid_by_member_id: form.value.paid_by_member_id || undefined,
      start_date: form.value.start_date ? new Date(form.value.start_date).toISOString() : undefined,
      end_date: form.value.end_date ? new Date(form.value.end_date).toISOString() : undefined,
      customer_portal: form.value.customer_portal,
      notes: form.value.notes,
      is_active: form.value.is_active,
      allows_self_reading: form.value.allows_self_reading,
      comparison_threshold: form.value.comparison_threshold,
      threshold_per_day: form.value.threshold_per_day ? parseFloat(form.value.threshold_per_day) : 0,
      default_category_id: form.value.default_category_id || undefined,
      default_bill_template_id: form.value.default_bill_template_id || undefined,
      split_override: form.value.split_override,
      split_member_ids: form.value.split_override === 'custom' ? JSON.stringify(splitMemberIds.value) : '',
      is_domiciled: form.value.is_domiciled,
      is_installment_based: form.value.is_installment_based,
      ...(isCurrencyLocked.value ? {} : { currency: form.value.currency }),
    }

    const { data } = await utilitiesAPI.update(props.utility.id, updateData)
    emit('updated', data)
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('utilities.addUtilityModal.genericError')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchMembers()
  fetchBillTemplates()
})
</script>
