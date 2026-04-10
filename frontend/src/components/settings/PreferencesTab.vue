<template>
  <div class="space-y-4">
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Preferenze Personali</h2>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tema</label>
          <select
            v-model="preferences.theme"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="it">Italiano</option>
            <option value="en">English</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Formato Data</label>
          <select
            v-model="preferences.date_format"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="DD/MM/YYYY">GG/MM/AAAA (31/12/2024)</option>
            <option value="MM/DD/YYYY">MM/GG/AAAA (12/31/2024)</option>
            <option value="YYYY-MM-DD">AAAA-MM-GG (2024-12-31)</option>
            <option value="DD MMM YYYY">GG MMM AAAA (31 dic 2024)</option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Expense Templates -->
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white">Modelli Spesa</h2>
        <button
          type="button"
          @click="showAddTemplate = true"
          class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
        >
          + Nuovo
        </button>
      </div>

      <div v-if="templatesLoading" class="text-sm text-gray-500 py-4 text-center">
        Caricamento...
      </div>

      <div v-else-if="templates.length === 0" class="text-sm text-gray-500 dark:text-gray-400 py-4 text-center">
        Nessun modello salvato. Puoi crearne uno qui o dal form "Nuova Spesa".
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700"
        >
          <span class="text-xl flex-shrink-0">{{ tpl.icon || tpl.category?.icon || '📋' }}</span>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ tpl.name }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ tpl.category?.name }}<span v-if="tpl.subcategory"> / {{ tpl.subcategory.name }}</span>
              <span v-if="tpl.amount"> · {{ formatCurrency(tpl.amount) }}</span>
            </div>
          </div>
          <button
            type="button"
            @click="editTemplate(tpl)"
            class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-600 text-gray-400 hover:text-blue-500"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            type="button"
            @click="deleteTemplate(tpl)"
            class="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-500"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Inline Add Template Form -->
      <div v-if="showAddTemplate" class="mt-4 p-4 border border-blue-200 dark:border-blue-800 rounded-lg bg-blue-50/50 dark:bg-blue-900/10 space-y-3">
        <Input
          v-model="tplForm.name"
          label="Nome modello *"
          placeholder="es. Spesa Esselunga"
          required
        />
        <div class="grid grid-cols-2 gap-3">
          <Input
            v-model.number="tplForm.amount"
            label="Importo (0 = chiedi)"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            inputmode="decimal"
          />
          <div>
            <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Categoria *</label>
            <select
              v-model.number="tplForm.category_id"
              @change="tplForm.subcategory_id = null"
              required
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="cat in tplCategories" :key="cat.id" :value="cat.id">
                {{ cat.icon }} {{ cat.name }}
              </option>
            </select>
          </div>
        </div>
        <div v-if="tplSubcategories.length > 0">
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Sottocategoria (opzionale)</label>
          <select
            v-model.number="tplForm.subcategory_id"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null">Nessuna sottocategoria</option>
            <option v-for="sub in tplSubcategories" :key="sub.id" :value="sub.id">
              {{ sub.name }}
            </option>
          </select>
        </div>
        <div class="flex gap-2">
          <Button size="sm" @click="handleSaveTemplate" :disabled="!tplForm.name || !tplForm.category_id">
            {{ editingTemplateId ? 'Aggiorna' : 'Salva' }}
          </Button>
          <Button size="sm" variant="secondary" @click="cancelTemplateForm">
            Annulla
          </Button>
        </div>
      </div>
    </Card>

    <!-- Notifications -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Notifiche</h2>
      <div class="space-y-4">
        <div class="space-y-3 mb-2">
          <p class="text-sm text-gray-600 dark:text-gray-400">Scegli quali notifiche ricevere.</p>
          <label class="flex items-center justify-between cursor-pointer">
            <span class="text-sm text-gray-700 dark:text-gray-300">Richieste di accesso</span>
            <input type="checkbox" v-model="notifyJoinRequests" @change="updateNotificationPrefs"
                   class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500" />
          </label>
          <label class="flex items-center justify-between cursor-pointer">
            <span class="text-sm text-gray-700 dark:text-gray-300">Spese condivise</span>
            <input type="checkbox" v-model="notifySharedExpenses" @change="updateNotificationPrefs"
                   class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500" />
          </label>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Conserva notifiche per (giorni)
          </label>
          <select
            v-model.number="notificationRetentionDays"
            @change="updateRetentionDays"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="30">30 giorni</option>
            <option :value="60">60 giorni</option>
            <option :value="90">90 giorni</option>
            <option :value="180">180 giorni</option>
            <option :value="365">1 anno</option>
          </select>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Le notifiche più vecchie verranno eliminate automaticamente.
          </p>
        </div>
      </div>
    </Card>

    <!-- Account -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Account</h2>
      <div class="space-y-3">
        <!-- Change Password toggle -->
        <div>
          <button
            type="button"
            @click="showChangePassword = !showChangePassword; pwError = null; pwSuccess = null"
            class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
          >
            {{ showChangePassword ? '✕ Annulla' : 'Cambia password' }}
          </button>

          <div v-if="showChangePassword" class="mt-4 space-y-3">
            <Input
              v-model="pwForm.current"
              label="Password attuale"
              type="password"
              placeholder="Password attuale"
              autocomplete="current-password"
            />
            <Input
              v-model="pwForm.newPw"
              label="Nuova password"
              type="password"
              placeholder="Minimo 6 caratteri"
              autocomplete="new-password"
            />
            <Input
              v-model="pwForm.confirm"
              label="Conferma nuova password"
              type="password"
              placeholder="Ripeti la nuova password"
              autocomplete="new-password"
            />

            <div v-if="pwError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
              {{ pwError }}
            </div>
            <div v-if="pwSuccess" class="text-green-700 text-sm bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
              {{ pwSuccess }}
            </div>

            <Button :disabled="pwLoading" @click="handleChangePassword">
              {{ pwLoading ? 'Salvataggio...' : 'Aggiorna password' }}
            </Button>
          </div>
        </div>

        <Button variant="danger" @click="handleLogout">
          Esci dall'account
        </Button>
      </div>
    </Card>

    <!-- Danger Zone: Delete Account -->
    <Card class="p-6 border-red-200 dark:border-red-800">
      <h2 class="text-xl font-bold text-red-600 dark:text-red-400 mb-2">Zona Pericolosa</h2>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Questa azione è irreversibile. Tutti i tuoi dati personali verranno eliminati permanentemente.
      </p>

      <!-- Step 0: Initial button -->
      <div v-if="deleteStep === 0">
        <Button variant="danger" @click="startDeleteAccount">
          Elimina il mio account
        </Button>
      </div>

      <!-- Loading state -->
      <div v-else-if="deleteStep === 'loading'" class="text-sm text-gray-500 py-4 text-center">
        Controllo in corso...
      </div>

      <!-- Step 1: Blocking — must nominate admins -->
      <div v-else-if="deleteStep === 'blocking'" class="space-y-4">
        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg">
          <p class="text-sm font-medium text-amber-800 dark:text-amber-300 mb-2">
            Sei l'unico amministratore di queste proprietà. Devi nominare un nuovo admin prima di poter eliminare il tuo account:
          </p>
        </div>

        <div v-for="bp in deleteCheckResult.blocking_properties" :key="bp.property_id"
             class="p-3 border border-gray-200 dark:border-gray-700 rounded-lg space-y-2">
          <div class="font-medium text-gray-900 dark:text-white">{{ bp.property_name }}</div>
          <div class="flex items-center gap-2">
            <select
              v-model="adminNominations[bp.property_id]"
              class="flex-1 px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option :value="null" disabled>Seleziona membro...</option>
              <option v-for="m in bp.members" :key="m.member_id" :value="m.member_id">
                {{ m.name }}
              </option>
            </select>
            <Button
              size="sm"
              :disabled="!adminNominations[bp.property_id] || promotingProperty === bp.property_id"
              @click="handlePromoteAdmin(bp.property_id)"
            >
              {{ promotingProperty === bp.property_id ? 'Salvataggio...' : 'Nomina' }}
            </Button>
          </div>
        </div>

        <div class="flex gap-2">
          <Button variant="secondary" size="sm" @click="deleteStep = 0">Annulla</Button>
        </div>
      </div>

      <!-- Step 2: Can delete — confirm -->
      <div v-else-if="deleteStep === 'confirm'" class="space-y-4">
        <div v-if="deleteCheckResult.data_loss_properties?.length > 0"
             class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg">
          <p class="text-sm font-medium text-red-800 dark:text-red-300 mb-1">
            Le seguenti proprietà e tutti i loro dati verranno eliminati definitivamente:
          </p>
          <ul class="text-sm text-red-700 dark:text-red-400 list-disc list-inside">
            <li v-for="name in deleteCheckResult.data_loss_properties" :key="name">{{ name }}</li>
          </ul>
        </div>

        <Input
          v-model="deletePassword"
          label="Inserisci la tua password per confermare"
          type="password"
          placeholder="Password"
          autocomplete="current-password"
        />

        <label class="flex items-start gap-2 cursor-pointer">
          <input v-model="deleteConfirmed" type="checkbox" class="mt-1 rounded border-gray-300 dark:border-gray-600" />
          <span class="text-sm text-gray-700 dark:text-gray-300">
            Confermo di voler eliminare il mio account e tutti i dati associati in modo irreversibile.
          </span>
        </label>

        <div v-if="deleteError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ deleteError }}
        </div>

        <div class="flex gap-2">
          <Button
            variant="danger"
            :disabled="!deletePassword || !deleteConfirmed || deleteLoading"
            @click="handleDeleteAccount"
          >
            {{ deleteLoading ? 'Eliminazione...' : 'Elimina definitivamente' }}
          </Button>
          <Button variant="secondary" @click="deleteStep = 0">Annulla</Button>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'PreferencesTab' })

