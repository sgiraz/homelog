<template>
  <div class="space-y-4">
    <div class="flex justify-between items-center">
      <span class="text-sm text-ink-muted">
        {{ t('utilities.priceHistoryTab.count', { n: utility.price_changes?.length || 0 }) }}
      </span>
    </div>

    <div v-if="!utility.price_changes?.length" class="text-center py-8">
      <p class="text-ink-muted">{{ t('utilities.priceHistoryTab.empty') }}</p>
      <p class="text-xs text-ink-faint mt-1">{{ t('utilities.priceHistoryTab.emptyHint') }}</p>
    </div>

    <!-- Price history timeline -->
    <div v-else class="space-y-1">
      <div
        v-for="(change, idx) in utility.price_changes"
        :key="change.id"
        class="flex gap-3"
      >
        <div class="flex flex-col items-center w-6 flex-shrink-0">
          <div class="w-3 h-3 rounded-full mt-4"
            :class="change.new_amount > change.old_amount ? 'bg-red-500' : 'bg-green-500'"
          />
          <div v-if="idx < utility.price_changes.length - 1" class="w-px flex-1 bg-surface-3" />
        </div>
        <div class="flex-1 pb-4 min-w-0">
          <div class="p-3 bg-surface rounded-xl border border-line">
            <div class="flex items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="font-medium text-ink">
                  {{ formatCurrency(change.old_amount) }} → {{ formatCurrency(change.new_amount) }}
                  <span :class="change.new_amount > change.old_amount ? 'text-red-500' : 'text-green-500'" class="text-sm ml-1">
                    ({{ change.new_amount > change.old_amount ? '+' : '' }}{{ formatCurrency(change.new_amount - change.old_amount) }})
                  </span>
                </div>
                <div class="text-sm text-ink-muted mt-0.5">
                  {{ t('utilities.priceHistoryTab.fromDate', { date: formatDate(change.effective_date) }) }}
                </div>
                <div v-if="change.reason" class="text-xs text-gray-400 mt-1">{{ change.reason }}</div>
                <div v-if="change.cancellation_deadline" class="mt-1 px-2 py-1 bg-yellow-50 dark:bg-yellow-900/20 rounded text-xs text-yellow-700 dark:text-yellow-300 inline-block">
                  {{ t('utilities.priceHistoryTab.cancellationDeadline', { date: formatDate(change.cancellation_deadline) }) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Current price summary -->
    <Card v-if="utility.recurring_amount" class="p-4">
      <div class="text-center">
        <div class="text-xs text-ink-muted mb-1">{{ t('utilities.priceHistoryTab.currentAmount') }}</div>
        <div class="text-2xl font-bold text-ink">{{ formatCurrency(utility.recurring_amount) }}</div>
        <div class="text-xs text-gray-400 mt-1">{{ t('utilities.priceHistoryTab.perMonth') }}</div>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'

defineOptions({ name: 'PriceHistoryTab' })

defineProps({
  utility: { type: Object, required: true },
})

const { t } = useI18n()
const settingsStore = useSettingsStore()

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}
</script>
