<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
       @click.self="$emit('close')">
    <Card class="w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">Nuovo Progetto</h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Name -->
        <Input
          v-model="form.name"
          label="Nome Progetto *"
          placeholder="es. Ristrutturazione Bagno, Matrimonio, Viaggio"
          required
        />

        <!-- Icon -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">
            Icona
          </label>
          <div class="flex gap-2 flex-wrap">
            <button
              v-for="icon in icons"
              :key="icon"
              type="button"
              @click="form.icon = icon"
              :class="[
                'w-12 h-12 rounded-lg text-2xl flex items-center justify-center transition-all',
                form.icon === icon
                  ? 'bg-blue-100 dark:bg-blue-900 ring-2 ring-blue-500'
                  : 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'
              ]"
            >
              {{ icon }}
            </button>
          </div>
        </div>

        <!-- Description -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Descrizione
          </label>
          <textarea
            v-model="form.description"
            rows="3"
            placeholder="Descrivi il progetto..."
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <!-- Budget -->
        <Input
          v-model.number="form.budget"
          label="Budget *"
          type="number"
          step="0.01"
          min="0.01"
          placeholder="0.00"
          required
        />

        <!-- Dates -->
        <div class="grid grid-cols-2 gap-4">
          <Input
            v-model="form.start_date"
            label="Data Inizio *"
            type="date"
            required
          />
          <Input
            v-model="form.end_date"
            label="Data Fine *"
            type="date"
            required
          />
        </div>

        <!-- Status -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Status *
          </label>
          <select
            v-model="form.status"
            required
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="planned">Pianificato</option>
            <option value="active">In Corso</option>
            <option value="completed">Completato</option>
            <option value="cancelled">Annullato</option>
          </select>
        </div>

        <!-- Error -->
        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <!-- Actions -->
        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" :disabled="loading" class="flex-1">
            {{ loading ? 'Creazione...' : 'Crea Progetto' }}
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useProjectsStore } from '@/stores/projects'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const emit = defineEmits(['close', 'created'])
const projectsStore = useProjectsStore()

const loading = ref(false)
const error = ref(null)

const form = ref({
  name: '',
  icon: '🏗️',
  description: '',
  budget: null,
  start_date: new Date().toISOString().split('T')[0],
  end_date: '',
  status: 'planned',
  property_id: 1
})

const icons = ['🏗️', '🔨', '🎨', '🛠️', '🏠', '🚪', '🪟', '💡', '🔌', '🚿', '🛏️', '🍽️', '🌳', '🏊', '🎉', '💍', '✈️', '🎓']

async function handleSubmit() {
  loading.value = true
  error.value = null

  if (new Date(form.value.end_date) < new Date(form.value.start_date)) {
    error.value = 'La data di fine deve essere successiva alla data di inizio'
    loading.value = false
    return
  }

  try {
    await projectsStore.createProject(form.value)
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message
  } finally {
    loading.value = false
  }
}
</script>
