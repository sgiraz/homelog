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

    <!-- Notifications -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Notifiche</h2>
      <div class="space-y-4">
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
  </div>
</template>

<script setup>
defineOptions({ name: 'PreferencesTab' })

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useDarkMode } from '@/composables/useDarkMode'
import { authAPI } from '@/api/client'
import apiClient from '@/api/client'
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
    // Sync theme composable with server value
    setTheme(preferences.value.theme)
  } catch (err) {
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

onMounted(() => {
  loadUserSettings()
})
</script>
