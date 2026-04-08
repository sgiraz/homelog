<template>
  <BaseModal title="Modifica Servizio" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Tipo Servizio (read-only badge) -->
      <div>
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">
          Tipo
        </label>
        <div class="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
          <span class="text-2xl">{{ typeInfo.icon }}</span>
          <span class="text-sm font-medium text-gray-900 dark:text-white">{{ typeInfo.label }}</span>
        </div>
      </div>

      <!-- Attivo / Disattivo toggle -->
      <div class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <button
          type="button"
          @click="form.is_active = !form.is_active"
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
            form.is_active ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
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
          <span class="text-sm font-medium text-gray-900 dark:text-white">
            {{ form.is_active ? 'Attivo' : 'Disattivo' }}
          </span>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ form.is_active ? 'Il servizio è attualmente attivo' : 'Il servizio è stato disattivato' }}
          </p>
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

      <!-- End Date -->
      <Input
        v-model="form.end_date"
        label="Data fine contratto"
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
          id="edit-allows-self-reading"
          v-model="form.allows_self_reading"
          class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
        />
        <div>
          <label for="edit-allows-self-reading" class="text-sm font-medium text-gray-900 dark:text-white cursor-pointer">
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
            <label for="edit-comparison-threshold" class="text-sm font-medium text-gray-900 dark:text-white">
              Soglia confronto letture
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Differenza per segnalare anomalie
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
              class="w-16 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded
                     bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                     focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ consumptionUnitLabel }}</span>
          </div>
        </div>
      </div>

      <!-- Threshold per Day (metered only) -->
      <div v-if="isMetered && form.type !== 'waste'" class="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <div class="flex items-center justify-between">
          <div>
            <label for="edit-threshold-per-day" class="text-sm font-medium text-gray-900 dark:text-white">
              Soglia consumo giornaliero
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Consumo massimo atteso al giorno
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
              class="w-20 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded
                     bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                     focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ consumptionUnitLabel }}/g</span>
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
              :id="'split-member-' + member.id"
              :value="member.id"
              v-model="splitMemberIds"
              class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
            />
            <label :for="'split-member-' + member.id" class="text-sm text-gray-900 dark:text-white cursor-pointer">
              {{ member.name }}{{ member.role ? ` (${member.role})` : '' }}
            </label>
          </div>
        </div>
      </div>

      <!-- Default Bill Template -->
      <div v-if="billTemplates.length > 0">
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          Template bolletta predefinito
        </label>
        <select
          v-model="form.default_bill_template_id"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">Nessun template (auto)</option>
          <option
            v-for="tpl in billTemplates"
            :key="tpl.id"
            :value="tpl.id"
          >
            {{ tpl.name }} — {{ tpl.provider }}
          </option>
        </select>
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
          Usato per estrarre automaticamente i dati dalle bollette PDF
        </p>
      </div>

      <!-- Default Category -->
      <div v-if="form.default_category_id !== undefined">
        <input type="hidden" v-model="form.default_category_id" />
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
          {{ loading ? 'Salvataggio...' : 'Salva modifiche' }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
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

const loading = ref(false)
const error = ref(null)
const members = ref([])
const billTemplates = ref([])
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

const allTypes = [...meteredTypes, ...fixedTypes]

const typeInfo = computed(() => {
  return allTypes.find(t => t.value === form.value.type) || { label: form.value.type, icon: '', iconClass: '' }
})

const isMetered = computed(() => {
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

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  // Handle ISO timestamps like "2025-01-15T00:00:00Z" -> "2025-01-15"
  if (dateStr.includes('T')) {
    return dateStr.split('T')[0]
  }
  // Already YYYY-MM-DD
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
    return dateStr
  }
  // Try to parse as date
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
})

// Initialize split member IDs from utility
if (props.utility.split_member_ids) {
  try {
    splitMemberIds.value = JSON.parse(props.utility.split_member_ids)
  } catch {
    splitMemberIds.value = []
  }
}

const splitOverrideHint = computed(() => {
  switch (form.value.split_override) {
    case 'no_split': return 'Le spese di questo servizio non verranno mai divise'
    case 'custom': return 'Le spese verranno divise con i membri selezionati sotto'
    default: return 'Segue le impostazioni di divisione della famiglia'
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
    // Filter templates for this utility type
    billTemplates.value = (data || []).filter(t => t.utility_type === props.utility.type)
  } catch (err) {
    console.error('Error fetching bill templates:', err)
  }
}

async function handleSubmit() {
  if (!form.value.provider) {
    error.value = 'Il fornitore è obbligatorio'
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
    }

    const { data } = await utilitiesAPI.update(props.utility.id, updateData)
    emit('updated', data)
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchMembers()
  fetchBillTemplates()
})
</script>
