import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api/client'

export const useSettingsStore = defineStore('settings', () => {
  // State
  const theme = ref('auto')
  const currency = ref('EUR')
  const language = ref('it')
  const dateFormat = ref('DD/MM/YYYY')
  const defaultSplitWithMemberIds = ref([])
  const defaultTemplates = ref({})
  const emailNotifications = ref(true)
  const billReminders = ref(true)
  const loaded = ref(false)

  // Getters - provides { date_format, language } object for formatDate/formatPeriod
  const dateSettings = computed(() => ({
    date_format: dateFormat.value,
    language: language.value
  }))

  // Actions
  async function loadSettings() {
    try {
      const { data } = await apiClient.get('/settings')
      theme.value = data.theme || 'auto'
      currency.value = data.currency || 'EUR'
      language.value = data.language || 'it'
      dateFormat.value = data.date_format || 'DD/MM/YYYY'
      emailNotifications.value = data.email_notifications ?? true
      billReminders.value = data.bill_reminders ?? true

      if (data.default_split_with_member_ids) {
        try {
          defaultSplitWithMemberIds.value = JSON.parse(data.default_split_with_member_ids)
        } catch (e) {
          defaultSplitWithMemberIds.value = []
        }
      }

      if (data.default_templates) {
        try {
          defaultTemplates.value = JSON.parse(data.default_templates)
        } catch (e) {
          defaultTemplates.value = {}
        }
      }

      loaded.value = true
    } catch (err) {
      console.error('Error loading settings:', err)
      // Keep defaults
      loaded.value = true
    }
  }

  async function updateSettings(updates) {
    try {
      const payload = { ...updates }

      // Apply updates locally
      if (payload.theme !== undefined) theme.value = payload.theme
      if (payload.currency !== undefined) currency.value = payload.currency
      if (payload.language !== undefined) language.value = payload.language
      if (payload.date_format !== undefined) dateFormat.value = payload.date_format

      // Serialize arrays/objects for API
      if (payload.default_split_with_member_ids && typeof payload.default_split_with_member_ids !== 'string') {
        payload.default_split_with_member_ids = JSON.stringify(payload.default_split_with_member_ids)
      }
      if (payload.default_templates && typeof payload.default_templates !== 'string') {
        payload.default_templates = JSON.stringify(payload.default_templates)
      }

      await apiClient.put('/settings', payload)
    } catch (err) {
      console.error('Error updating settings:', err)
      throw err
    }
  }

  function $reset() {
    theme.value = 'auto'
    currency.value = 'EUR'
    language.value = 'it'
    dateFormat.value = 'DD/MM/YYYY'
    defaultSplitWithMemberIds.value = []
    defaultTemplates.value = {}
    emailNotifications.value = true
    billReminders.value = true
    loaded.value = false
  }

  return {
    // State
    theme,
    currency,
    language,
    dateFormat,
    defaultSplitWithMemberIds,
    defaultTemplates,
    emailNotifications,
    billReminders,
    loaded,
    // Getters
    dateSettings,
    // Actions
    loadSettings,
    updateSettings,
    $reset
  }
})
