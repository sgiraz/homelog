<template>
  <div class="border border-purple-200 dark:border-purple-800 bg-purple-50 dark:bg-purple-900/20 rounded-lg p-4 space-y-3">
    <div class="flex items-center justify-between">
      <span class="text-sm font-medium text-purple-700 dark:text-purple-300">{{ t('utilities.installments.title') }}</span>
      <button v-if="!isEditing" type="button" @click="addInstallment" class="text-xs text-purple-700 dark:text-purple-300 hover:underline">
        {{ t('utilities.installments.addInstallment') }}
      </button>
    </div>
    <p class="text-xs text-ink-muted">
      {{ t('utilities.installments.totalSummary', { sum: fmt(installmentsSum), total: fmt(amountTotal) }) }}
    </p>
    <div v-for="(inst, idx) in installments" :key="inst.id || idx"
      class="grid gap-2 items-end grid-cols-[auto_1fr_1fr_auto]">
      <div v-if="!isEditing" class="text-xs text-ink-muted pb-2 w-6 text-center">#{{ inst.number }}</div>
      <div v-else class="pb-2 w-8 flex items-center justify-center">
        <input
          type="checkbox"
          :checked="!!inst.is_paid"
          :disabled="instUpdating === inst.id || !!inst.is_locked"
          @change="togglePaid(inst, $event.target.checked)"
          class="w-5 h-5 text-purple-600 rounded border-line focus:ring-purple-500 disabled:opacity-50 disabled:cursor-not-allowed"
          :title="inst.is_locked ? t('utilities.installments.lockedTitle') : (inst.is_paid ? t('utilities.installments.paidTitle') : t('utilities.installments.unpaidTitle'))"
        />
      </div>
      <div>
        <label class="block text-xs text-ink-soft mb-1">{{ isEditing ? t('utilities.installments.editingDueLabel', { n: inst.number }) : t('utilities.installments.dueLabel') }}</label>
        <input v-model="inst.due_date" type="date" :disabled="isEditing"
          class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-purple-500 disabled:opacity-70" />
      </div>
      <div>
        <label class="block text-xs text-ink-soft mb-1">{{ t('utilities.installments.amountLabel') }}</label>
        <input v-model="inst.amount" type="number" step="0.01" placeholder="0.00" :disabled="isEditing"
          class="w-full px-2 py-1.5 text-sm border border-line rounded bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-purple-500 disabled:opacity-70" />
      </div>
      <button v-if="!isEditing" type="button" @click="removeInstallment(idx)"
        class="text-red-600 dark:text-red-400 text-xs pb-2 px-1 hover:underline"
        :disabled="installments.length <= 1">−</button>
      <span v-else class="text-xs pb-2 px-1" :class="inst.is_paid ? 'text-green-600 dark:text-green-400' : 'text-ink-faint'">
        {{ inst.is_paid ? '✓' : '—' }}
      </span>
    </div>
    <div v-if="!isEditing && amountMismatch" class="text-xs text-red-600 dark:text-red-400">
      {{ t('utilities.installments.mismatch') }}
    </div>
    <p v-if="isEditing" class="text-xs text-ink-faint">
      {{ t('utilities.installments.editHint') }}
    </p>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { formatNumber as _formatNumber } from '@/utils/dateFormatter'
import { utilitiesAPI } from '@/api/client'
import { apiErrorMessage } from '@/utils/apiError'

const props = defineProps({
  utilityId: { type: [Number, String], required: true },
  billId: { type: [Number, String], default: null },
  isEditing: { type: Boolean, default: false },
  amountTotal: { type: [Number, String], default: 0 },
  defaultDueDate: { type: String, default: '' }
})

const emit = defineEmits(['installment-updated', 'error'])
const installments = defineModel({ type: Array, default: () => [] })

const { t } = useI18n()
const settingsStore = useSettingsStore()
const instUpdating = ref(null)

const installmentsSum = computed(() =>
  (installments.value || []).reduce((s, i) => s + (parseFloat(i.amount) || 0), 0)
)

const amountMismatch = computed(() => {
  if (installments.value.length <= 1) return false
  return Math.abs(installmentsSum.value - (parseFloat(props.amountTotal) || 0)) > 0.01
})

function addInstallment() {
  installments.value.push({
    number: installments.value.length + 1,
    due_date: props.defaultDueDate || '',
    amount: 0,
    is_paid: false
  })
}

function removeInstallment(idx) {
  installments.value.splice(idx, 1)
  installments.value.forEach((inst, i) => { inst.number = i + 1 })
}

async function togglePaid(inst, newValue) {
  if (!props.isEditing || !inst.id) return
  instUpdating.value = inst.id
  try {
    await utilitiesAPI.updateInstallment(props.utilityId, props.billId, inst.id, {
      is_paid: newValue,
      paid_at: newValue ? new Date().toISOString() : null
    })
    inst.is_paid = newValue
    emit('installment-updated')
  } catch (err) {
    console.error('Errore toggle rata:', err)
    emit('error', apiErrorMessage(err, t('utilities.installments.updateError')))
  } finally {
    instUpdating.value = null
  }
}

defineExpose({ amountMismatch })

function fmt(value) {
  if (value == null) return '-'
  return _formatNumber(value, settingsStore.formatSettings)
}
</script>