import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useDarkMode } from '@/composables/useDarkMode'
import { authAPI, accountAPI, expenseTemplatesAPI, categoriesAPI } from '@/api/client'
import apiClient from '@/api/client'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { setTheme } = useDarkMode()

const preferences = ref({
  theme: 'auto',
  currency: 'EUR',
  language: 'it',
  date_format: 'DD/MM/YYYY'
})

const notificationRetentionDays = ref(90)
const notifyJoinRequests = ref(true)
const notifySharedExpenses = ref(true)

// Expense templates
const templates = ref([])
const templatesLoading = ref(false)
const showAddTemplate = ref(false)
const editingTemplateId = ref(null)
const tplCategories = ref([])
const tplForm = ref({ name: '', amount: 0, category_id: null, subcategory_id: null })

const tplSubcategories = computed(() => {
  if (!tplForm.value.category_id) return []
  const cat = tplCategories.value.find(c => c.id === tplForm.value.category_id)
  return cat?.subcategories || []
})

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

async function loadTemplates() {
  templatesLoading.value = true
  try {
    const { data } = await expenseTemplatesAPI.list()
    templates.value = data || []
  } catch (err) {
    console.error('Error loading templates:', err)
  } finally {
    templatesLoading.value = false
  }
}

async function loadCategories() {
  try {
    const { data } = await categoriesAPI.list()
    tplCategories.value = data || []
    if (!tplForm.value.category_id && data.length > 0) {
      tplForm.value.category_id = data[0].id
    }
  } catch (err) {
    console.error('Error loading categories:', err)
  }
}

