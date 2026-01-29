<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Tutte le Spese</h1>
      <Button @click="showAddExpense = true">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Aggiungi Spesa
      </Button>
    </div>

    <Card class="p-6">
      <div v-if="expensesStore.loading" class="text-center py-8 text-gray-600 dark:text-gray-400">
        Caricamento...
      </div>

      <div v-else-if="expensesStore.expenses.length === 0" class="text-center py-8 text-gray-600 dark:text-gray-400">
        Nessuna spesa registrata.
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="expense in expensesStore.expenses"
          :key="expense.id"
          class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg
                 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors group"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-medium text-gray-900 dark:text-white truncate">
                  {{ expense.description || 'Senza descrizione' }}
                </span>
                <!-- Badge Split -->
                <span
                  v-if="expense.is_split"
                  class="px-2 py-0.5 text-xs rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300"
                >
                  Split
                </span>
                <!-- Badge Saldato/Da saldare -->
                <span
                  v-if="expense.is_split"
                  :class="[
                    'px-2 py-0.5 text-xs rounded-full',
                    isExpenseSettled(expense)
                      ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                      : 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300'
                  ]"
                >
                  {{ isExpenseSettled(expense) ? 'Saldato' : 'Da saldare' }}
                </span>
              </div>
              <div class="text-sm text-gray-600 dark:text-gray-400 mt-1 flex flex-wrap items-center gap-2">
                <span>{{ formatDate(expense.date) }}</span>
                <span
                  v-if="expense.category"
                  class="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs"
                >
                  {{ expense.category.name }}
                </span>
                <span v-if="expense.is_split && expense.paid_by" class="text-xs flex items-center gap-1">
                  Pagato da {{ expense.paid_by.name }}
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                  {{ getSplitPartners(expense) }}
                </span>
              </div>
            </div>
            <div class="text-right ml-4">
              <div class="text-xl font-bold text-blue-600 dark:text-blue-400">
                {{ formatCurrency(expense.amount) }}
              </div>
              <!-- Mostra quota se split -->
              <div v-if="expense.is_split && expense.splits?.length" class="text-xs text-gray-500 dark:text-gray-400">
                ({{ formatCurrency(expense.splits[0]?.amount || 0) }} a testa)
              </div>
              <div class="flex gap-2 justify-end mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  v-if="canEditExpense(expense)"
                  @click="editExpense(expense)"
                  class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"
                >
                  Modifica
                </button>
                <button
                  @click="deleteExpenseConfirm(expense.id)"
                  class="text-sm text-red-600 hover:text-red-700 dark:text-red-400"
                >
                  Elimina
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- Modal Aggiungi Spesa -->
    <AddExpenseModal
      v-if="showAddExpense"
      @close="showAddExpense = false"
      @created="onExpenseCreated"
    />

    <!-- Modal Modifica Spesa -->
    <EditExpenseModal
      v-if="showEditExpense && editingExpense"
      :expense="editingExpense"
      @close="showEditExpense = false; editingExpense = null"
      @updated="onExpenseUpdated"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddExpenseModal from '@/components/expenses/AddExpenseModal.vue'
import EditExpenseModal from '@/components/expenses/EditExpenseModal.vue'

const expensesStore = useExpensesStore()
const showAddExpense = ref(false)
const showEditExpense = ref(false)
const editingExpense = ref(null)

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(value || 0)
}

function formatDate(dateStr) {
  return new Date(dateStr).toLocaleDateString('it-IT', { day: '2-digit', month: 'short', year: 'numeric' })
}

function isExpenseSettled(expense) {
  if (!expense.is_split || !expense.splits || expense.splits.length === 0) return true
  return expense.splits.every(s => s.is_settled)
}

function getSplitPartners(expense) {
  if (!expense.is_split || !expense.splits) return ''
  const partners = expense.splits
    .filter(s => s.member_id !== expense.paid_by_member_id && s.member)
    .map(s => s.member.name)
  return partners.join(', ')
}

function canEditExpense(expense) {
  // Can edit if expense is not settled (or not split at all)
  if (!expense.is_split) return true
  if (!expense.splits || expense.splits.length === 0) return true
  // Can't edit if any split is settled
  return !expense.splits.some(s => s.is_settled)
}

function editExpense(expense) {
  editingExpense.value = expense
  showEditExpense.value = true
}

function onExpenseCreated() {
  showAddExpense.value = false
  expensesStore.fetchExpenses()
}

function onExpenseUpdated() {
  showEditExpense.value = false
  editingExpense.value = null
  expensesStore.fetchExpenses()
}

async function deleteExpenseConfirm(id) {
  if (confirm('Sei sicuro di voler eliminare questa spesa?')) {
    await expensesStore.deleteExpense(id)
  }
}

onMounted(() => {
  expensesStore.fetchExpenses()
})
</script>
