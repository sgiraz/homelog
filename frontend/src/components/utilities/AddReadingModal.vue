<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[60] p-4 pt-8 overflow-y-auto overflow-x-hidden"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-md p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ isEditing ? 'Modifica Lettura' : 'Nuova Lettura' }}
        </h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Data Lettura -->
        <Input
          v-model="form.reading_date"
          label="Data Lettura *"
          type="date"
          required
        />

        <!-- Electricity readings (F1/F2/F3) -->
        <div v-if="utility.type === 'electricity'" class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">Letture per fascia (kWh) *</p>
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="block text-xs text-red-600 dark:text-red-400 mb-1 font-medium">F1 (Punta)</label>
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
              <label class="block text-xs text-yellow-600 dark:text-yellow-400 mb-1 font-medium">F2 (Intermedia)</label>
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
              <label class="block text-xs text-green-600 dark:text-green-400 mb-1 font-medium">F3 (Fuori Punta)</label>
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
            :label="'Lettura Contatore (' + getConsumptionUnit(utility.type) + ') *'"
            type="number"
            step="0.001"
            min="0"
            placeholder="0"
            required
          />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Inserisci il valore visualizzato sul contatore
          </p>
        </div>

        <!-- Source (submitted to provider?) - Only show if provider accepts self-readings -->
        <div v-if="utility.allows_self_reading === true" class="flex items-center gap-3">
          <input
            type="checkbox"
            id="is-submitted"
            v-model="form.is_submitted"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <label for="is-submitted" class="text-sm text-gray-900 dark:text-white cursor-pointer">
            Inviata al fornitore
          </label>
        </div>

        <!-- Note -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Note
          </label>
          <textarea
            v-model="form.notes"
            rows="2"
            placeholder="Note opzionali..."
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" :disabled="loading" class="flex-1">
            {{ loading ? 'Salvataggio...' : 'Salva' }}
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useUtilitiesStore } from '@/stores/utilities'
import Card from '@/components/common/Card.vue'
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
const utilitiesStore = useUtilitiesStore()

const loading = ref(false)
const error = ref(null)

const isEditing = computed(() => !!props.reading)

const today = new Date().toISOString().split('T')[0]

const form = ref({
  reading_date: today,
  value: null,        // For gas/water single reading
  value_f1: null,     // For electricity F1
  value_f2: null,     // For electricity F2
  value_f3: null,     // For electricity F3
  is_submitted: false, // Whether submitted to provider
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

    // For electricity, send F1/F2/F3
    if (props.utility.type === 'electricity') {
      readingData.value_f1 = form.value.value_f1 ? parseFloat(form.value.value_f1) : null
      readingData.value_f2 = form.value.value_f2 ? parseFloat(form.value.value_f2) : null
      readingData.value_f3 = form.value.value_f3 ? parseFloat(form.value.value_f3) : null
    } else {
      // For gas/water, send single value
      readingData.value = form.value.value ? parseFloat(form.value.value) : null
    }

    if (isEditing.value) {
      await utilitiesStore.updateReading(props.utility.id, props.reading.id, readingData)
    } else {
      await utilitiesStore.addReading(props.utility.id, readingData)
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
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
