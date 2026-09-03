<template>
  <div class="space-y-4">
    <Card class="p-6">
      <h2 class="text-xl font-bold text-ink mb-4">{{ t('settings.preferences.title') }}</h2>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">{{ t('settings.preferences.theme') }}</label>
          <select
            v-model="preferences.theme"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="light">{{ t('settings.preferences.themeLight') }}</option>
            <option value="dark">{{ t('settings.preferences.themeDark') }}</option>
            <option value="auto">{{ t('settings.preferences.themeAuto') }}</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">{{ t('settings.preferences.colorTheme') }}</label>
          <div class="grid grid-cols-3 sm:grid-cols-5 gap-2">
            <button
              v-for="th in sortedThemes"
              :key="th.id"
              type="button"
              @click="selectColorTheme(th.id)"
              class="flex flex-col items-center gap-1.5 p-2 rounded-xl border transition-colors"
              :class="preferences.color_theme === th.id
                ? 'border-accent ring-2 ring-accent/40'
                : 'border-line hover:border-ink-faint'"
              :aria-pressed="preferences.color_theme === th.id"
            >
              <span class="flex h-8 w-full overflow-hidden rounded-lg border border-line">
                <span class="flex-1" :style="{ backgroundColor: th.swatch[0] }"></span>
                <span class="flex-1" :style="{ backgroundColor: th.swatch[1] }"></span>
                <span class="flex-1" :style="{ backgroundColor: th.swatch[2] }"></span>
              </span>
              <span class="text-xs font-medium text-ink">{{ t('settings.preferences.themes.' + th.id) }}</span>
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">{{ t('settings.preferences.currency') }}</label>
          <select
            v-model="preferences.currency"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="EUR">{{ t('settings.preferences.options.currencyEUR') }}</option>
            <option value="USD">{{ t('settings.preferences.options.currencyUSD') }}</option>
            <option value="GBP">{{ t('settings.preferences.options.currencyGBP') }}</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">{{ t('settings.preferences.language') }}</label>
          <select
            v-model="preferences.language"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option v-for="opt in localeOptions()" :key="opt.code" :value="opt.code">{{ opt.label }}</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">{{ t('settings.preferences.dateFormat') }}</label>
          <select
            v-model="preferences.date_format"
            @change="updateUserSettings"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="DD/MM/YYYY">{{ t('settings.preferences.options.dateFormatDDMMYYYY') }}</option>
            <option value="MM/DD/YYYY">{{ t('settings.preferences.options.dateFormatMMDDYYYY') }}</option>
            <option value="YYYY-MM-DD">{{ t('settings.preferences.options.dateFormatYYYYMMDD') }}</option>
            <option value="DD MMM YYYY">{{ t('settings.preferences.options.dateFormatDDMMMYYYY') }}</option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Expense Templates -->
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-ink">{{ t('settings.preferences.templates.title') }}</h2>
        <button
          type="button"
          @click="showAddTemplate = true"
          class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
        >
          {{ t('settings.preferences.templates.addNew') }}
        </button>
      </div>

      <div v-if="templatesLoading" class="text-sm text-ink-muted py-4 text-center">
        {{ t('common.states.loading') }}
      </div>

      <div v-else-if="templates.length === 0" class="text-sm text-ink-muted py-4 text-center">
        {{ t('settings.preferences.templates.empty') }}
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="flex items-center gap-3 p-3 rounded-lg border border-line"
        >
          <span class="text-xl flex-shrink-0">{{ tpl.icon || tpl.category?.icon || '📋' }}</span>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium text-ink truncate">{{ tpl.name }}</div>
            <div class="text-xs text-ink-muted">
              {{ categoryLabel(tpl.category) }}<span v-if="tpl.subcategory"> / {{ categoryLabel(tpl.subcategory) }}</span>
              <span v-if="tpl.amount"> · {{ formatCurrency(tpl.amount) }}</span>
            </div>
          </div>
          <button
            type="button"
            @click="editTemplate(tpl)"
            class="p-1.5 rounded hover:bg-surface-3 text-ink-faint hover:text-blue-500"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            type="button"
            @click="deleteTemplate(tpl)"
            class="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-ink-faint hover:text-red-500"
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
          :label="t('settings.preferences.templates.nameLabel')"
          :placeholder="t('settings.preferences.templates.namePlaceholder')"
          required
        />
        <div class="grid grid-cols-2 gap-3">
          <Input
            v-model.number="tplForm.amount"
            :label="t('settings.preferences.templates.amountLabel')"
            type="number"
            step="0.01"
            min="0"
            :placeholder="t('settings.preferences.templates.amountPlaceholder')"
            inputmode="decimal"
          />
          <div>
            <label class="block text-sm text-ink-soft mb-1">{{ t('settings.preferences.templates.categoryLabel') }}</label>
            <select
              v-model.number="tplForm.category_id"
              @change="tplForm.subcategory_id = null"
              required
              class="w-full px-3 py-3 border border-line rounded-lg
                     bg-surface text-ink text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="cat in tplCategories" :key="cat.id" :value="cat.id">
                {{ cat.icon }} {{ categoryLabel(cat) }}
              </option>
            </select>
          </div>
        </div>
        <div v-if="tplSubcategories.length > 0">
          <label class="block text-sm text-ink-soft mb-1">{{ t('settings.preferences.templates.subcategoryLabel') }}</label>
          <select
            v-model.number="tplForm.subcategory_id"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null">{{ t('settings.preferences.templates.noSubcategory') }}</option>
            <option v-for="sub in tplSubcategories" :key="sub.id" :value="sub.id">
              {{ categoryLabel(sub) }}
            </option>
          </select>
        </div>
        <div class="flex gap-2">
          <Button size="sm" @click="handleSaveTemplate" :disabled="!tplForm.name || !tplForm.category_id">
            {{ editingTemplateId ? t('settings.preferences.templates.saveUpdate') : t('settings.preferences.templates.saveCreate') }}
          </Button>
          <Button size="sm" variant="secondary" @click="cancelTemplateForm">
            {{ t('settings.preferences.templates.cancel') }}
          </Button>
        </div>
      </div>
    </Card>

    <!-- Notifications -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-ink mb-4">{{ t('settings.preferences.notifications.title') }}</h2>
      <div class="space-y-4">
        <div class="space-y-3 mb-2">
          <p class="text-sm text-ink-soft">{{ t('settings.preferences.notifications.description') }}</p>
          <label class="flex items-center justify-between cursor-pointer">
            <span class="text-sm text-ink-soft">{{ t('settings.preferences.notifications.joinRequests') }}</span>
            <input type="checkbox" v-model="notifyJoinRequests" @change="updateNotificationPrefs"
                   class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500" />
          </label>
          <label class="flex items-center justify-between cursor-pointer">
            <span class="text-sm text-ink-soft">{{ t('settings.preferences.notifications.sharedExpenses') }}</span>
            <input type="checkbox" v-model="notifySharedExpenses" @change="updateNotificationPrefs"
                   class="w-5 h-5 text-blue-600 rounded border-line focus:ring-blue-500" />
          </label>
        </div>
        <div>
          <label class="block text-sm font-medium text-ink-soft mb-2">
            {{ t('settings.preferences.notifications.retentionLabel') }}
          </label>
          <select
            v-model.number="notificationRetentionDays"
            @change="updateRetentionDays"
            class="w-full px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="30">{{ t('settings.preferences.notifications.retentionDays', { n: 30 }) }}</option>
            <option :value="60">{{ t('settings.preferences.notifications.retentionDays', { n: 60 }) }}</option>
            <option :value="90">{{ t('settings.preferences.notifications.retentionDays', { n: 90 }) }}</option>
            <option :value="180">{{ t('settings.preferences.notifications.retentionDays', { n: 180 }) }}</option>
            <option :value="365">{{ t('settings.preferences.notifications.retentionYear') }}</option>
          </select>
          <p class="text-xs text-ink-muted mt-1">
            {{ t('settings.preferences.notifications.retentionInfo') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- Account -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-ink mb-4">{{ t('settings.preferences.account.title') }}</h2>
      <div class="space-y-3">
        <!-- Change Password toggle -->
        <div>
          <button
            type="button"
            @click="showChangePassword = !showChangePassword; pwError = null; pwSuccess = null"
            class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
          >
            {{ showChangePassword ? t('settings.preferences.account.cancelChange') : t('settings.preferences.account.changePassword') }}
          </button>

          <div v-if="showChangePassword" class="mt-4 space-y-3">
            <Input
              v-model="pwForm.current"
              :label="t('settings.preferences.account.currentPasswordLabel')"
              type="password"
              :placeholder="t('settings.preferences.account.currentPasswordPlaceholder')"
              autocomplete="current-password"
            />
            <Input
              v-model="pwForm.newPw"
              :label="t('settings.preferences.account.newPasswordLabel')"
              type="password"
              :placeholder="t('settings.preferences.account.newPasswordPlaceholder')"
              autocomplete="new-password"
            />
            <Input
              v-model="pwForm.confirm"
              :label="t('settings.preferences.account.confirmPasswordLabel')"
              type="password"
              :placeholder="t('settings.preferences.account.confirmPasswordPlaceholder')"
              autocomplete="new-password"
            />

            <div v-if="pwError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
              {{ pwError }}
            </div>
            <div v-if="pwSuccess" class="text-green-700 text-sm bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
              {{ pwSuccess }}
            </div>

            <Button :disabled="pwLoading" @click="handleChangePassword">
              {{ pwLoading ? t('settings.preferences.account.submittingButton') : t('settings.preferences.account.submitButton') }}
            </Button>
          </div>
        </div>

        <Button variant="danger" @click="handleLogout">
          {{ t('settings.preferences.account.logout') }}
        </Button>
      </div>
    </Card>

    <DeleteAccountSection />
  </div>
</template>

<script setup>
defineOptions({ name: 'PreferencesTab' })

import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useDarkMode } from '@/composables/useDarkMode'
import { useTheme } from '@/composables/useTheme'
import { THEMES } from '@/config/themes'
import { categoryLabel } from '@/utils/categoryLabel'
import { authAPI, expenseTemplatesAPI, categoriesAPI } from '@/api/client'
import apiClient from '@/api/client'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'
import DeleteAccountSection from './DeleteAccountSection.vue'
import { DEFAULT_LOCALE, localeOptions } from '@/i18n'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { setTheme } = useDarkMode()
const { setColorTheme } = useTheme()

const preferences = ref({
  theme: 'auto',
  color_theme: 'slate',
  currency: 'EUR',
  language: DEFAULT_LOCALE,
  date_format: 'DD/MM/YYYY'
})

// Themes shown alphabetically by their localized label (per language).
const sortedThemes = computed(() =>
  [...THEMES].sort((a, b) =>
    t(`settings.preferences.themes.${a.id}`).localeCompare(t(`settings.preferences.themes.${b.id}`))
  )
)

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
      const idx = templates.value.findIndex(tpl => tpl.id === editingTemplateId.value)
      if (idx !== -1) templates.value[idx] = data
      window.$toast?.success(t('settings.preferences.templates.updated'))
    } else {
      const { data } = await expenseTemplatesAPI.create(payload)
      templates.value.push(data)
      window.$toast?.success(t('settings.preferences.templates.created'))
    }
    cancelTemplateForm()
  } catch {
    window.$toast?.error(editingTemplateId.value ? t('settings.preferences.templates.updateError') : t('settings.preferences.templates.createError'))
  }
}

