<template>
  <div class="space-y-4">
    <!-- Pending Join Requests (admin only) -->
    <Card v-if="isAdmin && pendingRequests.length > 0" class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Richieste di Accesso</h2>
      <div class="space-y-3">
        <div
          v-for="req in pendingRequests"
          :key="req.id"
          class="flex items-center justify-between gap-3 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"
        >
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-10 h-10 rounded-full bg-yellow-100 dark:bg-yellow-900/40 flex items-center justify-center text-sm font-medium text-yellow-700 dark:text-yellow-300 flex-shrink-0">
              {{ getInitials(req.user?.name || '?') }}
            </div>
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white truncate">{{ req.user?.name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ req.user?.email }}</div>
            </div>
          </div>
          <div class="flex gap-2 flex-shrink-0">
            <Button size="sm" @click="resolveRequest(req.id, 'approved')" :disabled="resolvingRequest === req.id">
              Approva
            </Button>
            <Button size="sm" variant="secondary" @click="resolveRequest(req.id, 'rejected')" :disabled="resolvingRequest === req.id">
              Rifiuta
            </Button>
          </div>
        </div>
      </div>
    </Card>

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
          <label v-if="isAdmin" class="relative inline-flex items-center cursor-pointer ml-4">
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
          <span v-else class="ml-4 text-sm font-medium" :class="splitMode ? 'text-green-600 dark:text-green-400' : 'text-gray-400'">
            {{ splitMode ? 'Attivo' : 'Disattivo' }}
          </span>
        </div>

        <!-- Split Settings -->
        <div v-if="splitMode" class="pl-6 space-y-4 border-l-2 border-blue-200 dark:border-blue-800">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Dividi automaticamente le spese con:
            </label>

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
                  @change="emitUpdateUserSettings"
                  class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  <img
                    v-if="member.avatar_path"
                    :src="'/' + member.avatar_path"
                    :alt="member.name"
                    class="w-8 h-8 rounded-full object-cover flex-shrink-0"
                  />
                  <div
                    v-else
                    class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900
                              flex items-center justify-center text-sm font-medium
                              text-blue-600 dark:text-blue-300 flex-shrink-0"
                  >
                    {{ getInitials(member.name) }}
                  </div>
                  <span class="text-gray-900 dark:text-white truncate">{{ member.name }}</span>
                  <span v-if="member.is_virtual" class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">(virtuale)</span>
                  <span v-if="member.user_role === 'admin'" class="text-xs bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 px-1.5 py-0.5 rounded font-medium flex-shrink-0">Admin</span>
                </div>
                <div class="flex items-center gap-1 flex-shrink-0">
                  <!-- Admin: toggle admin role -->
                  <button
                    v-if="isAdmin && !member.is_virtual && member.user_id !== currentUserId"
                    @click="toggleAdminRole(member)"
                    class="p-2 transition-colors"
                    :class="member.user_role === 'admin' ? 'text-amber-500 hover:text-amber-700' : 'text-gray-400 hover:text-amber-500'"
                    :title="member.user_role === 'admin' ? 'Rimuovi ruolo admin' : 'Promuovi ad admin'"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  </button>
                  <!-- Admin: delete virtual member -->
                  <button
                    v-if="isAdmin && member.is_virtual"
                    @click="deleteMember(member.id)"
                    class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-2"
                    aria-label="Elimina membro"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                  <!-- Admin: delete user account -->
                  <button
                    v-if="isAdmin && !member.is_virtual && member.user_id !== currentUserId"
                    @click="deleteUserAccount(member)"
                    class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-2"
                    aria-label="Elimina account utente"
                    title="Elimina account e tutti i dati"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728l-12.728-12.728" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            <div v-if="householdMembers.length === 0" class="text-sm text-gray-600 dark:text-gray-400 italic p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              Nessun altro membro nella casa. Aggiungi un membro per dividere le spese.
            </div>

            <!-- Add new member (admin only) -->
            <div v-if="isAdmin" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div class="flex flex-col sm:flex-row gap-2">
                <input
                  v-model="newMemberName"
                  type="text"
                  placeholder="Nome membro"
                  class="flex-1 px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                         focus:outline-none focus:ring-2 focus:ring-blue-500"
                  @keyup.enter="addMember"
                />
                <Button @click="addMember" :disabled="!newMemberName.trim()">
                  Aggiungi
                </Button>
              </div>
            </div>

            <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
              Quando aggiungi una nuova spesa, sara automaticamente divisa con le persone selezionate.
            </p>
          </div>
        </div>

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
  </div>
</template>

<script setup>
defineOptions({ name: 'FamilyTab' })

import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { adminAPI, joinRequestAPI } from '@/api/client'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const isAdmin = computed(() => settingsStore.isPropertyAdmin)
const currentUserId = computed(() => authStore.user?.id)