function editTemplate(tpl) {
  editingTemplateId.value = tpl.id
  tplForm.value = {
    name: tpl.name,
    amount: tpl.amount || 0,
    category_id: tpl.category_id,
    subcategory_id: tpl.subcategory_id || null,
  }
  showAddTemplate.value = true
}

function cancelTemplateForm() {
  showAddTemplate.value = false
  editingTemplateId.value = null
  tplForm.value = { name: '', amount: 0, category_id: tplCategories.value[0]?.id, subcategory_id: null }
}

async function handleSaveTemplate() {
  try {
    const payload = {
      name: tplForm.value.name,
      amount: parseFloat(tplForm.value.amount) || 0,
      category_id: tplForm.value.category_id,
    }
    if (tplForm.value.subcategory_id) payload.subcategory_id = tplForm.value.subcategory_id

    if (editingTemplateId.value) {
      const { data } = await expenseTemplatesAPI.update(editingTemplateId.value, payload)
      const idx = templates.value.findIndex(t => t.id === editingTemplateId.value)
      if (idx !== -1) templates.value[idx] = data
      window.$toast?.success('Modello aggiornato!')
    } else {
      const { data } = await expenseTemplatesAPI.create(payload)
      templates.value.push(data)
      window.$toast?.success('Modello creato!')
    }
    cancelTemplateForm()
  } catch {
    window.$toast?.error(editingTemplateId.value ? 'Errore nell\'aggiornamento' : 'Errore nella creazione del modello')
  }
}

async function deleteTemplate(tpl) {
  try {
    await expenseTemplatesAPI.delete(tpl.id)
    templates.value = templates.value.filter(t => t.id !== tpl.id)
    window.$toast?.success('Modello eliminato')
  } catch {
    window.$toast?.error('Errore nell\'eliminazione')
  }
}

// Change password
const showChangePassword = ref(false)
const pwLoading = ref(false)
const pwError = ref(null)
const pwSuccess = ref(null)
const pwForm = ref({ current: '', newPw: '', confirm: '' })

async function loadUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    preferences.value = {
      theme: data.theme || 'auto',
      currency: data.currency || 'EUR',
      language: data.language || 'it',
      date_format: data.date_format || 'DD/MM/YYYY'
    }
    notificationRetentionDays.value = data.notification_retention_days ?? 90
    notifyJoinRequests.value = data.notify_join_requests ?? true
    notifySharedExpenses.value = data.notify_shared_expenses ?? true
    // Sync theme composable with server value
    setTheme(preferences.value.theme)
  } catch {
    console.log('Using default user settings')
  }
}

