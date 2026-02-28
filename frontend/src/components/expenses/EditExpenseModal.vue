<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">Modifica Spesa</h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <Input
          v-model="form.amount"
          label="Importo *"
          type="number"
          step="0.01"
          min="0.01"
          placeholder="0.00"
          required
        />

        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Descrizione *
          </label>
          <textarea
            v-model="form.description"
            rows="2"
            required
            placeholder="Es: Spesa supermercato"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Categoria *
          </label>
          <select
            v-model.number="form.category_id"
            @change="form.subcategory_id = null"
            required
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="" disabled>Seleziona categoria</option>
            <option
              v-for="cat in categories"
              :key="cat.id"
              :value="cat.id"
            >
              {{ cat.icon }} {{ cat.name }}
            </option>
          </select>
        </div>

        <!-- Subcategory selector -->
        <div v-if="selectedCategorySubcategories.length > 0">
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Sottocategoria (opzionale)
          </label>
          <select
            v-model.number="form.subcategory_id"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null">Nessuna sottocategoria</option>
            <option
              v-for="sub in selectedCategorySubcategories"
              :key="sub.id"
              :value="sub.id"
            >
              {{ sub.name }}
            </option>
          </select>
        </div>

        <Input
          v-model="form.date"
          label="Data *"
          type="date"
          required
        />

        <!-- Project (Optional) -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
            Progetto (opzionale)
          </label>
          <select
            v-model.number="form.project_id"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null">Nessun progetto</option>
            <option
              v-for="proj in activeProjects"
              :key="proj.id"
              :value="proj.id"
            >
              {{ proj.icon }} {{ proj.name }}
            </option>
          </select>
        </div>

        <!-- Note about split - editing split is not allowed -->
        <div v-if="expense.is_split" class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
          <p class="text-sm text-yellow-800 dark:text-yellow-200">
            Le impostazioni di divisione non possono essere modificate. Per cambiare la divisione, elimina e ricrea la spesa.
          </p>
        </div>

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div class="flex gap-3 pt-4">
          <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
            Annulla
          </Button>
          <Button type="submit" :disabled="loading" class="flex-1">
            {{ loading ? 'Salvataggio...' : 'Salva Modifiche' }}
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { categoriesAPI, projectsAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  expense: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated'])
const expensesStore = useExpensesStore()

const loading = ref(false)
const error = ref(null)
const categories = ref([])
const activeProjects = ref([])

const form = ref({
  amount: null,
  description: '',
  category_id: 1,
  subcategory_id: null,
  date: '',
  project_id: null
})

const selectedCategorySubcategories = computed(() => {
  if (!form.value.category_id) return []
  const cat = categories.value.find(c => c.id === form.value.category_id)
  return cat?.subcategories || []
})

async function fetchCategories() {
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data
  } catch (err) {
    console.error('Error fetching categories:', err)
  }
}

async function fetchActiveProjects() {
  try {
    const { data } = await projectsAPI.list({ status: 'active' })
    activeProjects.value = data || []
  } catch (err) {
    console.error('Error fetching projects:', err)
  }
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    const expenseData = {
      amount: parseFloat(form.value.amount),
      description: form.value.description,
      category_id: form.value.category_id,
      subcategory_id: form.value.subcategory_id || undefined,
      date: form.value.date,
      project_id: form.value.project_id
    }

    console.log('Updating expense with data:', expenseData)

    await expensesStore.updateExpense(props.expense.id, expenseData)
    window.$toast?.success('Spesa aggiornata!')
    emit('updated')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchCategories()
  fetchActiveProjects()
  // Pre-populate form with existing expense data
  form.value = {
    amount: props.expense.amount,
    description: props.expense.description,
    category_id: props.expense.category_id || props.expense.category?.id || 1,
    subcategory_id: props.expense.subcategory_id || null,
    date: props.expense.date ? props.expense.date.split('T')[0] : '',
    project_id: props.expense.project_id || null
  }
})
</script>
