<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <span class="text-sm text-ink-muted">
        {{ t('utilities.readingsTab.count', { n: utility.readings?.length || 0 }) }}
      </span>
      <Button size="sm" @click="openAddReading">
        <svg class="w-4 h-4 sm:mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span class="hidden sm:inline">{{ t('utilities.readingsTab.addButton') }}</span>
      </Button>
    </div>

    <div v-if="!utility.readings?.length" class="text-center py-8">
      <p class="text-ink-muted mb-3">{{ t('utilities.readingsTab.empty') }}</p>
      <Button size="sm" @click="openAddReading">{{ t('utilities.readingsTab.addReadingButton') }}</Button>
    </div>

    <!-- Timeline -->
    <div v-else class="space-y-1">
      <div
        v-for="(reading, idx) in utility.readings"
        :key="reading.id"
        class="flex gap-3"
      >
        <!-- Timeline line -->
        <div class="flex flex-col items-center w-6 flex-shrink-0">
          <div class="w-3 h-3 rounded-full mt-4"
            :class="idx === 0 ? 'bg-blue-500' : 'bg-surface-3'"
          />
          <div v-if="idx < utility.readings.length - 1" class="w-px flex-1 bg-surface-3" />
        </div>
        <!-- Content -->
        <div class="flex-1 pb-4 min-w-0">
          <div class="p-3 bg-surface rounded-xl border border-line">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="font-medium text-ink">
                  <template v-if="utility.type === 'electricity'">
                    <span v-if="reading.value_f1" class="mr-2">F1: {{ reading.value_f1 }}</span>
                    <span v-if="reading.value_f2" class="mr-2">F2: {{ reading.value_f2 }}</span>
                    <span v-if="reading.value_f3">F3: {{ reading.value_f3 }}</span>
                    <span class="text-ink-faint text-sm ml-1">kWh</span>
                  </template>
                  <template v-else>
                    {{ reading.value || '-' }} {{ consumptionUnit }}
                  </template>
                </div>
                <div class="flex items-center gap-2 text-sm text-ink-muted mt-0.5 flex-wrap">
                  <span>{{ formatDate(reading.reading_date) }}</span>
                  <span v-if="reading.source === 'submitted'" class="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-300 text-xs rounded">
                    {{ t('utilities.readingsTab.submittedBadge') }}
                  </span>
                  <span v-if="readingBillMap[reading.id]" class="px-1.5 py-0.5 bg-green-100 dark:bg-green-900/50 text-green-600 dark:text-green-300 text-xs rounded font-mono">
                    {{ t('utilities.readingsTab.billBadge', { number: readingBillMap[reading.id] }) }}
                  </span>
                </div>
                <div v-if="reading.notes" class="text-xs text-ink-faint mt-1">{{ reading.notes }}</div>
              </div>
              <div class="flex items-center gap-0.5 flex-shrink-0">
                <button
                  @click="openEditReading(reading)"
                  class="p-2 rounded-lg text-ink-faint hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                  :title="t('utilities.readingsTab.editTitle')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                </button>
                <button
                  @click="confirmDeleteReading(reading)"
                  class="p-2 rounded-lg text-ink-faint hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                  :title="t('utilities.readingsTab.deleteTitle')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Reading Modal -->
    <AddReadingModal
      v-if="showReadingModal"
      :utility="utility"
      :reading="editingReading"
      @close="closeReadingModal"
      @saved="onReadingSaved"
    />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { utilitiesAPI } from '@/api/client'
import { formatDate as _formatDate } from '@/utils/dateFormatter'
import Button from '@/components/common/Button.vue'
import AddReadingModal from '@/components/utilities/AddReadingModal.vue'

defineOptions({ name: 'ReadingsTab' })

const props = defineProps({
  utility: { type: Object, required: true },
  consumptionUnit: { type: String, default: '' },
  active: { type: Boolean, default: false },
})

const emit = defineEmits(['reading-saved', 'reading-deleted'])

const { t } = useI18n()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const readingBillMap = ref({})
const showReadingModal = ref(false)
const editingReading = ref(null)

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

async function fetchReadingBillMap() {
  if (!props.utility) return
  try {
    const { data } = await utilitiesAPI.getReadings(props.utility.id)
    const map = {}
    for (const r of data || []) {
      if (r.associated_bill_number) map[r.id] = r.associated_bill_number
    }
    readingBillMap.value = map
  } catch { /* non-critical */ }
}

function openAddReading() {
  editingReading.value = null
  showReadingModal.value = true
}

function openEditReading(reading) {
  editingReading.value = reading
  showReadingModal.value = true
}

function closeReadingModal() {
  showReadingModal.value = false
  editingReading.value = null
}

function onReadingSaved() {
  closeReadingModal()
  emit('reading-saved')
}

async function confirmDeleteReading(reading) {
  const ok = await confirm({
    title: t('utilities.readingsTab.deleteConfirm.title'),
    message: t('utilities.readingsTab.deleteConfirm.message'),
    confirmText: t('utilities.readingsTab.deleteConfirm.action'),
    variant: 'danger'
  })
  if (!ok) return
  try {
    await utilitiesStore.deleteReading(props.utility.id, reading.id)
    emit('reading-deleted')
  } catch (err) {
    console.error('Error deleting reading:', err)
  }
}

watch(() => props.active, (isActive) => {
  if (isActive) fetchReadingBillMap()
}, { immediate: true })
</script>
