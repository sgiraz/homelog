<template>
  <BaseModal title="Modifica Progetto" size="2xl" @close="$emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Name -->
      <Input
        v-model="form.name"
        label="Nome Progetto *"
        placeholder="es. Ristrutturazione Bagno"
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
          autocorrect="off"
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
        inputmode="decimal"
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
          class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="planned">Pianificato</option>
          <option value="active">In Corso</option>
          <option value="completed">Completato</option>
          <option value="cancelled">Annullato</option>
        </select>
      </div>

      <!-- Share with household members -->
      <div v-if="otherMembers.length > 0" class="border-t border-gray-200 dark:border-gray-700 pt-4">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Condividi con
        </label>
        <div class="space-y-2">
          <div
            v-for="member in otherMembers"
            :key="member.id"
            class="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <label class="flex items-center gap-3 cursor-pointer flex-1 min-w-0">
              <input
                type="checkbox"
                :checked="isMemberSelected(member.user_id)"
                @change="toggleMember(member.user_id)"
                class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500 flex-shrink-0"
              />
              <div class="w-7 h-7 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center text-xs font-medium text-purple-700 dark:text-purple-300 flex-shrink-0">
                {{ getInitials(member.name) }}
              </div>
              <span class="text-sm text-gray-900 dark:text-white truncate">{{ member.name }}</span>
            </label>
            <select
              v-if="isMemberSelected(member.user_id)"
              :value="getMemberRole(member.user_id)"
              @change="setMemberRole(member.user_id, $event.target.value)"
              class="text-xs px-2 py-1 rounded border border-gray-200 dark:border-gray-600
                     bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300 flex-shrink-0"
            >
              <option value="member">Membro</option>
              <option value="owner">Co-proprietario</option>
            </select>
          </div>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
          I membri possono aggiungere spese. I co-proprietari possono anche modificare il progetto.
        </p>
      </div>

      <!-- Error -->
      <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ error }}
      </div>

      <!-- Actions -->
      <div class="flex gap-3 pt-2">
        <Button type="button" variant="secondary" @click="$emit('close')" class="flex-1">
          Annulla
        </Button>
        <Button type="submit" :disabled="loading" class="flex-1">
          {{ loading ? 'Salvataggio...' : 'Salva Modifiche' }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useProjectsStore } from '@/stores/projects'
import { useAuthStore } from '@/stores/auth'
import apiClient from '@/api/client'
import BaseModal from '@/components/common/BaseModal.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  project: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'updated'])
const projectsStore = useProjectsStore()
const authStore = useAuthStore()

const loading = ref(false)
const error = ref(null)
const householdMembers = ref([])

const form = ref({
  name: '',
  icon: '🏗️',
  description: '',
  budget: null,
  start_date: '',
  end_date: '',
  status: 'planned',
  members: []
})

const icons = ['🏗️', '🔨', '🎨', '🛠️', '🏠', '🚪', '🪟', '💡', '🔌', '🚿', '🛏️', '🍽️', '🌳', '🏊', '🎉', '💍', '✈️', '🎓']

const otherMembers = computed(() =>
  householdMembers.value.filter(m => m.user_id && m.user_id !== authStore.user?.id)
)

function getInitials(name) {
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
}

function isMemberSelected(userId) {
  return form.value.members.some(m => m.user_id === userId)
}

function getMemberRole(userId) {
  return form.value.members.find(m => m.user_id === userId)?.role || 'member'
}

function toggleMember(userId) {
  const idx = form.value.members.findIndex(m => m.user_id === userId)
  if (idx >= 0) {
    form.value.members.splice(idx, 1)
  } else {
    form.value.members.push({ user_id: userId, role: 'member' })
  }
}

function setMemberRole(userId, role) {
  const member = form.value.members.find(m => m.user_id === userId)
  if (member) member.role = role
}

async function fetchHouseholdMembers() {
  try {
    const propertyId = props.project.property_id
    if (!propertyId) return
    const { data } = await apiClient.get(`/properties/${propertyId}/members`)
    householdMembers.value = data || []
  } catch (err) {
    console.error('Error fetching members:', err)
  }
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  if (new Date(form.value.end_date) < new Date(form.value.start_date)) {
    error.value = 'La data di fine deve essere successiva alla data di inizio'
    loading.value = false
    return
  }

  try {
    await projectsStore.updateProject(props.project.id, form.value)
    window.$toast?.success('Progetto aggiornato!')
    emit('updated')
    emit('close')
  } catch (err) {
    error.value = err.response?.data?.error || err.message
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // Build members array from project.members (new format) or shared_with (backward compat)
  let members = []
  if (props.project.members?.length > 0) {
    members = props.project.members
      .filter(m => m.role !== 'creator')
      .map(m => ({ user_id: m.id, role: m.role }))
  } else if (props.project.shared_with?.length > 0) {
    members = props.project.shared_with.map(u => ({ user_id: u.id, role: 'member' }))
  }

  form.value = {
    name: props.project.name,
    icon: props.project.icon || '🏗️',
    description: props.project.description || '',
    budget: props.project.budget,
    start_date: props.project.start_date ? props.project.start_date.split('T')[0] : '',
    end_date: props.project.end_date ? props.project.end_date.split('T')[0] : '',
    status: props.project.status,
    members
  }
  await fetchHouseholdMembers()
})
</script>
