<template>
  <div class="space-y-6">
    <!-- Loading State -->
    <div v-if="balanceStore.loading" class="text-center py-12">
      <svg class="animate-spin h-12 w-12 mx-auto text-blue-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
      <p class="mt-4 text-ink-soft">{{ t('balance.loading') }}</p>
    </div>

    <template v-else>
      <!-- Balance Card -->
      <Card class="p-4 sm:p-8">
        <div class="text-center">
          <div class="text-sm text-ink-soft mb-2">
            {{ t('balance.balanceWith', { name: balanceStore.otherMemberName || t('balance.partnerFallback') }) }}
          </div>
          <div :class="[
            'text-3xl sm:text-5xl font-bold mb-3',
            balanceStore.balance > 0 ? 'text-green-600 dark:text-green-400' :
            balanceStore.balance < 0 ? 'text-red-600 dark:text-red-400' :
            'text-ink-soft'
          ]">
            {{ balanceStore.balance > 0 ? '+' : '' }}{{ formatCurrency(balanceStore.balance) }}
          </div>
          <div class="text-lg text-ink-soft">
            {{ balanceMessage }}
          </div>

          <div class="flex justify-center gap-4 mt-6">
            <Button
              v-if="balanceStore.balance !== 0"
              @click="showSettlementModal = true"
              :variant="balanceStore.balance > 0 ? 'success' : 'primary'"
            >
              <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
              {{ balanceStore.balance > 0 ? t('balance.receivePayment') : t('balance.settleUp') }}
            </Button>
            <Button v-else variant="secondary" disabled>
              <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              {{ t('balance.evenBadge') }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- Stats Cards -->
      <div class="grid grid-cols-2 gap-4">
        <Card class="p-4 sm:p-6 text-center">
          <div class="text-sm text-ink-soft mb-1">{{ t('balance.stats.unsettledCount') }}</div>
          <div class="text-2xl font-bold text-amber-600 dark:text-amber-400">
            {{ balanceStore.unsettledSplits.length }}
          </div>
        </Card>
        <Card class="p-4 sm:p-6 text-center">
          <div class="text-sm text-ink-soft mb-1">{{ t('balance.stats.totalSettled') }}</div>
          <div class="text-2xl font-bold text-green-600 dark:text-green-400">
            {{ formatCurrency(totalSettled) }}
          </div>
        </Card>
      </div>

      <!-- Spese da saldare -->
      <Card v-if="balanceStore.unsettledSplits.length > 0" class="p-4 sm:p-6">
        <button
          @click="unsettledOpen = !unsettledOpen"
          :aria-expanded="unsettledOpen"
          aria-controls="unsettled-splits-list"
          class="w-full flex items-center justify-between text-left"
        >
          <h3 class="text-lg font-semibold text-ink flex items-center gap-2">
            {{ t('balance.unsettled.title') }}
            <span class="text-sm font-normal px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-300">
              {{ balanceStore.unsettledSplits.length }}
            </span>
          </h3>
          <svg
            :class="['w-5 h-5 text-gray-400 transition-transform', unsettledOpen ? 'rotate-180' : '']"
            fill="none" stroke="currentColor" viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <Transition name="filter-expand">
          <div v-if="unsettledOpen" id="unsettled-splits-list" class="mt-4 space-y-3">
            <!-- Totale da saldare -->
            <div class="text-sm text-ink-soft flex justify-between border-b border-line pb-2">
              <span>{{ t('balance.unsettled.totalLabel') }}</span>
              <span class="font-semibold text-amber-600 dark:text-amber-400">{{ formatCurrency(totalUnsettled) }}</span>
            </div>

            <div
              v-for="split in balanceStore.unsettledSplits"
              :key="split.expense_id"
              class="p-3 sm:p-4 border border-line rounded-lg"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="flex-1 min-w-0">
                  <div class="font-medium text-ink truncate">
                    {{ split.description || t('balance.unsettled.noDescription') }}
                  </div>
                  <div class="text-sm text-ink-soft mt-1 flex items-center gap-2">
                    <span>{{ formatDate(split.date) }}</span>
                    <span class="text-xs">{{ t('balance.unsettled.paidBy', { name: split.paid_by_name }) }}</span>
                  </div>
                </div>
                <div class="text-lg font-bold shrink-0" :class="split.paid_by_id === balanceStore.currentMemberId
                  ? 'text-green-600 dark:text-green-400'
                  : 'text-red-600 dark:text-red-400'
                ">
                  {{ split.paid_by_id === balanceStore.currentMemberId ? '+' : '-' }}{{ formatCurrency(split.amount) }}
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </Card>

      <!-- Storico Pagamenti -->
      <Card class="p-4 sm:p-6">
        <h3 class="text-lg font-semibold text-ink mb-4">{{ t('balance.history.title') }}</h3>
        <div v-if="balanceStore.settlements.length === 0" class="text-center py-8">
          <svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <p class="text-ink-soft">{{ t('balance.history.empty') }}</p>
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="settlement in balanceStore.settlements"
            :key="settlement.id"
            class="p-3 sm:p-4 border border-line rounded-lg"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="font-medium text-ink flex items-center gap-2">
                  <span class="truncate">{{ settlement.from_member_name }}</span>
                  <svg class="w-4 h-4 shrink-0 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                  <span class="truncate">{{ settlement.to_member_name }}</span>
                </div>
                <div class="text-sm text-ink-soft mt-1 flex items-center gap-2">
                  <span>{{ formatDate(settlement.date) }}</span>
                  <span v-if="settlement.payment_method" class="text-xs px-2 py-0.5 bg-surface-2 rounded">
                    {{ paymentMethodLabel(settlement.payment_method) }}
                  </span>
                </div>
                <div v-if="settlement.note" class="text-sm text-ink-muted mt-1 italic">
                  "{{ settlement.note }}"
                </div>
              </div>
              <div class="text-lg sm:text-xl font-bold text-green-600 dark:text-green-400 shrink-0">
                {{ formatCurrency(settlement.amount) }}
              </div>
            </div>
          </div>
        </div>
      </Card>
    </template>

    <!-- Settlement Modal -->
    <SettlementModal
      v-if="showSettlementModal"
      :balance="balanceStore.balance"
      :other-member-name="balanceStore.otherMemberName"
      :other-member-id="balanceStore.otherMemberId"
      :current-member-id="balanceStore.currentMemberId"
      :property-id="currentPropertyId"
      @close="showSettlementModal = false"
      @created="onSettlementCreated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { useExpensesStore } from '@/stores/expenses'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import SettlementModal from '@/components/balance/SettlementModal.vue'

const { t } = useI18n()
const emit = defineEmits(['settlement-created'])

const balanceStore = useBalanceStore()
const settingsStore = useSettingsStore()

const showSettlementModal = ref(false)
const currentPropertyId = ref(null)
const unsettledOpen = ref(false)

const balanceMessage = computed(() => {
  const name = balanceStore.otherMemberName || t('balance.partnerFallback')
  if (balanceStore.balance > 0) return t('balance.owesYou', { name })
  if (balanceStore.balance < 0) return t('balance.youOwe', { name })
  return t('balance.even')
})

const totalSettled = computed(() => {
  return balanceStore.settlements.reduce((sum, s) => sum + s.amount, 0)
})

const totalUnsettled = computed(() => {
  return balanceStore.unsettledSplits.reduce((sum, s) => sum + s.amount, 0)
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function paymentMethodLabel(method) {
  const key = `balance.paymentMethods.${method}`
  const label = t(key)
  return label === key ? method : label
}

const expensesStore = useExpensesStore()

async function onSettlementCreated() {
  showSettlementModal.value = false
  if (currentPropertyId.value) {
    await balanceStore.fetchBalanceDetails(currentPropertyId.value)
  }
  // Refresh expenses so "Da saldare" labels update to "Saldato"
  await expensesStore.fetchExpenses({})
  emit('settlement-created')
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
      await balanceStore.fetchBalanceDetails(currentProp.id)
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

onMounted(() => {
  fetchCurrentProperty()
})
</script>

<style scoped>
.filter-expand-enter-active,
.filter-expand-leave-active {
  transition: opacity 0.2s ease, max-height 0.25s ease;
  overflow: hidden;
  max-height: 2000px;
}
.filter-expand-enter-from,
.filter-expand-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
