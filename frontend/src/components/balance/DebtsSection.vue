<template>
  <div class="space-y-6">
    <!-- Loading State -->
    <div v-if="balanceStore.debtsLoading && balanceStore.debts.length === 0" class="text-center py-12">
      <svg class="animate-spin h-12 w-12 mx-auto text-blue-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
      <p class="mt-4 text-ink-soft">{{ t('balance.debts.loading') }}</p>
    </div>

    <template v-else>
      <!-- Empty state -->
      <Card v-if="balanceStore.debts.length === 0" class="p-6 sm:p-10 text-center">
        <svg class="w-16 h-16 mx-auto text-ink-faint mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 21h18M5 21V7l7-4 7 4v14M9 21v-6h6v6" />
        </svg>
        <h3 class="text-lg font-semibold text-ink mb-2">{{ t('balance.debts.emptyTitle') }}</h3>
        <p class="text-ink-soft max-w-md mx-auto">{{ t('balance.debts.emptyBody') }}</p>
        <p class="text-sm text-ink-muted mt-3 max-w-md mx-auto">{{ t('balance.debts.emptyHint') }}</p>
      </Card>

      <template v-else>
        <!-- Totals -->
        <div class="grid grid-cols-2 gap-4">
          <Card class="p-4 sm:p-6 text-center">
            <div class="text-sm text-ink-soft mb-1">{{ t('balance.debts.totalIOwe') }}</div>
            <div class="text-2xl font-bold text-red-600 dark:text-red-400">
              {{ formatCurrency(balanceStore.totalIOwe) }}
            </div>
          </Card>
          <Card class="p-4 sm:p-6 text-center">
            <div class="text-sm text-ink-soft mb-1">
              {{ t('balance.debts.totalTheyOwe', { name: balanceStore.otherMemberName || t('balance.partnerFallback') }) }}
            </div>
            <div class="text-2xl font-bold text-green-600 dark:text-green-400">
              {{ formatCurrency(balanceStore.totalTheyOwe) }}
            </div>
          </Card>
        </div>

        <!-- Active debts -->
        <Card v-for="debt in activeDebts" :key="debt.expense_id" class="p-4 sm:p-6">
          <div class="flex items-start justify-between gap-3 mb-3">
            <div class="min-w-0">
              <h3 class="font-semibold text-ink truncate">
                {{ debt.description || t('balance.debts.noDescription') }}
              </h3>
              <div class="text-sm text-ink-soft mt-1 flex items-center gap-2 flex-wrap">
                <span>{{ formatDate(debt.date) }}</span>
                <span
                  v-if="debt.project_name"
                  class="text-xs px-2 py-0.5 bg-surface-2 rounded"
                >{{ debt.project_name }}</span>
              </div>
            </div>
            <div class="text-right shrink-0">
              <div class="text-xs text-ink-soft">{{ t('balance.debts.remainingLabel') }}</div>
              <div
                :class="[
                  'text-xl sm:text-2xl font-bold',
                  debt.i_owe ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'
                ]"
              >
                {{ formatCurrency(debt.remaining_amount) }}
              </div>
            </div>
          </div>

          <p class="text-sm text-ink-soft mb-2">
            {{ debt.i_owe
              ? t('balance.debts.youOwe', { name: debt.counterpart_name })
              : t('balance.debts.owesYou', { name: debt.counterpart_name }) }}
          </p>

          <!-- Progress -->
          <div class="h-2 rounded-full bg-surface-2 overflow-hidden">
            <div
              class="h-full rounded-full bg-accent transition-all"
              :style="{ width: progressPercent(debt) + '%' }"
            />
          </div>
          <div class="flex items-center justify-between mt-1.5 text-xs text-ink-muted">
            <span>{{ t('balance.debts.repaidOf', {
              paid: formatCurrency(debt.settled_amount),
              total: formatCurrency(debt.amount)
            }) }}</span>
            <span class="tabular-nums">{{ progressPercent(debt) }}%</span>
          </div>

          <div class="flex flex-wrap gap-3 mt-4">
            <Button size="sm" @click="payingDebt = debt">
              {{ debt.i_owe ? t('balance.debts.payButton') : t('balance.debts.receiveButton') }}
            </Button>
            <Button
              v-if="debt.settled_amount === 0"
              size="sm"
              variant="secondary"
              @click="moveBackToBalance(debt)"
            >
              {{ t('balance.debts.moveBackButton') }}
            </Button>
          </div>
        </Card>

        <!-- Repaid debts -->
        <Card v-if="repaidDebts.length > 0" class="p-4 sm:p-6">
          <button
            @click="repaidOpen = !repaidOpen"
            :aria-expanded="repaidOpen"
            class="w-full flex items-center justify-between text-left"
          >
            <h3 class="text-lg font-semibold text-ink flex items-center gap-2">
              {{ t('balance.debts.repaidTitle') }}
              <span class="text-sm font-normal px-2 py-0.5 rounded-full bg-surface-2 text-ink-soft">
                {{ repaidDebts.length }}
              </span>
            </h3>
            <svg
              :class="['w-5 h-5 text-ink-faint transition-transform', repaidOpen ? 'rotate-180' : '']"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          <div v-if="repaidOpen" class="mt-4 space-y-2">
            <div
              v-for="debt in repaidDebts"
              :key="debt.expense_id"
              class="flex items-center justify-between gap-3 p-3 border border-line rounded-lg"
            >
              <div class="min-w-0">
                <div class="text-sm font-medium text-ink truncate">
                  {{ debt.description || t('balance.debts.noDescription') }}
                </div>
                <div class="text-xs text-ink-muted">{{ formatDate(debt.date) }}</div>
              </div>
              <div class="text-sm font-medium text-green-600 dark:text-green-400 shrink-0">
                {{ t('balance.debts.settledBadge') }}
              </div>
            </div>
          </div>
        </Card>

        <!-- Payment history across all debts -->
        <Card class="p-4 sm:p-6">
          <h3 class="text-lg font-semibold text-ink mb-4">{{ t('balance.debts.historyTitle') }}</h3>
          <p v-if="paymentHistory.length === 0" class="text-ink-soft text-center py-6">
            {{ t('balance.debts.historyEmpty') }}
          </p>
          <div v-else class="space-y-3">
            <div
              v-for="payment in paymentHistory"
              :key="payment.settlement_id"
              class="flex items-start justify-between gap-3 p-3 border border-line rounded-lg"
            >
              <div class="min-w-0">
                <div class="text-sm font-medium text-ink truncate">{{ payment.debtLabel }}</div>
                <div class="text-sm text-ink-soft mt-1 flex items-center gap-2 flex-wrap">
                  <span>{{ formatDate(payment.date) }}</span>
                  <span v-if="payment.payment_method" class="text-xs px-2 py-0.5 bg-surface-2 rounded">
                    {{ paymentMethodLabel(payment.payment_method) }}
                  </span>
                </div>
                <div v-if="payment.source_label" class="text-xs text-ink-muted mt-1">
                  {{ t('balance.debts.fundedBy', { source: payment.source_label }) }}
                </div>
              </div>
              <div class="text-base font-bold text-green-600 dark:text-green-400 shrink-0">
                {{ formatCurrency(payment.amount) }}
              </div>
            </div>
          </div>
        </Card>
      </template>
    </template>

    <SettlementModal
      v-if="payingDebt"
      :debt="payingDebt"
      :other-member-name="payingDebt.counterpart_name"
      :other-member-id="balanceStore.otherMemberId"
      :current-member-id="balanceStore.currentMemberId"
      :property-id="propertyId"
      @close="payingDebt = null"
      @created="onPaymentCreated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import SettlementModal from '@/components/balance/SettlementModal.vue'