async function updateRetentionDays() {
  try {
    await apiClient.put('/settings', { notification_retention_days: notificationRetentionDays.value })
    settingsStore.notificationRetentionDays = notificationRetentionDays.value
  } catch (err) {
    console.error('Error updating retention days:', err)
  }
}

async function updateNotificationPrefs() {
  try {
    await apiClient.put('/settings', {
      notify_join_requests: notifyJoinRequests.value,
      notify_shared_expenses: notifySharedExpenses.value,
    })
    settingsStore.notifyJoinRequests = notifyJoinRequests.value
    settingsStore.notifySharedExpenses = notifySharedExpenses.value
  } catch (err) {
    console.error('Error updating notification preferences:', err)
  }
}

async function updateUserSettings() {
  try {
    // Get current settings to preserve split member IDs
    const { data: currentSettings } = await apiClient.get('/settings')
    const payload = {
      theme: preferences.value.theme,
      currency: preferences.value.currency,
      language: preferences.value.language,
      date_format: preferences.value.date_format,
      default_split_with_member_ids: currentSettings.default_split_with_member_ids || '[]'
    }
    await apiClient.put('/settings', payload)
    settingsStore.theme = preferences.value.theme
    setTheme(preferences.value.theme)
    settingsStore.currency = preferences.value.currency
    settingsStore.language = preferences.value.language
    settingsStore.dateFormat = preferences.value.date_format
  } catch (err) {
    console.error('Error updating user settings:', err)
  }
}

async function handleChangePassword() {
  pwError.value = null
  pwSuccess.value = null

  if (!pwForm.value.current) {
    pwError.value = 'Inserisci la password attuale.'
    return
  }
  if (pwForm.value.newPw.length < 6) {
    pwError.value = 'La nuova password deve avere almeno 6 caratteri.'
    return
  }
  if (pwForm.value.newPw !== pwForm.value.confirm) {
    pwError.value = 'Le due password non coincidono.'
    return
  }

  pwLoading.value = true
  try {
    await authAPI.changePassword(pwForm.value.current, pwForm.value.newPw)
    pwSuccess.value = 'Password aggiornata con successo!'
    pwForm.value = { current: '', newPw: '', confirm: '' }
    setTimeout(() => { showChangePassword.value = false; pwSuccess.value = null }, 2000)
  } catch (err) {
    pwError.value = err.response?.data?.error || 'Errore durante il cambio password.'
  } finally {
    pwLoading.value = false
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

// Account deletion
const deleteStep = ref(0) // 0, 'loading', 'blocking', 'confirm'
const deleteCheckResult = ref(null)
const adminNominations = ref({})
const promotingProperty = ref(null)
const deletePassword = ref('')
const deleteConfirmed = ref(false)
const deleteError = ref(null)
const deleteLoading = ref(false)

async function startDeleteAccount() {
  deleteStep.value = 'loading'
  deleteError.value = null
  try {
    const { data } = await accountAPI.deleteCheck()
    deleteCheckResult.value = data
    if (!data.can_delete) {
      // Initialize nomination selections
      adminNominations.value = {}
      for (const bp of data.blocking_properties) {
        adminNominations.value[bp.property_id] = null
      }
      deleteStep.value = 'blocking'
    } else {
      deleteStep.value = 'confirm'
    }
  } catch {
    deleteStep.value = 0
    window.$toast?.error('Errore nel controllo account')
  }
}

async function handlePromoteAdmin(propertyId) {
  const memberId = adminNominations.value[propertyId]
  if (!memberId) return
  promotingProperty.value = propertyId
  try {
    await accountAPI.promoteAdmin(propertyId, memberId)
    window.$toast?.success('Admin nominato con successo')
    // Re-check deletion status
    await startDeleteAccount()
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || 'Errore nella nomina admin')
  } finally {
    promotingProperty.value = null
  }
}

async function handleDeleteAccount() {
  deleteError.value = null
  deleteLoading.value = true
  try {
    await accountAPI.deleteAccount(deletePassword.value)
    authStore.logout()
    router.push('/login')
    window.$toast?.success('Account eliminato con successo')
  } catch (err) {
    deleteError.value = err.response?.data?.error || 'Errore durante l\'eliminazione dell\'account'
  } finally {
    deleteLoading.value = false
  }
}

onMounted(() => {
  loadUserSettings()
  loadTemplates()
  loadCategories()
})
</script>
