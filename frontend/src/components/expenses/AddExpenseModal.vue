<template>
  <BaseModal :title="t('expenses.modal.addTitle')" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Quick Templates -->
      <div v-if="expenseTemplates.length > 0">
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.templateLabel') }}
        </label>
        <select
          v-model.number="selectedTemplateId"
          @change="onTemplateSelect"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">{{ t('expenses.modal.templateNone') }}</option>
          <option
            v-for="tpl in expenseTemplates"
            :key="tpl.id"
            :value="tpl.id"
          >
            {{ tpl.icon || tpl.category?.icon || '' }} {{ tpl.name }}{{ tpl.amount ? ` (${formatCurrency(tpl.amount)})` : '' }}
          </option>
        </select>
      </div>

      <!-- Amount + Currency -->
      <div>
        <label class="block text-sm text-ink-soft mb-1">{{ t('expenses.modal.amountLabel') }}</label>
        <div class="flex gap-2">
          <div class="flex-1">
            <Input
              v-model="form.amount"
              type="number"
              step="0.01"
              min="0.01"
              placeholder="0.00"
              inputmode="decimal"
              required
            />
          </div>
          <select
            v-model="selectedCurrency"
            class="w-24 px-2 py-3 border border-line rounded-lg
                   bg-surface text-ink text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option v-for="c in allCurrencies" :key="c.code" :value="c.code">
              {{ c.code }}
            </option>
          </select>
        </div>

        <!-- Conversion preview -->
        <div v-if="isForeignCurrency && form.amount" class="mt-1.5">
          <div v-if="rateLoading" class="text-xs text-ink-faint">
            {{ t('expenses.modal.rateLoading') }}
          </div>
          <div v-else-if="convertedAmount != null" class="text-xs text-green-600 dark:text-green-400">
            {{ formatOriginal(form.amount, selectedCurrency) }} ≈ {{ formatCurrency(convertedAmount) }}
            <span class="text-ink-faint">{{ t('expenses.modal.rateInfo', { rate: exchangeRate?.toFixed(6) }) }}</span>
          </div>
          <div v-else-if="rateError" class="space-y-1">
            <p class="text-xs text-amber-600 dark:text-amber-400">
              {{ t('expenses.modal.rateUnavailable') }}
            </p>
            <div class="flex items-center gap-2">
              <span class="text-xs text-ink-muted">1 {{ selectedCurrency }} =</span>
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
              <span class="text-xs text-ink-muted">{{ settingsStore.currency }}</span>
            </div>
            <div v-if="manualConvertedAmount != null" class="text-xs text-green-600 dark:text-green-400">
              {{ formatOriginal(form.amount, selectedCurrency) }} ≈ {{ formatCurrency(manualConvertedAmount) }}
            </div>
          </div>
        </div>
      </div>

      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.descriptionLabel') }}
        </label>
        <textarea
          v-model="form.description"
          rows="2"
          required
          :placeholder="t('expenses.modal.descriptionPlaceholder')"
          autocorrect="off"
          autocapitalize="off"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.categoryLabel') }}
        </label>
        <select
          v-model.number="form.category_id"
          @change="form.subcategory_id = null"
          required
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="" disabled>{{ t('expenses.modal.categoryPlaceholder') }}</option>
          <option
            v-for="cat in categories"
            :key="cat.id"
            :value="cat.id"
          >
            {{ cat.icon }} {{ categoryLabel(cat) }}
          </option>
        </select>
      </div>

      <!-- Subcategory (shown only when selected category has subcategories) -->
      <div v-if="selectedCategorySubcategories.length > 0">
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.subcategoryLabel') }}
        </label>
        <select
          v-model.number="form.subcategory_id"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">{{ t('expenses.modal.subcategoryNone') }}</option>
          <option
            v-for="sub in selectedCategorySubcategories"
            :key="sub.id"
            :value="sub.id"
          >
            {{ categoryLabel(sub) }}
          </option>
        </select>
      </div>

      <Input
        v-model="form.date"
        :label="t('expenses.modal.dateLabel')"
        type="date"
        required
      />

      <!-- Project (Optional) -->
      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.projectLabel') }}
        </label>
        <select
          v-model.number="form.project_id"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option :value="null">{{ t('expenses.modal.projectNone') }}</option>
          <option
            v-for="proj in activeProjects"
            :key="proj.id"
            :value="proj.id"
          >
            {{ proj.icon }} {{ proj.name }}
          </option>
        </select>
      </div>

      <!-- Sezione Split -->
      <div v-if="hasMultipleUsers" class="border-t border-line pt-4 space-y-3">
        <div class="flex items-center gap-3">
          <input
            type="checkbox"
            id="split-checkbox"
            v-model="form.is_split"
            class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
          />
          <label for="split-checkbox" class="text-sm font-medium text-ink cursor-pointer">
            {{ t('expenses.modal.splitToggle') }}
          </label>
        </div>

        <div v-if="form.is_split" class="space-y-4 pl-2 border-l-2 border-blue-200 dark:border-blue-800 ml-2">
          <!-- Chi ha pagato -->
          <div class="pl-4">
            <label class="block text-sm font-medium text-ink-soft mb-2">
              {{ t('expenses.modal.paidByLabel') }}
            </label>
            <div class="space-y-2">
              <label
                v-for="user in householdUsers"
                :key="user.id"
                class="flex items-center gap-3 p-2 rounded-lg hover:bg-surface-2 cursor-pointer"
              >
                <input
                  type="radio"
                  :value="user.id"
                  v-model="form.paid_by_member_id"
                  class="w-4 h-4 text-blue-600 border-line focus:ring-blue-500"
                />
                <span class="text-ink">{{ user.name }}</span>
                <span v-if="user.user_id === authStore.user?.id" class="text-xs text-ink-muted">{{ t('expenses.modal.paidByYou') }}</span>
              </label>
            </div>
          </div>

          <!-- Con chi dividere -->
          <div class="pl-4">
            <label class="block text-sm font-medium text-ink-soft mb-2">
              {{ t('expenses.modal.splitWithLabel') }}
            </label>
            <div class="space-y-2">
              <label
                v-for="user in otherUsers"
                :key="user.id"
                class="flex items-center gap-3 p-2 rounded-lg hover:bg-surface-2 cursor-pointer"
              >
                <input
                  type="checkbox"
                  :value="user.id"
                  v-model="form.split_with_member_ids"
                  class="w-4 h-4 text-blue-600 rounded border-line focus:ring-blue-500"
                />
                <span class="text-ink">{{ user.name }}</span>
              </label>
            </div>
          </div>

          <!-- Riepilogo divisione -->
          <div
            v-if="form.split_with_member_ids.length > 0 && form.amount"
            class="ml-4 bg-blue-50 dark:bg-blue-900/30 p-4 rounded-lg"
          >
            <div class="text-sm space-y-1">
              <div class="font-medium text-ink mb-2">{{ t('expenses.modal.summaryTitle') }}</div>
              <div class="text-ink-soft">
                {{ t('expenses.modal.summaryTotal') }} <span class="font-medium">
                  <template v-if="isForeignCurrency">{{ formatOriginal(form.amount, selectedCurrency) }}</template>
                  <template v-else>{{ formatCurrency(form.amount) }}</template>
                </span>
                <span v-if="isForeignCurrency && finalConvertedAmount != null" class="text-xs text-green-600 dark:text-green-400 ml-1">
                  ≈ {{ formatCurrency(finalConvertedAmount) }}
                </span>
              </div>
              <div class="text-ink-soft">
                {{ t('expenses.modal.summaryDividedBetween', { n: totalPeople }) }}
              </div>
              <div class="text-lg font-bold text-blue-600 dark:text-blue-400 mt-2">
                {{ t('expenses.modal.summaryEach', { amount: formatCurrency(splitAmount) }) }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <!-- Save as template -->
      <div v-if="form.category_id && form.description" class="border-t border-line pt-3">
        <button
          type="button"
          @click="saveAsTemplate"
          class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
        >
          {{ t('expenses.modal.saveAsTemplate') }}
        </button>
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          {{ t('expenses.modal.cancel') }}
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? t('expenses.modal.saving') : t('expenses.modal.save') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import apiClient, { categoriesAPI, projectsAPI, expenseTemplatesAPI, exchangeAPI } from '@/api/client'
import { currencies as allCurrencies } from '@/utils/currencies'
import { categoryLabel } from '@/utils/categoryLabel'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()

const props = defineProps({
  projectId: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['close', 'created'])
const expensesStore = useExpensesStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const loading = ref(false)
const error = ref(null)
const userSettings = ref(null)
const categories = ref([])
const activeProjects = ref([])
const expenseTemplates = ref([])
const selectedTemplateId = ref(null)

const householdUsers = ref([])
const currentPropertyId = ref(null)

// Currency conversion
const selectedCurrency = ref(settingsStore.currency || 'EUR')
const exchangeRate = ref(null)
const convertedAmount = ref(null)
const rateLoading = ref(false)
const rateError = ref(false)
const manualRate = ref(null)
let fetchRateTimer = null

onUnmounted(() => {
  clearTimeout(fetchRateTimer)
})

const isForeignCurrency = computed(() => selectedCurrency.value !== settingsStore.currency)

const manualConvertedAmount = computed(() => {
  if (!manualRate.value || !form.value.amount) return null
  return parseFloat(form.value.amount) * manualRate.value
})

// The final converted amount (auto or manual)
const finalConvertedAmount = computed(() => {
  if (!isForeignCurrency.value) return null
  if (convertedAmount.value != null) return convertedAmount.value
  if (manualConvertedAmount.value != null) return manualConvertedAmount.value
  return null
})

const form = ref({
  amount: null,
  description: '',
  category_id: 1,
  subcategory_id: null,
  date: new Date().toISOString().split('T')[0],
  paid_by_member_id: null,
  is_split: false,
  split_with_member_ids: [],
  project_id: props.projectId,
  property_id: null
})

const hasMultipleUsers = computed(() => householdUsers.value.length > 1)

const selectedCategorySubcategories = computed(() => {
  if (!form.value.category_id) return []
  const cat = categories.value.find(c => c.id === form.value.category_id)
  return cat?.subcategories || []
})

const otherUsers = computed(() => {
  return householdUsers.value.filter(u => u.id !== form.value.paid_by_member_id)
})

// Effective split members: exclude the payer and any stale/deleted member IDs
const effectiveSplitMembers = computed(() => {
  const validIds = new Set(otherUsers.value.map(u => u.id))
  return form.value.split_with_member_ids.filter(id => validIds.has(id))
})

const totalPeople = computed(() => {
  return effectiveSplitMembers.value.length + 1
})

const splitAmount = computed(() => {
  if (!form.value.amount || totalPeople.value === 0) return 0
  const base = isForeignCurrency.value && finalConvertedAmount.value != null
    ? finalConvertedAmount.value
    : parseFloat(form.value.amount)
  return base / totalPeople.value
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatOriginal(value, currency) {
  return _formatCurrency(value, { ...settingsStore.formatSettings, currency })
}

async function fetchExchangeRate() {
  const amount = parseFloat(form.value.amount)
  if (!isForeignCurrency.value || !amount || amount <= 0) {
    exchangeRate.value = null
    convertedAmount.value = null
    return
  }
  rateLoading.value = true
  rateError.value = false
  try {
    const { data } = await exchangeAPI.getRate(selectedCurrency.value, settingsStore.currency, amount)
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

// Debounced watcher for currency/amount changes
watch([selectedCurrency, () => form.value.amount], () => {
  clearTimeout(fetchRateTimer)
  if (!isForeignCurrency.value) {
    exchangeRate.value = null
    convertedAmount.value = null
    manualRate.value = null
    rateError.value = false
    return
  }
  fetchRateTimer = setTimeout(fetchExchangeRate, 500)
})

async function fetchCategories() {
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data
    if (!form.value.category_id && data.length > 0) {
      form.value.category_id = data[0].id
    }
  } catch (err) {
    console.error('Error fetching categories:', err)
  }
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
      form.value.property_id = currentProp.id
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function fetchUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    userSettings.value = data

    if (data.default_split_with_member_ids) {
      try {
        const defaultMembers = JSON.parse(data.default_split_with_member_ids)
        if (defaultMembers && defaultMembers.length > 0) {
          // Filter out the payer and any stale/deleted member IDs
          const validOtherIds = new Set(
            householdUsers.value
              .filter(u => u.id !== form.value.paid_by_member_id)
              .map(u => u.id)
          )
          const filtered = defaultMembers.filter(id => validOtherIds.has(id))
          if (filtered.length > 0) {
            form.value.is_split = true
            form.value.split_with_member_ids = filtered
          }
        }
      } catch {
        console.log('No default split members configured')
      }
    }
  } catch {
    console.log('Using default settings')
  }
}

async function fetchActiveProjects() {
  try {
    const { data } = await projectsAPI.list({ status: 'active' })
    activeProjects.value = data || []
  } catch (err) {
    console.error('Error fetching projects:', err)
  }
}

async function fetchHouseholdUsers() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/members`)
    householdUsers.value = data

    const currentUserId = authStore.user?.id
    const myMember = data.find(m => m.user_id === currentUserId)
    if (myMember) {
      form.value.paid_by_member_id = myMember.id
    }
  } catch (err) {
    console.log('Error fetching household members:', err)
    householdUsers.value = []
  }
}

watch(() => form.value.is_split, (newVal) => {
  if (!newVal) {
    form.value.split_with_member_ids = []
  }
})

watch(() => form.value.paid_by_member_id, (newPayerId) => {
  // Remove the new payer from split members (they're already counted)
  form.value.split_with_member_ids = form.value.split_with_member_ids.filter(id => id !== newPayerId)
})

watch(() => form.value.project_id, (newProjectId) => {
  if (!newProjectId) return
  const project = activeProjects.value.find(p => p.id === newProjectId)
  if (project?.shared_with?.length > 0) {
    const sharedUserIds = project.shared_with.map(u => u.id)
    const matchingMemberIds = householdUsers.value
      .filter(m => m.user_id && sharedUserIds.includes(m.user_id) && m.id !== form.value.paid_by_member_id)
      .map(m => m.id)
    if (matchingMemberIds.length > 0) {
      form.value.is_split = true
      form.value.split_with_member_ids = matchingMemberIds
    }
  }
})

async function fetchExpenseTemplates() {
  try {
    const { data } = await expenseTemplatesAPI.list()
    expenseTemplates.value = data || []
  } catch (err) {
    console.error('Error fetching expense templates:', err)
  }
}

function onTemplateSelect() {
  if (!selectedTemplateId.value) return
  const tpl = expenseTemplates.value.find(t => t.id === selectedTemplateId.value)
  if (!tpl) return
  if (tpl.amount) form.value.amount = tpl.amount
  form.value.description = tpl.description || tpl.name
  form.value.category_id = tpl.category_id
  form.value.subcategory_id = tpl.subcategory_id || null
  if (tpl.project_id && !props.projectId) form.value.project_id = tpl.project_id
  // Apply template currency
  selectedCurrency.value = tpl.currency || settingsStore.currency
}

async function saveAsTemplate() {
  const cat = categories.value.find(c => c.id === form.value.category_id)
  try {
    const { data } = await expenseTemplatesAPI.create({
      name: form.value.description,
      icon: cat?.icon || '',
      amount: parseFloat(form.value.amount) || 0,
      currency: isForeignCurrency.value ? selectedCurrency.value : '',
      description: form.value.description,
      category_id: form.value.category_id,
      subcategory_id: form.value.subcategory_id || undefined,
      project_id: form.value.project_id || undefined,
    })
    expenseTemplates.value.push(data)
    window.$toast?.success(t('expenses.modal.templateSavedToast'))
  } catch {
    window.$toast?.error(t('expenses.modal.templateSaveErrorToast'))
  }
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    const isForeign = isForeignCurrency.value
    const converted = finalConvertedAmount.value

    if (isForeign && converted == null) {
      error.value = t('expenses.modal.rateMissingError')
      loading.value = false
      return
    }

    const expenseData = {
      amount: isForeign ? converted : parseFloat(form.value.amount),
      original_amount: isForeign ? parseFloat(form.value.amount) : undefined,
      original_currency: isForeign ? selectedCurrency.value : undefined,
      description: form.value.description,
      category_id: form.value.category_id,
      subcategory_id: form.value.subcategory_id || undefined,
      date: form.value.date,
      property_id: form.value.property_id,
      project_id: form.value.project_id,
      paid_by_member_id: form.value.paid_by_member_id,
      is_split: form.value.is_split,
      split_with_member_ids: form.value.is_split
        ? effectiveSplitMembers.value
        : []
    }

    await expensesStore.createExpense(expenseData)
    window.$toast?.success(t('expenses.modal.createdToast'))
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('expenses.modal.genericSaveError')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  fetchCategories()
  fetchExpenseTemplates()
  await fetchCurrentProperty()
  await fetchHouseholdUsers()
  await fetchUserSettings()
  fetchActiveProjects()
})
</script>
