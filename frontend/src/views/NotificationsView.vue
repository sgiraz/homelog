<template>
  <div class="max-w-3xl mx-auto px-4 py-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <button
          @click="$router.back()"
          class="p-2 -ml-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('notifications.title') }}</h1>
      </div>

      <!-- Bulk actions -->
      <div v-if="notifications.length > 0" class="flex items-center gap-2">
        <button
          v-if="hasUnread"
          @click="markAllRead"
          class="text-sm text-blue-600 dark:text-blue-400 hover:underline font-medium"
        >
          {{ t('notifications.markAllRead') }}
        </button>
        <button
          v-if="hasRead"
          @click="handleDeleteAllRead"
          class="text-sm text-red-600 dark:text-red-400 hover:underline font-medium"
        >
          {{ t('notifications.deleteRead') }}
        </button>
      </div>
    </div>

    <!-- Filter tabs -->
    <div class="flex gap-1 mb-4 overflow-x-auto pb-1">
      <button
        v-for="tab in filterTabs"
        :key="tab.value"
        @click="activeFilter = tab.value"
        class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors"
        :class="activeFilter === tab.value
          ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
          : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
      >
        {{ tab.label }}
        <span
          v-if="tab.count > 0"
          class="ml-1 min-w-[20px] h-5 flex items-center justify-center rounded-full text-xs px-1"
          :class="activeFilter === tab.value
            ? 'bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-blue-200'
            : 'bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300'"
        >
          {{ tab.count }}
        </span>
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 5" :key="i" class="bg-white dark:bg-gray-800 rounded-xl p-4 animate-pulse">
        <div class="flex items-start gap-3">
          <div class="w-9 h-9 bg-gray-200 dark:bg-gray-700 rounded-lg" />
          <div class="flex-1 space-y-2">
            <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
            <div class="h-3 bg-gray-200 dark:bg-gray-700 rounded w-2/3" />
            <div class="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/4" />
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else-if="filteredNotifications.length === 0" class="text-center py-16">
      <svg class="w-16 h-16 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400 text-lg font-medium">
        {{ activeFilter === 'unread' ? t('notifications.emptyUnread') : t('notifications.emptyAll') }}
      </p>
    </div>

    <!-- Notification list -->
    <div v-else class="space-y-2">
      <TransitionGroup name="list">
        <div
          v-for="notif in filteredNotifications"
          :key="`${notif._source}-${notif.id}`"
          class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700
                 hover:shadow-md transition-all cursor-pointer"
          :class="{ 'border-l-4 border-l-blue-500': !notif.is_read }"
          @click="openNotification(notif)"
        >
          <div class="flex items-start gap-3 p-4">
            <!-- Icon -->
            <div :class="['p-2 rounded-lg flex-shrink-0', getNotifBgClass(notif)]">
              <span class="text-base">{{ getNotifIcon(notif) }}</span>
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-0.5">
                <span class="text-sm font-semibold text-gray-900 dark:text-white truncate">
                  {{ notif.title || notif.utility?.provider || t('nav.notifications.fallbackTitle') }}
                </span>
                <span
                  v-if="notif.is_important"
                  class="px-1.5 py-0.5 text-[10px] font-bold rounded bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300"
                >
                  {{ t('notifications.important') }}
                </span>
                <span
                  v-if="!notif.is_read"
                  class="w-2.5 h-2.5 bg-blue-500 rounded-full flex-shrink-0"
                />
              </div>
              <p class="text-sm text-gray-600 dark:text-gray-300 line-clamp-2">
                {{ notif.content }}
              </p>
              <div class="flex items-center gap-3 mt-2">
                <span v-if="notif.utility?.provider" class="text-xs text-gray-400">
                  {{ notif.utility.provider }}
                </span>
                <span v-else-if="notif.property?.name" class="text-xs text-gray-400">
                  {{ notif.property.name }}
                </span>
                <span class="text-xs text-gray-400">
                  {{ formatTimeAgo(notif.created_at) }}
                </span>
                <span
                  v-if="notif.type"
                  class="px-1.5 py-0.5 text-[10px] font-medium rounded bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400"
                >
                  {{ getTypeLabel(notif.type) }}
                </span>
              </div>
            </div>

            <!-- Delete button -->
            <button
              @click.stop="handleDelete(notif)"
              class="p-2 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors flex-shrink-0"
              :title="t('notifications.deleteAria')"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </div>

  <ConfirmDialog />
</template>

<script setup>
defineOptions({ name: 'NotificationsView' })

import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { communicationsAPI, notificationsAPI, utilitiesAPI } from '@/api/client'
import { useConfirm } from '@/composables/useConfirm'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const router = useRouter()
const { t, te } = useI18n()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const notifications = ref([])
const loading = ref(true)
const activeFilter = ref('all')

function getTypeLabel(type) {
  const key = `notifications.types.${type}`
  return te(key) ? t(key) : type
}

const filteredNotifications = computed(() => {
  if (activeFilter.value === 'unread') return notifications.value.filter(n => !n.is_read)
  return notifications.value
})

const hasUnread = computed(() => notifications.value.some(n => !n.is_read))
const hasRead = computed(() => notifications.value.some(n => n.is_read))

const filterTabs = computed(() => [
  { value: 'all', label: t('notifications.filterAll'), count: notifications.value.length },
  { value: 'unread', label: t('notifications.filterUnread'), count: notifications.value.filter(n => !n.is_read).length },
])