const splitMode = computed({
  get: () => settingsStore.splitMode,
  set: (val) => { settingsStore.splitMode = val }
})
const currentPropertyId = computed(() => settingsStore.householdPropertyId)
const householdMembers = ref([])
const defaultSplitMemberIds = ref([])
const currentUserMemberId = ref(null)
const newMemberName = ref('')
const pendingRequests = ref([])
const resolvingRequest = ref(null)

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
    await settingsStore.loadHouseholdSettings()
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function fetchHouseholdMembers() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/members`)
    const userId = authStore.user?.id
    householdMembers.value = data.filter(m => m.user_id !== userId)

    const myMember = data.find(m => m.user_id === userId)
    if (myMember) {
      currentUserMemberId.value = myMember.id
    }
  } catch {
    console.log('Using empty members list')
    householdMembers.value = []
  }
}

async function loadUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    if (data.default_split_with_member_ids) {
      try {
        defaultSplitMemberIds.value = JSON.parse(data.default_split_with_member_ids)
      } catch {
        defaultSplitMemberIds.value = []
      }
    }
  } catch {
    console.log('Using default split settings')
  }
}

async function updateSplitMode() {
  try {
    await settingsStore.updateHouseholdSettings({
      split_mode: splitMode.value
    })
  } catch (err) {
    console.error('Error updating split mode:', err)
    settingsStore.splitMode = !settingsStore.splitMode
  }
}

async function updateUserSettings() {
  try {
    // We only update the split member IDs from this tab
    const { data } = await apiClient.get('/settings')
    const payload = {
      theme: data.theme || 'auto',
      currency: data.currency || 'EUR',
      language: data.language || 'it',
      date_format: data.date_format || 'DD/MM/YYYY',
      default_split_with_member_ids: JSON.stringify(defaultSplitMemberIds.value)
    }
    await apiClient.put('/settings', payload)
  } catch (err) {
    console.error('Error updating user settings:', err)
  }
}

function emitUpdateUserSettings() {
  updateUserSettings()
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
  } catch (err) {
    console.error('Error adding member:', err)
  }
}

async function deleteMember(memberId) {
  const ok = await confirm({
    title: 'Elimina membro',
    message: 'Sei sicuro di voler eliminare questo membro?',
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return

  try {
    await apiClient.delete(`/members/${memberId}`)
    defaultSplitMemberIds.value = defaultSplitMemberIds.value.filter(id => id !== memberId)
    await updateUserSettings()
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error deleting member:', err)
    if (err.response?.data?.error) {
      window.$toast?.error(err.response.data.error)
    }
  }
}

async function deleteUserAccount(member) {
  const ok = await confirm({
    title: 'Elimina account utente',
    message: `Eliminare definitivamente l'account di "${member.name}"? Tutti i dati associati (spese, utenze, bollette, letture, progetti, template) verranno cancellati in modo irreversibile.`,
    confirmText: 'Elimina definitivamente',
    variant: 'danger'
  })
  if (!ok) return

  try {
    await adminAPI.deleteUser(member.user_id)
    window.$toast?.success(`Account di "${member.name}" eliminato con successo`)
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error deleting user account:', err)
    window.$toast?.error(err.response?.data?.error || "Errore durante l'eliminazione dell'account")
  }
}

async function toggleAdminRole(member) {
  const newRole = member.user_role === 'admin' ? 'user' : 'admin'
  const action = newRole === 'admin' ? 'promuovere ad admin' : 'rimuovere il ruolo admin da'
  const ok = await confirm({
    title: newRole === 'admin' ? 'Promuovi ad admin' : 'Rimuovi ruolo admin',
    message: `Sei sicuro di voler ${action} "${member.name}"?`,
    confirmText: newRole === 'admin' ? 'Promuovi' : 'Rimuovi',
    variant: newRole === 'admin' ? 'primary' : 'danger'
  })
  if (!ok) return

  try {
    await adminAPI.setUserRole(member.user_id, newRole)
    window.$toast?.success(`Ruolo di "${member.name}" aggiornato a ${newRole}`)
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error toggling admin role:', err)
    window.$toast?.error(err.response?.data?.error || 'Errore durante il cambio ruolo')
  }
}

async function fetchPendingRequests() {
  if (!isAdmin.value) return
  try {
    const { data } = await joinRequestAPI.list()
    pendingRequests.value = (data || []).filter(r => r.status === 'pending')
  } catch {
    pendingRequests.value = []
  }
}

async function resolveRequest(requestId, status) {
  resolvingRequest.value = requestId
  try {
    await joinRequestAPI.resolve(requestId, status)
    window.$toast?.success(status === 'approved' ? 'Richiesta approvata!' : 'Richiesta rifiutata')
    await fetchPendingRequests()
    await fetchHouseholdMembers()
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || 'Errore durante la risoluzione della richiesta')
  } finally {
    resolvingRequest.value = null
  }
}

onMounted(() => {
  fetchCurrentProperty()
  loadUserSettings()
  fetchPendingRequests()
})
</script>
