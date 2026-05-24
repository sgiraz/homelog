<template>
  <BaseModal :title="isEditing ? t('utilities.addReadingModal.editTitle') : t('utilities.addReadingModal.addTitle')" @close="$emit('close')">
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <Input
          v-model="form.reading_date"
          :label="t('utilities.addReadingModal.dateLabel')"
          type="date"
          required
        />

        <!-- Electricity readings (F1/F2/F3) -->
        <div v-if="utility.type === 'electricity'" class="space-y-4">
          <p class="text-sm text-ink-muted">{{ t('utilities.addReadingModal.tariffsLabel') }}</p>
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="block text-xs text-red-600 dark:text-red-400 mb-1 font-medium">{{ t('utilities.addReadingModal.f1Label') }}</label>
              <Input
                v-model="form.value_f1"
                type="number"
                step="0.001"
                min="0"
                placeholder="0"
                required
              />
            </div>
            <div>
              <label class="block text-xs text-yellow-600 dark:text-yellow-400 mb-1 font-medium">{{ t('utilities.addReadingModal.f2Label') }}</label>
              <Input
                v-model="form.value_f2"
                type="number"
                step="0.001"
                min="0"
                placeholder="0"
                required
              />
            </div>
            <div>
              <label class="block text-xs text-green-600 dark:text-green-400 mb-1 font-medium">{{ t('utilities.addReadingModal.f3Label') }}</label>
              <Input
                v-model="form.value_f3"
                type="number"
                step="0.001"
                min="0"
                placeholder="0"
                required
              />
            </div>
          </div>
        </div>

        <!-- Gas/Water single reading -->
        <div v-else-if="utility.type === 'gas' || utility.type === 'water'">
          <Input
            v-model="form.value"
            :label="t('utilities.addReadingModal.meterLabel', { unit: getConsumptionUnit(utility.type) })"
            type="number"
            step="0.001"
            min="0"
            placeholder="0"
            required
          />
          <p class="text-xs text-ink-muted mt-1">
            {{ t('utilities.addReadingModal.meterHint') }}
          </p>
        </div>

        <!-- Source (submitted to provider?) -->
        <div v-if="utility.allows_self_reading === true" class="flex items-center gap-3">
          <input
            type="checkbox"
            id="is-submitted"
            v-model="form.is_submitted"
            class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500"
          />
          <label for="is-submitted" class="text-sm text-ink cursor-pointer">
            {{ t('utilities.addReadingModal.submittedLabel') }}
          </label>
        </div>

        <!-- Note -->
        <div>
          <label class="block text-sm text-ink-soft mb-1">
            {{ t('utilities.addReadingModal.notesLabel') }}
          </label>
          <textarea
            v-model="form.notes"
            rows="2"
            :placeholder="t('utilities.addReadingModal.notesPlaceholder')"
            class="w-full px-3 py-2 border border-line rounded-lg
                   bg-surface text-ink
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            {{ t('utilities.addReadingModal.cancel') }}
          </Button>
          <Button type="submit" :disabled="loading" class="flex-1">
            {{ loading ? t('utilities.addReadingModal.saving') : t('utilities.addReadingModal.save') }}
          </Button>
        </div>
      </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  utility: {
    type: Object,
    required: true
  },
  reading: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])
const { t } = useI18n()
const utilitiesStore = useUtilitiesStore()

const loading = ref(false)
const error = ref(null)

const isEditing = computed(() => !!props.reading)

const today = new Date().toISOString().split('T')[0]

const form = ref({
  reading_date: today,
  value: null,
  value_f1: null,
  value_f2: null,
  value_f3: null,
  is_submitted: false,
  notes: ''
})

function getConsumptionUnit(type) {
  const units = {
    electricity: 'kWh',
    gas: 'Smc',
    water: 'mc',
    waste: ''
  }
  return units[type] || ''
}

function formatDateForInput(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toISOString().split('T')[0]
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    const readingData = {
      reading_date: new Date(form.value.reading_date).toISOString(),
      notes: form.value.notes,
      source: form.value.is_submitted ? 'submitted' : 'manual'
    }

    if (props.utility.type === 'electricity') {
      readingData.value_f1 = form.value.value_f1 ? parseFloat(form.value.value_f1) : null
      readingData.value_f2 = form.value.value_f2 ? parseFloat(form.value.value_f2) : null
      readingData.value_f3 = form.value.value_f3 ? parseFloat(form.value.value_f3) : null
    } else {
      readingData.value = form.value.value ? parseFloat(form.value.value) : null
    }

    if (isEditing.value) {
      await utilitiesStore.updateReading(props.utility.id, props.reading.id, readingData)
    } else {
      await utilitiesStore.addReading(props.utility.id, readingData)
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || t('utilities.addReadingModal.genericError')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (props.reading) {
    form.value = {
      reading_date: formatDateForInput(props.reading.reading_date),
      value: props.reading.value,
      value_f1: props.reading.value_f1,
      value_f2: props.reading.value_f2,
      value_f3: props.reading.value_f3,
      is_submitted: props.reading.source === 'submitted',
      notes: props.reading.notes || ''
    }
  }
})
</script>