function getUtilityIcon(type) {
  const icons = {
    electricity: '⚡', gas: '🔥', water: '💧', waste: '♻️',
    internet: '🌐', insurance: '🛡️', affitto: '🏠', mutuo: '🏦'
  }
  return icons[type] || '📬'
}

function getUtilityBgClass(type) {
  const classes = {
    electricity: 'bg-yellow-100 dark:bg-yellow-900/30',
    gas: 'bg-orange-100 dark:bg-orange-900/30',
    water: 'bg-blue-100 dark:bg-blue-900/30',
    waste: 'bg-green-100 dark:bg-green-900/30',
    internet: 'bg-indigo-100 dark:bg-indigo-900/30',
    insurance: 'bg-emerald-100 dark:bg-emerald-900/30',
    affitto: 'bg-purple-100 dark:bg-purple-900/30',
    mutuo: 'bg-sky-100 dark:bg-sky-900/30',
  }
  return classes[type] || 'bg-gray-100 dark:bg-gray-700'
}

function getNotifIcon(notif) {
  if (notif._source === 'notification') {
    if (notif.type === 'join_request') return '👤'
    if (notif.type === 'expense_shared') return '💳'
    return '🔔'
  }
  return getUtilityIcon(notif.utility?.type)
}

function getNotifBgClass(notif) {
  if (notif._source === 'notification') {
    if (notif.type === 'join_request') return 'bg-violet-100 dark:bg-violet-900/30'
    if (notif.type === 'expense_shared') return 'bg-emerald-100 dark:bg-emerald-900/30'
    return 'bg-gray-100 dark:bg-gray-700'
  }
  return getUtilityBgClass(notif.utility?.type)
}

function formatTimeAgo(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now - d) / 1000)
  if (diff < 60) return t('nav.notifications.timeAgo.now')
  if (diff < 3600) return t('nav.notifications.timeAgo.minutes', { n: Math.floor(diff / 60) })
  if (diff < 86400) return t('nav.notifications.timeAgo.hours', { n: Math.floor(diff / 3600) })
  if (diff < 604800) return t('nav.notifications.timeAgo.days', { n: Math.floor(diff / 86400) })
  return d.toLocaleDateString(settingsStore.language === 'en' ? 'en-US' : 'it-IT', { day: 'numeric', month: 'short', year: 'numeric' })
}

async function fetchNotifications() {
  loading.value = true
  try {
    const [commRes, notifRes] = await Promise.all([
      communicationsAPI.getAll({ limit: 100 }),
      notificationsAPI.getAll({ limit: 100 }),
    ])
    const comms = (commRes.data || []).map(c => ({ ...c, _source: 'communication' }))
    const notifs = (notifRes.data || []).map(n => ({ ...n, _source: 'notification' }))
    notifications.value = [...comms, ...notifs]
      .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
  } catch {
    notifications.value = []
  } finally {
    loading.value = false
  }
}

function openNotification(notif) {
  if (!notif.is_read) {
    if (notif._source === 'notification') {
      notificationsAPI.markRead(notif.id)
    } else if (notif.utility_id) {
      utilitiesAPI.markCommunicationRead(notif.utility_id, notif.id)
    }
    notif.is_read = true
  }
  if (notif._source === 'notification') {
    if (notif.type === 'join_request') router.push('/settings?tab=family')
    else if (notif.type === 'expense_shared') router.push('/expenses')
  } else if (notif.utility_id) {
    router.push(`/utilities/${notif.utility_id}`)
  }
}

async function markAllRead() {
  const promises = []
  // Mark all generic notifications as read in one call
  if (notifications.value.some(n => !n.is_read && n._source === 'notification')) {
    promises.push(notificationsAPI.markAllRead())
  }
  // Mark communications individually (no bulk endpoint)
  for (const notif of notifications.value) {
    if (!notif.is_read && notif._source === 'communication' && notif.utility_id) {
      promises.push(utilitiesAPI.markCommunicationRead(notif.utility_id, notif.id))
    }
  }
  await Promise.allSettled(promises)
  notifications.value.forEach(n => { n.is_read = true })
}

async function handleDelete(notif) {
  const ok = await confirm({
    title: t('notifications.deleteOneTitle'),
    message: t('notifications.deleteOneMessage'),
    confirmText: t('notifications.deleteOneConfirm'),
    variant: 'danger'
  })
  if (!ok) return
  try {
    if (notif._source === 'notification') {
      await notificationsAPI.delete(notif.id)
    } else if (notif.utility_id) {
      await utilitiesAPI.deleteCommunication(notif.utility_id, notif.id)
    }
    notifications.value = notifications.value.filter(n => !(n.id === notif.id && n._source === notif._source))
  } catch (err) {
    console.error('Error deleting notification:', err)
  }
}

async function handleDeleteAllRead() {
  const readCount = notifications.value.filter(n => n.is_read).length
  const ok = await confirm({
    title: t('notifications.deleteAllTitle'),
    message: t('notifications.deleteAllMessage', { n: readCount }),
    confirmText: t('notifications.deleteAllConfirm'),
    variant: 'danger'
  })
  if (!ok) return
  try {
    await Promise.allSettled([
      communicationsAPI.deleteAllRead(),
      notificationsAPI.deleteAllRead(),
    ])
    notifications.value = notifications.value.filter(n => !n.is_read)
  } catch (err) {
    console.error('Error deleting read notifications:', err)
  }
}

onMounted(() => {
  fetchNotifications()
})
</script>

<style scoped>
.list-enter-active,
.list-leave-active {
  transition: all 0.3s ease;
}
.list-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}
.list-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
