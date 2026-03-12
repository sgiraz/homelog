<template>
  <div class="space-y-6">
    <!-- Loading State -->
    <div v-if="balanceStore.loading" class="text-center py-12">
      <svg class="animate-spin h-12 w-12 mx-auto text-blue-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
      <p class="mt-4 text-gray-600 dark:text-gray-400">Caricamento bilancio...</p>
    </div>

    <template v-else>
      <!-- Balance Card -->
      <Card class="p-4 sm:p-8">
        <div class="text-center">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-2">
            Bilancio con {{ balanceStore.otherMemberName || 'Partner' }}
          </div>
          <div :class="[
            'text-3xl sm:text-5xl font-bold mb-3',
            balanceStore.balance > 0 ? 'text-green-600 dark:text-green-400' :
            balanceStore.balance < 0 ? 'text-red-600 dark:text-red-400' :
            'text-gray-600 dark:text-gray-400'
          ]">
            {{ balanceStore.balance > 0 ? '+' : '' }}{{ formatCurrency(balanceStore.balance) }}
          </div>
          <div class="text-lg text-gray-700 dark:text-gray-300">
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
              {{ balanceStore.balance > 0 ? 'Ricevi Pagamento' : 'Salda Conto' }}
            </Button>
            <Button v-else variant="secondary" disabled>
              <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              Siete in pari
            </Button>
          </div>
        </div>
      </Card>

      <!-- Stats Cards -->
      <div class="grid grid-cols-2 gap-4">
        <Card class="p-4 sm:p-6 text-center">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Pagamenti effettuati</div>
          <div class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ balanceStore.settlements.length }}
          </div>
        </Card>
        <Card class="p-4 sm:p-6 text-center">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Totale saldato</div>
          <div class="text-2xl font-bold text-green-600 dark:text-green-400">
            {{ formatCurrency(totalSettled) }}
          </div>
        </Card>
      </div>

      <!-- Storico Pagamenti -->
      <Card class="p-4 sm:p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Storico Pagamenti</h3>
        <div v-if="balanceStore.settlements.length === 0" class="text-center py-8">
          <svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <p class="text-gray-600 dark:text-gray-400">Nessun pagamento registrato</p>
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="settlement in balanceStore.settlements"
            :key="settlement.id"
            class="p-3 sm:p-4 border border-gray-200 dark:border-gray-700 rounded-lg"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="font-medium text-gray-900 dark:text-white flex items-center gap-2">
                  <span class="truncate">{{ settlement.from_member_name }}</span>
                  <svg class="w-4 h-4 shrink-0 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                  <span class="truncate">{{ settlement.to_member_name }}</span>
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400 mt-1 flex items-center gap-2">
                  <span>{{ formatDate(settlement.date) }}</span>
                  <span v-if="settlement.payment_method" class="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded">
                    {{ paymentMethodLabel(settlement.payment_method) }}
                  </span>
                </div>
                <div v-if="settlement.note" class="text-sm text-gray-500 dark:text-gray-500 mt-1 italic">
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
import { useBalanceStore } from '@/stores/balance'
import { useSettingsStore } from '@/stores/settings'
import { useExpensesStore } from '@/stores/expenses'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import apiClient from '@/api/client'

const emit = defineEmits(['settlement-created'])
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import SettlementModal from '@/components/balance/SettlementModal.vue'

const balanceStore = useBalanceStore()
const settingsStore = useSettingsStore()

const showSettlementModal = ref(false)
const currentPropertyId = ref(null)

const balanceMessage = computed(() => {
  if (balanceStore.balance > 0) return `${balanceStore.otherMemberName || 'Partner'} ti deve`
  if (balanceStore.balance < 0) return `Devi a ${balanceStore.otherMemberName || 'Partner'}`
  return 'Siete in pari!'
})

const totalSettled = computed(() => {
  return balanceStore.settlements.reduce((sum, s) => sum + s.amount, 0)
})

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR'
  }).format(value || 0)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function paymentMethodLabel(method) {
  const labels = {
    bank_transfer: 'Bonifico',
    cash: 'Contanti',
    satispay: 'Satispay',
    paypal: 'PayPal',
    revolut: 'Revolut'
  }
  return labels[method] || method
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
