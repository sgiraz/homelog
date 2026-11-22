<template>
  <div
    v-if="balanceStore.counterparts.length > 1"
    class="flex gap-1.5 overflow-x-auto pb-1"
    role="group"
    :aria-label="t('balance.counterpartsLabel')"
  >
    <button
      v-for="cp in balanceStore.counterparts"
      :key="cp.member_id"
      type="button"
      :aria-pressed="cp.member_id === balanceStore.otherMemberId"
      :class="[
        'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap',
        cp.member_id === balanceStore.otherMemberId
          ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
          : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
      ]"
      @click="select(cp.member_id)"
    >
      <span>{{ cp.name }}</span>
      <span
        v-if="cp.balance !== 0"
        :class="cp.balance > 0 ? 'text-positive-soft' : 'text-danger-soft'"
      >{{ cp.balance > 0 ? '+' : '' }}{{ formatCurrency(cp.balance) }}</span>
    </button>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'

const props = defineProps({
  householdId: { type: Number, default: null }
})

const { t } = useI18n()
const balanceStore = useBalanceStore()
const settingsStore = useSettingsStore()

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function select(memberId) {
  if (!props.householdId || memberId === balanceStore.otherMemberId) return
  balanceStore.selectCounterpart(props.householdId, memberId)
}
</script>