const { t } = useI18n()
const balanceStore = useBalanceStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const propertyId = ref(null)
const payingDebt = ref(null)
const repaidOpen = ref(false)

const activeDebts = computed(() => balanceStore.debts.filter(d => !d.is_fully_repaid))
const repaidDebts = computed(() => balanceStore.debts.filter(d => d.is_fully_repaid))

// One list across every debt — which debt a payment went to is on each row.
const paymentHistory = computed(() => {
  const rows = []
  for (const debt of balanceStore.debts) {
    for (const payment of debt.payments || []) {
      rows.push({ ...payment, debtLabel: debt.description || t('balance.debts.noDescription') })
    }
  }
  return rows.sort((a, b) => String(b.date).localeCompare(String(a.date)))
})

function progressPercent(debt) {
  if (!debt.amount) return 0
  return Math.min(100, Math.round((debt.settled_amount / debt.amount) * 100))
}

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

async function moveBackToBalance(debt) {
  const ok = await confirm({
    title: t('balance.debts.moveBackConfirmTitle'),
    message: t('balance.debts.moveBackConfirmBody'),
    confirmText: t('balance.debts.moveBackButton')
  })
  if (!ok) return

  try {
    await balanceStore.setLongTermDebt(debt.expense_id, false, propertyId.value)
    window.$toast?.success(t('balance.debts.moveBackToast'))
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || t('balance.debts.genericError'))
  }
}

async function onPaymentCreated() {
  payingDebt.value = null
  if (propertyId.value) await balanceStore.fetchDebts(propertyId.value)
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      propertyId.value = currentProp.id
      await balanceStore.fetchDebts(currentProp.id)
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

onMounted(() => {
  fetchCurrentProperty()
})

// See BalanceSection: the parent view is kept alive, so a repayment recorded
// elsewhere would otherwise show a stale remainder until a hard reload.
onActivated(() => {
  if (propertyId.value) balanceStore.fetchDebts(propertyId.value)
})
</script>
