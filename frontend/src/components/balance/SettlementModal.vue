<template>
  <BaseModal :title="modalTitle" @close="$emit('close')">
    <!-- What is being paid: the running balance, or one long-term debt -->
    <div class="bg-blue-50 dark:bg-blue-900/30 rounded-xl p-6 text-center mb-6">
      <div v-if="debt" class="text-sm font-medium text-ink mb-1 truncate">
        {{ debt.description || t('balance.debts.noDescription') }}
      </div>
      <div class="text-sm text-ink-soft mb-1">
        {{ debt ? t('balance.debts.remainingLabel') : t('balance.settlement.amountLabel') }}
      </div>
      <div class="text-4xl font-bold text-blue-600 dark:text-blue-400">
        {{ formatCurrency(maxAmount) }}
      </div>
      <div class="text-sm text-ink-soft mt-2">
        {{ directionLabel }}
      </div>
    </div>

    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div>
        <div class="flex items-center justify-between mb-1">
          <label class="block text-sm text-ink-soft">
            {{ t('balance.settlement.payAmountLabel') }}
          </label>
          <button
            type="button"
            class="text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline"
            @click="form.amount = maxAmount"
          >
            {{ t('balance.settlement.payAll') }}
          </button>
        </div>
        <Input
          v-model="form.amount"
          type="number"
          step="0.01"
          min="0.01"
          :max="maxAmount"
          inputmode="decimal"
          required
        />
        <p class="text-xs text-ink-faint mt-1">
          {{ t('balance.settlement.payAmountHint', { total: formatCurrency(maxAmount) }) }}
        </p>
      </div>

      <Input
        v-model="form.date"
        :label="t('balance.settlement.dateLabel')"
        type="date"
        required
      />

      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('balance.settlement.methodLabel') }}
        </label>
        <select
          v-model="form.payment_method"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="bank_transfer">{{ t('balance.settlement.methods.bank_transfer') }}</option>
          <option value="cash">{{ t('balance.settlement.methods.cash') }}</option>
          <option value="satispay">{{ t('balance.settlement.methods.satispay') }}</option>
          <option value="paypal">{{ t('balance.settlement.methods.paypal') }}</option>
          <option value="revolut">{{ t('balance.settlement.methods.revolut') }}</option>
        </select>
      </div>

      <div>
        <label class="block text-sm text-ink-soft mb-1">
          {{ t('balance.settlement.noteLabel') }}
        </label>
        <textarea
          v-model="form.note"
          rows="2"
          :placeholder="t('balance.settlement.notePlaceholder')"
          autocorrect="off"
          class="w-full px-3 py-3 border border-line rounded-lg
                 bg-surface text-ink text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
        <p class="text-sm text-yellow-800 dark:text-yellow-200">
          {{ debt ? t('balance.debts.repayWarning') : t('balance.settlement.warning') }}
        </p>
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          {{ t('balance.settlement.cancel') }}
        </Button>
        <Button type="submit" variant="success" :disabled="loading" class="flex-1">
          {{ loading ? t('balance.settlement.saving') : t('balance.settlement.submit') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()

const props = defineProps({
  balance: {
    type: Number,
    default: 0
  },
  // When set, the payment targets this long-term debt instead of the running
  // balance: the amount is capped by its remainder and the direction comes
  // from the debt itself, not from the sign of `balance`.
  debt: {
    type: Object,
    default: null
  },
  otherMemberName: {
    type: String,
    default: 'Partner'
  },
  otherMemberId: {
    type: Number,
    default: null
  },
  currentMemberId: {
    type: Number,
    default: null
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

const maxAmount = computed(() =>
  props.debt ? (props.debt.remaining_amount ?? 0) : Math.abs(props.balance)
)

// Who pays whom: for a debt it's fixed by the debt's own direction, otherwise
// by the sign of the running balance.
const iAmPaying = computed(() => (props.debt ? props.debt.i_owe : props.balance < 0))

const modalTitle = computed(() => {
  if (props.debt) return t('balance.debts.repayTitle')
  return props.balance > 0 ? t('balance.receivePayment') : t('balance.settleUp')
})

const directionLabel = computed(() =>
  iAmPaying.value
    ? t('balance.settlement.toOther', { name: props.otherMemberName })
    : t('balance.settlement.fromOther', { name: props.otherMemberName })
)

const form = ref({
  amount: maxAmount.value,
  date: new Date().toISOString().split('T')[0],
  payment_method: 'bank_transfer',
  note: ''
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

async function handleSubmit() {
  error.value = null

  const amount = parseFloat(form.value.amount)
  if (!amount || amount <= 0) {
    error.value = t('balance.settlement.invalidAmount')
    return
  }
  if (amount > maxAmount.value + 0.005) {
    error.value = t('balance.settlement.amountExceedsBalance')
    return
  }

  loading.value = true

  try {
    const currentMemberId = props.currentMemberId
    const otherMemberId = props.otherMemberId

    if (!currentMemberId || !otherMemberId) {
      error.value = t('balance.settlement.missingMembers')
      loading.value = false
      return
    }

    const settlementData = {
      property_id: props.propertyId,
      from_member_id: iAmPaying.value ? currentMemberId : otherMemberId,
      to_member_id: iAmPaying.value ? otherMemberId : currentMemberId,
      amount,
      date: form.value.date,
      payment_method: form.value.payment_method,
      note: form.value.note
    }
    if (props.debt) settlementData.target_expense_id = props.debt.expense_id

    await balanceStore.createSettlement(settlementData)
    window.$toast?.success(t('balance.settlement.successToast'))
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('balance.settlement.genericError')
  } finally {
    loading.value = false
  }
}
</script>
