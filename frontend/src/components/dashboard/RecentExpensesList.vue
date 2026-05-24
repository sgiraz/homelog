<template>
  <Card class="p-6">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-xl font-bold text-ink">{{ t('dashboard.recent.title') }}</h2>
      <router-link
        v-if="expenses.length > 3"
        to="/expenses"
        class="text-blue-600 hover:text-blue-700 dark:text-blue-400 text-sm"
      >
        {{ t('dashboard.recent.viewAll') }}
      </router-link>
    </div>

    <div v-if="loading" class="text-center py-8 text-ink-soft">
      <svg class="animate-spin h-8 w-8 mx-auto mb-2" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
      {{ t('dashboard.recent.loading') }}
    </div>

    <div v-else-if="expenses.length === 0" class="text-center py-8">
      <svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
      <p class="text-ink-soft">{{ t('dashboard.recent.empty') }}</p>
      <Button @click="emit('add')" class="mt-4">
        {{ t('dashboard.recent.addFirst') }}
      </Button>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="expense in expenses.slice(0, 3)"
        :key="expense.id"
        class="p-3 sm:p-4 border border-line rounded-lg
               hover:bg-surface-2 transition-colors group"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-medium text-ink line-clamp-2">
                {{ expense.description || t('expenses.noDescription') }}
              </span>
              <!-- Badge Split -->
              <span
                v-if="expense.is_split"
                class="px-2 py-0.5 text-xs rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300"
              >
                {{ t('expenses.splitBadge') }}
              </span>
              <!-- Badge Saldato/Da saldare -->
              <span
                v-if="expense.is_split"
                :class="[
                  'px-2 py-0.5 text-xs rounded-full',
                  isExpenseSettled(expense)
                    ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                    : 'bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-700'
                ]"
              >
                {{ isExpenseSettled(expense) ? t('expenses.settled') : t('expenses.unsettled') }}
              </span>
            </div>
            <div class="text-sm text-ink-soft mt-1 flex flex-wrap items-center gap-2">
              <span>{{ formatDate(expense.date) }}</span>
              <span
                v-if="expense.category"
                class="px-2 py-0.5 bg-surface-2 rounded text-xs"
              >
                {{ expense.category.name }}
              </span>
              <span v-if="expense.is_split && expense.paid_by" class="text-xs flex items-center gap-1 max-w-full overflow-hidden">
                <span class="hidden sm:inline">{{ t('expenses.paidBy') }}</span>
                <span class="truncate">{{ expense.paid_by.name }}</span>
                <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                </svg>
                <span class="truncate">{{ getSplitPartners(expense) }}</span>
              </span>
            </div>
          </div>
          <div class="text-right shrink-0">
            <div class="text-xl font-bold text-blue-600 dark:text-blue-400">
              {{ formatCurrency(expense.amount) }}
            </div>
            <!-- Mostra quota se split -->
            <div v-if="expense.is_split && expense.splits?.length" class="text-xs text-ink-muted">
              {{ t('expenses.shareEach', { amount: formatCurrency(expense.splits[0]?.amount || 0) }) }}
            </div>
            <!-- Bill-linked indicator -->
            <div v-if="expense.bill_id" class="text-xs text-orange-600 dark:text-orange-400 mt-1 text-right">
              {{ t('expenses.fromBill') }}
            </div>
            <!-- Actions: only visible to the creator, not for bill-linked or fully-settled expenses -->
            <div
              v-if="isOwner(expense) && !expense.bill_id && !(expense.is_split && isExpenseSettled(expense))"
              class="flex gap-1 justify-end mt-1 [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 transition-opacity"
            >
              <button
                @click="emit('edit', expense)"
                class="p-1.5 text-blue-600 hover:text-blue-700 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded"
                :aria-label="t('expenses.editAria')"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
              <button
                @click="emit('delete', expense.id)"
                class="p-1.5 text-red-600 hover:text-red-700 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                :aria-label="t('expenses.deleteAria')"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup>
defineOptions({ name: 'RecentExpensesList' })

import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()
const props = defineProps({
  expenses: {
    type: Array,
    required: true
  },
  loading: {
    type: Boolean,
    required: true
  },
  formatCurrency: {
    type: Function,
    required: true
  },
  formatDate: {
    type: Function,
    required: true
  },
  currentUserId: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['add', 'edit', 'delete'])

function isOwner(expense) {
  return expense.user_id === props.currentUserId
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
</script>