async function deleteTemplate(tpl) {
  try {
    await expenseTemplatesAPI.delete(tpl.id)
    templates.value = templates.value.filter(item => item.id !== tpl.id)
    window.$toast?.success(t('settings.preferences.templates.deleted'))
  } catch {
    window.$toast?.error(t('settings.preferences.templates.deleteError'))
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
      color_theme: data.color_theme || 'slate',
      currency: data.currency || 'EUR',
      // Locale comes from the store, not the raw payload: on the shared demo
      // account the stored language belongs to whoever visited last.
      language: settingsStore.language,
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

async function selectColorTheme(id) {
  preferences.value.color_theme = id
  setColorTheme(id) // apply instantly; persist in the background
  try {
    await apiClient.put('/settings', { color_theme: id })
    settingsStore.colorTheme = id
  } catch (err) {
    console.error('Error updating color theme:', err)
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
    // Goes through the store so the locale is remembered per browser and is
    // never written to the shared demo account.
    await settingsStore.updateSettings({
      theme: preferences.value.theme,
      currency: preferences.value.currency,
      language: preferences.value.language,
      date_format: preferences.value.date_format,
      default_split_with_member_ids: currentSettings.default_split_with_member_ids || '[]'
    })
  } catch (err) {
    console.error('Error updating user settings:', err)
  }
}

async function handleChangePassword() {
  pwError.value = null
  pwSuccess.value = null

  if (!pwForm.value.current) {
    pwError.value = t('settings.preferences.account.currentRequired')
    return
  }
  if (pwForm.value.newPw.length < 6) {
    pwError.value = t('settings.preferences.account.newTooShort')
    return
  }
  if (pwForm.value.newPw !== pwForm.value.confirm) {
    pwError.value = t('settings.preferences.account.mismatch')
    return
  }

  pwLoading.value = true
  try {
    await authAPI.changePassword(pwForm.value.current, pwForm.value.newPw)
    pwSuccess.value = t('settings.preferences.account.success')
    pwForm.value = { current: '', newPw: '', confirm: '' }
    setTimeout(() => { showChangePassword.value = false; pwSuccess.value = null }, 2000)
  } catch (err) {
    pwError.value = err.response?.data?.error || t('settings.preferences.account.genericError')
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
  loadTemplates()
  loadCategories()
})
</script>
