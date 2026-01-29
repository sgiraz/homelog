<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Impostazioni</h1>
      <p class="text-gray-600 dark:text-gray-400 mt-1">Configura le preferenze dell'app</p>
    </div>

    <!-- Household Settings -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Impostazioni Famiglia</h2>

      <div class="space-y-4">
        <!-- Split Mode Toggle -->
        <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-xl">
          <div class="flex-1">
            <div class="font-medium text-gray-900 dark:text-white">Modalita Split</div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Traccia chi deve cosa dividendo le spese tra i membri della famiglia
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer ml-4">
            <input
              type="checkbox"
              v-model="splitMode"
              @change="updateSplitMode"
              class="sr-only peer"
            />
            <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4
                        peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer
                        dark:bg-gray-600 peer-checked:after:translate-x-full
                        peer-checked:after:border-white after:content-[''] after:absolute
                        after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300
                        after:border after:rounded-full after:h-5 after:w-5 after:transition-all
                        dark:border-gray-500 peer-checked:bg-blue-600">
            </div>
          </label>
        </div>

        <!-- Split Settings (only if Split Mode ON) -->
        <div v-if="splitMode" class="pl-6 space-y-4 border-l-2 border-blue-200 dark:border-blue-800">
          <!-- Default split with users -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Dividi automaticamente le spese con:
            </label>

            <!-- User list -->
            <div v-if="householdMembers.length > 0" class="space-y-2">
              <div
                v-for="member in householdMembers"
                :key="member.id"
                class="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg
                       border border-gray-200 dark:border-gray-600 hover:bg-gray-50
                       dark:hover:bg-gray-700 transition-colors"
              >
                <input
                  type="checkbox"
                  :value="member.id"
                  v-model="defaultSplitMemberIds"
                  @change="updateUserSettings"
                  class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex items-center gap-2 flex-1">
                  <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900
                              flex items-center justify-center text-sm font-medium
                              text-blue-600 dark:text-blue-300">
                    {{ getInitials(member.name) }}
                  </div>
                  <span class="text-gray-900 dark:text-white">{{ member.name }}</span>
                  <span v-if="member.is_virtual" class="text-xs text-gray-500 dark:text-gray-400">(virtuale)</span>
                </div>
                <button
                  v-if="member.is_virtual"
                  @click="deleteMember(member.id)"
                  class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-1"
                  title="Elimina membro"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- No members message -->
            <div v-if="householdMembers.length === 0" class="text-sm text-gray-600 dark:text-gray-400 italic p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              Nessun altro membro nella casa. Aggiungi un membro per dividere le spese.
            </div>

            <!-- Add new member -->
            <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div class="flex gap-2">
                <input
                  v-model="newMemberName"
                  type="text"
                  placeholder="Nome nuovo membro (es: Partner)"
                  class="flex-1 px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                  @keyup.enter="addMember"
                />
                <Button @click="addMember" :disabled="!newMemberName.trim()">
                  Aggiungi
                </Button>
              </div>
            </div>

            <!-- Hint -->
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
              Quando aggiungi una nuova spesa, sara automaticamente divisa con le persone selezionate.
              Puoi sempre modificare per singola spesa.
            </p>
          </div>
        </div>

        <!-- Info Split Mode -->
        <div v-if="splitMode" class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <div class="text-sm text-gray-700 dark:text-gray-300">
            <div class="font-medium mb-2">Split Mode Attivo</div>
            <ul class="list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400">
              <li>Ogni spesa puo essere divisa tra membri della famiglia</li>
              <li>Il sistema traccia chi deve cosa</li>
              <li>Puoi saldare i conti dalla pagina Bilancio</li>
            </ul>
          </div>
        </div>
      </div>
    </Card>

    <!-- User Preferences -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Preferenze Personali</h2>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tema</label>
          <select
            v-model="preferences.theme"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="light">Chiaro</option>
            <option value="dark">Scuro</option>
            <option value="auto">Automatico</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Valuta</label>
          <select
            v-model="preferences.currency"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="EUR">EUR (Euro)</option>
            <option value="USD">USD (Dollaro)</option>
            <option value="GBP">GBP (Sterlina)</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Lingua</label>
          <select
            v-model="preferences.language"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="it">Italiano</option>
            <option value="en">English</option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Account Info -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Account</h2>

      <div class="flex items-center gap-4">
        <div class="w-16 h-16 rounded-full bg-gradient-to-br from-blue-500 to-purple-600
                    flex items-center justify-center text-white text-xl font-bold">
          {{ userInitials }}
        </div>
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ authStore.user?.name }}</div>
          <div class="text-sm text-gray-600 dark:text-gray-400">{{ authStore.user?.email }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-500 mt-1">
            {{ authStore.user?.role === 'admin' ? 'Amministratore' : 'Utente' }}
          </div>
        </div>
      </div>

      <div class="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
        <Button variant="danger" @click="handleLogout">
          Esci dall'account
        </Button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import apiClient from '@/api/client'

const router = useRouter()
const authStore = useAuthStore()

const splitMode = ref(false)
const currentPropertyId = ref(null)
const householdMembers = ref([])
const defaultSplitMemberIds = ref([])
const currentUserMemberId = ref(null)
const newMemberName = ref('')

const preferences = ref({
  theme: 'auto',
  currency: 'EUR',
  language: 'it'
})

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function getInitials(name) {
  return name
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
      console.log('Current property ID:', currentProp.id)
      await loadHouseholdSettings()
      await fetchHouseholdMembers()
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function fetchHouseholdMembers() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/members`)
    // Filter out current user's member from the list (only show other members)
    const currentUserId = authStore.user?.id
    householdMembers.value = data.filter(m => m.user_id !== currentUserId)

    // Find current user's member ID
    const myMember = data.find(m => m.user_id === currentUserId)
    if (myMember) {
      currentUserMemberId.value = myMember.id
    }

    console.log('Household members:', householdMembers.value)
    console.log('Current user member ID:', currentUserMemberId.value)
  } catch (err) {
    console.log('Using empty members list')
    householdMembers.value = []
  }
}

async function loadHouseholdSettings() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/settings`)
    splitMode.value = data.split_mode || false
  } catch (err) {
    console.log('Using default household settings')
    splitMode.value = false
  }
}

