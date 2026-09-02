<template>
  <BaseModal :title="t('expenses.modal.editTitle')" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Settled expense notice -->
      <div v-if="isSettled" class="flex items-start gap-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
        <svg class="w-4 h-4 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
        </svg>
        <p class="text-sm text-amber-800 dark:text-amber-200">
          {{ t('expenses.modal.settledNotice') }}
        </p>
      </div>

      <div>
        <Input
          v-model="form.amount"
          :label="t('expenses.modal.amountLabel')"
          type="number"
          step="0.01"
          min="0.01"
          placeholder="0.00"
          inputmode="decimal"
          :required="!amountLocked"
          :disabled="amountLocked"
        />
        <p v-if="amountLocked && !isSettled" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
          {{ t('expenses.modal.amountLockedNotice') }}
        </p>
        <p v-if="expense.original_currency" class="mt-1 text-xs text-ink-muted">
          {{ t('expenses.modal.originalAmountInfo', { amount: formatOriginal(expense.original_amount, expense.original_currency) }) }}
          <span v-if="expense.original_amount && expense.amount">
            {{ t('expenses.modal.originalRateInfo', { rate: (expense.amount / expense.original_amount).toFixed(6) }) }}
          </span>
        </p>
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

      <!-- Subcategory selector -->
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
        :required="!isSettled"
        :disabled="isSettled"
      />

      <!-- Project (Optional) -->
      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('expenses.modal.projectLabel') }}
        </label>
        <select
          v-model.number="form.project_id"
          :disabled="isSettled"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500
                 disabled:opacity-50 disabled:cursor-not-allowed"
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

      <!-- Note about split -->
      <div v-if="expense.is_split && !isSettled" class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
        <p class="text-sm text-yellow-800 dark:text-yellow-200">
          {{ t('expenses.modal.splitEditNotice') }}
        </p>
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          {{ t('expenses.modal.cancel') }}
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? t('expenses.modal.saving') : t('expenses.modal.saveChanges') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExpensesStore } from '@/stores/expenses'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import { categoryLabel } from '@/utils/categoryLabel'
import { categoriesAPI, projectsAPI } from '@/api/client'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()

const props = defineProps({
  expense: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated'])
const expensesStore = useExpensesStore()
const settingsStore = useSettingsStore()

function formatOriginal(value, currency) {
  return _formatCurrency(value, { ...settingsStore.formatSettings, currency })
}

const loading = ref(false)
const error = ref(null)
const categories = ref([])
const activeProjects = ref([])

const form = ref({
  amount: null,
  description: '',
  category_id: 1,
  subcategory_id: null,
  date: '',
  project_id: null
})

const selectedCategorySubcategories = computed(() => {
  if (!form.value.category_id) return []
  const cat = categories.value.find(c => c.id === form.value.category_id)
  return cat?.subcategories || []
})

const isSettled = computed(() => {
  if (!props.expense.is_split) return false
  const splits = props.expense.splits
  return Array.isArray(splits) && splits.length > 0 && splits.every(s => s.is_settled)
})

const amountLocked = computed(() => {
  if (isSettled.value) return true
  if (!props.expense.is_split) return false
  const splits = props.expense.splits
  if (!Array.isArray(splits) || splits.length === 0) return false
  const payerId = props.expense.paid_by_member_id
  return splits.some(s => s.member_id !== payerId && s.is_settled)
})

async function fetchCategories() {
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data
  } catch (err) {
    console.error('Error fetching categories:', err)
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

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    const expenseData = {
      description: form.value.description,
      category_id: form.value.category_id,
      subcategory_id: form.value.subcategory_id || undefined,
    }

    if (!isSettled.value) {
      if (!amountLocked.value) {
        expenseData.amount = parseFloat(form.value.amount)
      }
      expenseData.date = form.value.date
      expenseData.project_id = form.value.project_id
    }

    await expensesStore.updateExpense(props.expense.id, expenseData)
    window.$toast?.success(t('expenses.modal.updatedToast'))
    emit('updated')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('expenses.modal.genericSaveError')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchCategories()
  fetchActiveProjects()
  form.value = {
    amount: props.expense.amount,
    description: props.expense.description,
    category_id: props.expense.category_id || props.expense.category?.id || 1,
    subcategory_id: props.expense.subcategory_id || null,
    date: props.expense.date ? props.expense.date.split('T')[0] : '',
    project_id: props.expense.project_id || null
  }
})
</script>
