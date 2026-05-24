import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api/client'
import { useDarkMode } from '@/composables/useDarkMode'
import { useTheme } from '@/composables/useTheme'

export const useSettingsStore = defineStore('settings', () => {
  // State
  const theme = ref('auto')
  const colorTheme = ref('paper')
  const currency = ref('EUR')
  const language = ref('it')
  const dateFormat = ref('DD/MM/YYYY')
  const defaultSplitWithMemberIds = ref([])
  const defaultTemplates = ref({})
  const emailNotifications = ref(true)
  const billReminders = ref(true)
  const notificationRetentionDays = ref(90)
  const notifyJoinRequests = ref(true)
  const notifySharedExpenses = ref(true)
  const loaded = ref(false)
  const onboardingCompleted = ref(false)
  const hasProperty = ref(false)
  const isPropertyAdmin = ref(false)
  const pendingJoinRequest = ref(null)

  // Household settings (shared across all members of the property)
  const splitMode = ref(false)
  const householdPropertyId = ref(null)

  // Getters - provides { date_format, language } object for formatDate/formatPeriod
  const dateSettings = computed(() => ({
    date_format: dateFormat.value,
    language: language.value
  }))

  // Includes currency for formatCurrency/formatNumber
  const formatSettings = computed(() => ({
    date_format: dateFormat.value,
    language: language.value,
    currency: currency.value
  }))

  // Actions
  async function loadSettings() {
    const { setTheme } = useDarkMode()
    const { setColorTheme } = useTheme()
    try {
      const { data } = await apiClient.get('/settings')
      theme.value = data.theme || 'auto'
      colorTheme.value = data.color_theme || 'paper'
      currency.value = data.currency || 'EUR'
      language.value = data.language || 'it'
      dateFormat.value = data.date_format || 'DD/MM/YYYY'
      emailNotifications.value = data.email_notifications ?? true
      billReminders.value = data.bill_reminders ?? true
      notificationRetentionDays.value = data.notification_retention_days ?? 90
      notifyJoinRequests.value = data.notify_join_requests ?? true
      notifySharedExpenses.value = data.notify_shared_expenses ?? true
      onboardingCompleted.value = data.onboarding_completed ?? false
      hasProperty.value = data.has_property ?? false
      isPropertyAdmin.value = data.is_property_admin ?? false
      pendingJoinRequest.value = data.pending_join_request ?? null

      // Apply theme + color theme from server settings
      setTheme(theme.value)
      setColorTheme(colorTheme.value)

      if (data.default_split_with_member_ids) {
        try {
          defaultSplitWithMemberIds.value = JSON.parse(data.default_split_with_member_ids)
        } catch {
          defaultSplitWithMemberIds.value = []
        }
      }

      if (data.default_templates) {
        try {
          defaultTemplates.value = JSON.parse(data.default_templates)
        } catch {
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

  async function loadHouseholdSettings() {
    try {
      // Find the current property
      const { data: properties } = await apiClient.get('/properties')
      if (!properties || properties.length === 0) return

      const currentProp = properties.find(p => p.is_current) || properties[0]
      householdPropertyId.value = currentProp.id

      const { data } = await apiClient.get(`/properties/${currentProp.id}/settings`)
      splitMode.value = data.split_mode || false
    } catch (err) {
      console.error('Error loading household settings:', err)
      splitMode.value = false
      householdPropertyId.value = null
    }
  }

  async function updateHouseholdSettings(updates) {
    if (!householdPropertyId.value) return

    try {
      if (updates.split_mode !== undefined) splitMode.value = updates.split_mode
      await apiClient.put(`/properties/${householdPropertyId.value}/settings`, updates)
    } catch (err) {
      console.error('Error updating household settings:', err)
      throw err
    }
  }

  async function updateSettings(updates) {
    const { setTheme } = useDarkMode()
    const { setColorTheme } = useTheme()
    try {
      const payload = { ...updates }

      // Apply updates locally
      if (payload.theme !== undefined) {
        theme.value = payload.theme
        setTheme(payload.theme)
      }
      if (payload.color_theme !== undefined) {
        colorTheme.value = payload.color_theme
        setColorTheme(payload.color_theme)
      }
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
    colorTheme.value = 'paper'
    currency.value = 'EUR'
    language.value = 'it'
    dateFormat.value = 'DD/MM/YYYY'
    defaultSplitWithMemberIds.value = []
    defaultTemplates.value = {}
    emailNotifications.value = true
    billReminders.value = true
    notificationRetentionDays.value = 90
    notifyJoinRequests.value = true
    notifySharedExpenses.value = true
    loaded.value = false
    onboardingCompleted.value = false
    hasProperty.value = false
    isPropertyAdmin.value = false
    pendingJoinRequest.value = null
    splitMode.value = false
    householdPropertyId.value = null
  }

  return {
    // State
    theme,
    colorTheme,
    currency,
    language,
    dateFormat,
    defaultSplitWithMemberIds,
    defaultTemplates,
    emailNotifications,
    billReminders,
    notificationRetentionDays,
    notifyJoinRequests,
    notifySharedExpenses,
    loaded,
    onboardingCompleted,
    hasProperty,
    isPropertyAdmin,
    pendingJoinRequest,
    splitMode,
    householdPropertyId,
    // Getters
    dateSettings,
    formatSettings,
    // Actions
    loadSettings,
    loadHouseholdSettings,
    updateHouseholdSettings,
    updateSettings,
    $reset
  }
})