async function loadUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    preferences.value = {
      theme: data.theme || 'auto',
      currency: data.currency || 'EUR',
      language: data.language || 'it'
    }

    // Parse default split member IDs
    if (data.default_split_with_member_ids) {
      try {
        defaultSplitMemberIds.value = JSON.parse(data.default_split_with_member_ids)
      } catch (e) {
        defaultSplitMemberIds.value = []
      }
    }
  } catch (err) {
    console.log('Using default user settings')
  }
}

async function updateSplitMode() {
  if (!currentPropertyId.value) {
    console.error('No property selected')
    return
  }

  try {
    await apiClient.put(`/properties/${currentPropertyId.value}/settings`, {
      split_mode: splitMode.value
    })
    console.log('Split mode updated:', splitMode.value)
  } catch (err) {
    console.error('Error updating split mode:', err)
    splitMode.value = !splitMode.value
  }
}

async function updateUserSettings() {
  try {
    const payload = {
      theme: preferences.value.theme,
      currency: preferences.value.currency,
      language: preferences.value.language,
      default_split_with_member_ids: JSON.stringify(defaultSplitMemberIds.value)
    }
    await apiClient.put('/settings', payload)
    console.log('User settings updated:', payload)
  } catch (err) {
    console.error('Error updating user settings:', err)
  }
}

async function addMember() {
  if (!newMemberName.value.trim() || !currentPropertyId.value) return

  try {
    await apiClient.post(`/properties/${currentPropertyId.value}/members`, {
      name: newMemberName.value.trim(),
      role: ''
    })
    newMemberName.value = ''
    await fetchHouseholdMembers()
    console.log('Member added successfully')
  } catch (err) {
    console.error('Error adding member:', err)
  }
}

async function deleteMember(memberId) {
  if (!confirm('Sei sicuro di voler eliminare questo membro?')) return

  try {
    await apiClient.delete(`/members/${memberId}`)
    // Remove from default split if present
    defaultSplitMemberIds.value = defaultSplitMemberIds.value.filter(id => id !== memberId)
    await updateUserSettings()
    await fetchHouseholdMembers()
    console.log('Member deleted successfully')
  } catch (err) {
    console.error('Error deleting member:', err)
    if (err.response?.data?.error) {
      alert(err.response.data.error)
    }
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

onMounted(() => {
  loadUserSettings()
  fetchCurrentProperty()
})
</script>
