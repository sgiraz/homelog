<template>
  <BaseModal :title="t('balance.compensate.title')" @close="$emit('close')">
    <!-- The credit being spent -->
    <div class="bg-blue-50 dark:bg-blue-900/30 rounded-xl p-5 text-center mb-5">
      <div class="text-sm font-medium text-ink mb-1 truncate">
        {{ split.description || t('balance.unsettled.noDescription') }}
      </div>
      <div class="text-sm text-ink-soft mb-1">{{ t('balance.compensate.creditLabel') }}</div>
      <div class="text-3xl font-bold text-blue-600 dark:text-blue-400">
        {{ formatCurrency(creditAmount) }}
      </div>
    </div>

    <p class="text-sm text-ink-soft mb-3">{{ t('balance.compensate.pickDebt') }}</p>

    <div class="space-y-2">
      <label
        v-for="debt in debts"
        :key="debt.expense_id"
        :class="[
          'flex items-center gap-3 p-3 border rounded-lg cursor-pointer transition-colors',
          selectedId === debt.expense_id ? 'border-accent bg-surface-2' : 'border-line hover:bg-surface-2'
        ]"
      >
        <input
          type="radio"
          :value="debt.expense_id"
          v-model="selectedId"
          class="w-5 h-5 text-blue-600 border-line focus:ring-blue-500"
        />
        <div class="min-w-0 flex-1">
          <div class="text-sm font-medium text-ink truncate">
            {{ debt.description || t('balance.debts.noDescription') }}
          </div>
          <div class="text-xs text-ink-muted mt-0.5">
            {{ t('balance.debts.remainingLabel') }}: {{ formatCurrency(debt.remaining_amount) }}
          </div>
        </div>
      </label>
    </div>

    <div v-if="selectedDebt" class="mt-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
      <p class="text-sm text-yellow-800 dark:text-yellow-200">
        {{ t('balance.compensate.resultHint', {
          amount: formatCurrency(appliedAmount),
          debt: selectedDebt.description || t('balance.debts.noDescription')
        }) }}
      </p>
    </div>

    <div v-if="error" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
      {{ error }}
    </div>

    <div class="flex gap-3 pt-5">
      <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
        {{ t('balance.settlement.cancel') }}
      </Button>
      <Button type="button" :disabled="loading || !selectedDebt" @click="handleSubmit" class="flex-1">
        {{ loading ? t('balance.settlement.saving') : t('balance.compensate.submit') }}
      </Button>
    </div>
  </BaseModal>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import BaseModal from '@/components/common/BaseModal.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()

const props = defineProps({
  // The unsettled share the counterpart owes you, used here as payment.
  split: {
    type: Object,
    required: true
  },
  // Long-term debts you owe, the only ones this credit can pay down.
  debts: {
    type: Array,
    default: () => []
  },
  propertyId: {
    type: Number,
    default: 1
  }
})

const emit = defineEmits(['close', 'created'])

const balanceStore = useBalanceStore()
const settingsStore = useSettingsStore()

const loading = ref(false)
const error = ref(null)
const selectedId = ref(props.debts.length === 1 ? props.debts[0].expense_id : null)

const creditAmount = computed(() => props.split.remaining_amount ?? props.split.amount ?? 0)
const selectedDebt = computed(() => props.debts.find(d => d.expense_id === selectedId.value) || null)

// Whichever runs out first caps the transfer: the credit, or what's left of the debt.
const appliedAmount = computed(() => {
  if (!selectedDebt.value) return 0
  return Math.min(creditAmount.value, selectedDebt.value.remaining_amount ?? 0)
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

async function handleSubmit() {
  if (!selectedDebt.value) return
  loading.value = true
  error.value = null

  try {
    await balanceStore.createCompensation({
      property_id: props.propertyId,
      source_split_id: props.split.split_id,
      target_expense_id: selectedDebt.value.expense_id
    })
    window.$toast?.success(t('balance.compensate.successToast'))
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('balance.settlement.genericError')
  } finally {
    loading.value = false
  }
}
</script>
