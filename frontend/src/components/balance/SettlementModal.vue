<template>
  <BaseModal :title="balance > 0 ? t('balance.receivePayment') : t('balance.settleUp')" @close="$emit('close')">
    <!-- Balance Info -->
    <div class="bg-blue-50 dark:bg-blue-900/30 rounded-xl p-6 text-center mb-6">
      <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">{{ t('balance.settlement.amountLabel') }}</div>
      <div class="text-4xl font-bold text-blue-600 dark:text-blue-400">
        {{ formatCurrency(Math.abs(balance)) }}
      </div>
      <div class="text-sm text-gray-600 dark:text-gray-400 mt-2">
        {{ balance > 0 ? t('balance.settlement.fromOther', { name: otherMemberName }) : t('balance.settlement.toOther', { name: otherMemberName }) }}
      </div>
    </div>

    <form @submit.prevent="handleSubmit" class="space-y-4">
      <Input
        v-model="form.date"
        :label="t('balance.settlement.dateLabel')"
        type="date"
        required
      />

      <div>
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          {{ t('balance.settlement.methodLabel') }}
        </label>
        <select
          v-model="form.payment_method"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          {{ t('balance.settlement.noteLabel') }}
        </label>
        <textarea
          v-model="form.note"
          rows="2"
          :placeholder="t('balance.settlement.notePlaceholder')"
          autocorrect="off"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
        <p class="text-sm text-yellow-800 dark:text-yellow-200">
          {{ t('balance.settlement.warning') }}
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
import { ref } from 'vue'
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
    required: true
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

const form = ref({
  amount: Math.abs(props.balance),
  date: new Date().toISOString().split('T')[0],
  payment_method: 'bank_transfer',
  note: ''
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

async function handleSubmit() {
  loading.value = true
  error.value = null

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
      from_member_id: props.balance < 0 ? currentMemberId : otherMemberId,
      to_member_id: props.balance < 0 ? otherMemberId : currentMemberId,
      amount: parseFloat(form.value.amount),
      date: form.value.date,
      payment_method: form.value.payment_method,
      note: form.value.note
    }

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
