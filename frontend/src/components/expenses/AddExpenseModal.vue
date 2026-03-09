<template>
  <BaseModal title="Nuova Spesa" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <Input
        v-model="form.amount"
        label="Importo *"
        type="number"
        step="0.01"
        min="0.01"
        placeholder="0.00"
        inputmode="decimal"
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
          autocorrect="off"
          autocapitalize="off"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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

      <!-- Subcategory (shown only when selected category has subcategories) -->
      <div v-if="selectedCategorySubcategories.length > 0">
        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
          Sottocategoria (opzionale)
        </label>
        <select
          v-model.number="form.subcategory_id"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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

      <!-- Sezione Split -->
      <div v-if="hasMultipleUsers" class="border-t border-gray-200 dark:border-gray-700 pt-4 space-y-3">
        <div class="flex items-center gap-3">
          <input
            type="checkbox"
            id="split-checkbox"
            v-model="form.is_split"
            class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <label for="split-checkbox" class="text-sm font-medium text-gray-900 dark:text-white cursor-pointer">
            Dividi questa spesa
          </label>
        </div>

        <div v-if="form.is_split" class="space-y-4 pl-2 border-l-2 border-blue-200 dark:border-blue-800 ml-2">
          <!-- Chi ha pagato -->
          <div class="pl-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Chi ha pagato?
            </label>
            <div class="space-y-2">
              <label
                v-for="user in householdUsers"
                :key="user.id"
                class="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
              >
                <input
                  type="radio"
                  :value="user.id"
                  v-model="form.paid_by_member_id"
                  class="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500"
                />
                <span class="text-gray-900 dark:text-white">{{ user.name }}</span>
                <span v-if="user.user_id === authStore.user?.id" class="text-xs text-gray-500">(tu)</span>
              </label>
            </div>
          </div>

          <!-- Con chi dividere -->
          <div class="pl-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Dividi con:
            </label>
            <div class="space-y-2">
              <label
                v-for="user in otherUsers"
                :key="user.id"
                class="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
              >
                <input
                  type="checkbox"
                  :value="user.id"
                  v-model="form.split_with_member_ids"
                  class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
                />
                <span class="text-gray-900 dark:text-white">{{ user.name }}</span>
              </label>
            </div>
          </div>

          <!-- Riepilogo divisione -->
          <div
            v-if="form.split_with_member_ids.length > 0 && form.amount"
            class="ml-4 bg-blue-50 dark:bg-blue-900/30 p-4 rounded-lg"
          >
            <div class="text-sm space-y-1">
              <div class="font-medium text-gray-900 dark:text-white mb-2">Riepilogo divisione:</div>
              <div class="text-gray-600 dark:text-gray-400">
                Totale: <span class="font-medium">{{ formatCurrency(form.amount) }}</span>
              </div>
              <div class="text-gray-600 dark:text-gray-400">
                Diviso tra {{ totalPeople }} persone
              </div>
              <div class="text-lg font-bold text-blue-600 dark:text-blue-400 mt-2">
                {{ formatCurrency(splitAmount) }} a testa
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          Annulla
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? 'Salvataggio...' : 'Salva' }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useExpensesStore } from '@/stores/expenses'
import { useAuthStore } from '@/stores/auth'
import apiClient, { categoriesAPI, projectsAPI } from '@/api/client'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'


const emit = defineEmits(['close', 'created'])
const expensesStore = useExpensesStore()
const authStore = useAuthStore()

const loading = ref(false)
const error = ref(null)
const userSettings = ref(null)
const categories = ref([])
const activeProjects = ref([])

const householdUsers = ref([])
const currentPropertyId = ref(null)

const form = ref({
  amount: null,
  description: '',
  category_id: 1,
  subcategory_id: null,
  date: new Date().toISOString().split('T')[0],
  paid_by_member_id: null,
  is_split: false,
  split_with_member_ids: [],
  project_id: null,
  property_id: null
})

const hasMultipleUsers = computed(() => householdUsers.value.length > 1)

const selectedCategorySubcategories = computed(() => {
  if (!form.value.category_id) return []
  const cat = categories.value.find(c => c.id === form.value.category_id)
  return cat?.subcategories || []
})

const otherUsers = computed(() => {
  return householdUsers.value.filter(u => u.id !== form.value.paid_by_member_id)
})

const totalPeople = computed(() => {
  return form.value.split_with_member_ids.length + 1
})

const splitAmount = computed(() => {
  if (!form.value.amount || totalPeople.value === 0) return 0
  return parseFloat(form.value.amount) / totalPeople.value
})

function formatCurrency(value) {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR'
  }).format(value || 0)
}

async function fetchCategories() {
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data
    if (!form.value.category_id && data.length > 0) {
      form.value.category_id = data[0].id
    }
  } catch (err) {
    console.error('Error fetching categories:', err)
  }
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
      form.value.property_id = currentProp.id
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function fetchUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    userSettings.value = data

    if (data.default_split_with_member_ids) {
      try {
        const defaultMembers = JSON.parse(data.default_split_with_member_ids)
        if (defaultMembers && defaultMembers.length > 0) {
          // Filter out the payer's own member ID to avoid counting them twice
          const filtered = defaultMembers.filter(id => id !== form.value.paid_by_member_id)
          if (filtered.length > 0) {
            form.value.is_split = true
            form.value.split_with_member_ids = filtered
          }
        }
      } catch (e) {
        console.log('No default split members configured')
      }
    }
  } catch (err) {
    console.log('Using default settings')
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

async function fetchHouseholdUsers() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/members`)
    householdUsers.value = data

    const currentUserId = authStore.user?.id
    const myMember = data.find(m => m.user_id === currentUserId)
    if (myMember) {
      form.value.paid_by_member_id = myMember.id
    }
  } catch (err) {
    console.log('Error fetching household members:', err)
    householdUsers.value = []
  }
}

watch(() => form.value.is_split, (newVal) => {
  if (!newVal) {
    form.value.split_with_member_ids = []
  }
})

watch(() => form.value.paid_by_member_id, () => {
  form.value.split_with_member_ids = []
})

watch(() => form.value.project_id, (newProjectId) => {
  if (!newProjectId) return
  const project = activeProjects.value.find(p => p.id === newProjectId)
  if (project?.shared_with?.length > 0) {
    const sharedUserIds = project.shared_with.map(u => u.id)
    const matchingMemberIds = householdUsers.value
      .filter(m => m.user_id && sharedUserIds.includes(m.user_id) && m.id !== form.value.paid_by_member_id)
      .map(m => m.id)
    if (matchingMemberIds.length > 0) {
      form.value.is_split = true
      form.value.split_with_member_ids = matchingMemberIds
    }
  }
})

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
      property_id: form.value.property_id,
      project_id: form.value.project_id,
      paid_by_member_id: form.value.paid_by_member_id,
      is_split: form.value.is_split,
      split_with_member_ids: form.value.is_split
        ? form.value.split_with_member_ids.filter(id => id !== form.value.paid_by_member_id)
        : []
    }

    await expensesStore.createExpense(expenseData)
    window.$toast?.success('Spesa creata con successo!')
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Errore durante il salvataggio'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  fetchCategories()
  await fetchCurrentProperty()
  await fetchHouseholdUsers()
  await fetchUserSettings()
  fetchActiveProjects()
})
</script>
