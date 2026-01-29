<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-md p-6">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ balance > 0 ? 'Ricevi Pagamento' : 'Salda Conto' }}
        </h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Balance Info -->
      <div class="bg-blue-50 dark:bg-blue-900/30 rounded-xl p-6 text-center mb-6">
        <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Importo da saldare</div>
        <div class="text-4xl font-bold text-blue-600 dark:text-blue-400">
          {{ formatCurrency(Math.abs(balance)) }}
        </div>
        <div class="text-sm text-gray-600 dark:text-gray-400 mt-2">
          {{ balance > 0 ? `Da ricevere da ${otherMemberName}` : `Da pagare a ${otherMemberName}` }}
        </div>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <Input
          v-model="form.amount"
          label="Importo effettivo"
          type="number"
          step="0.01"
          min="0.01"
          required
        />

        <Input
          v-model="form.date"
          label="Data pagamento"
          type="date"
          required
        />

        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Metodo pagamento
          </label>
          <select
            v-model="form.payment_method"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="bank_transfer">Bonifico bancario</option>
            <option value="cash">Contanti</option>
            <option value="satispay">Satispay</option>
            <option value="paypal">PayPal</option>
            <option value="revolut">Revolut</option>
          </select>
        </div>

        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Nota (opzionale)
          </label>
          <textarea
            v-model="form.note"
            rows="2"
            placeholder="Es: Pareggio gennaio 2026"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
          <p class="text-sm text-yellow-800 dark:text-yellow-200">
            Registrando questo pagamento, le spese collegate verranno marcate come saldate.
          </p>
        </div>

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" variant="success" :disabled="loading" class="flex-1">
            {{ loading ? 'Registrazione...' : 'Conferma Pagamento' }}
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useBalanceStore } from '@/stores/balance'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

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
const authStore = useAuthStore()

const loading = ref(false)
const error = ref(null)

const form = ref({
  amount: Math.abs(props.balance),
  date: new Date().toISOString().split('T')[0],
  payment_method: 'bank_transfer',
  note: ''
})

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR'
  }).format(value || 0)
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    // Determine from/to based on balance direction
    const currentMemberId = props.currentMemberId
    const otherMemberId = props.otherMemberId

    if (!currentMemberId || !otherMemberId) {
      error.value = 'Impossibile determinare i membri per il saldo'
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
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante la registrazione'
  } finally {
    loading.value = false
  }
}
</script>
