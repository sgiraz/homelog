<template>
  <!-- Top nav -->
  <nav class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 lg:px-6 py-4 sticky top-0 z-30">
    <div class="flex items-center justify-between max-w-7xl mx-auto">
      <!-- Left: Logo + Desktop nav -->
      <div class="flex items-center gap-8">
        <router-link to="/" class="text-xl font-bold text-gray-900 dark:text-white">
          HomeLog
        </router-link>

        <!-- Desktop Navigation -->
        <div class="hidden md:flex items-center gap-1">
          <router-link
            v-for="link in navLinks"
            :key="link.path"
            :to="link.path"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="isActive(link.path)
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
            :aria-current="isActive(link.path) ? 'page' : undefined"
          >
            {{ link.label }}
          </router-link>
        </div>
      </div>

      <!-- Right: Actions -->
      <div class="flex items-center gap-1.5">
        <!-- Search -->
        <router-link
          to="/search"
          class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          aria-label="Cerca"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M21 21l-4.35-4.35M10.5 18a7.5 7.5 0 100-15 7.5 7.5 0 000 15z" />
          </svg>
        </router-link>

        <!-- Dark mode toggle -->
        <button
          @click="handleToggleDarkMode"
          class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          :aria-label="isDark ? 'Passa a modalit\u00E0 chiara' : 'Passa a modalit\u00E0 scura'"
        >
          <svg v-if="isDark" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364-6.364l-.707.707M6.343 17.657l-.707.707M17.657 17.657l-.707-.707M6.343 6.343l-.707-.707M12 8a4 4 0 100 8 4 4 0 000-8z" />
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
        </button>

        <!-- Notification bell -->
        <div class="relative">
          <button
            @click="toggleNotifications"
            class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors relative"
            aria-label="Notifiche"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
            <!-- Badge -->
            <span
              v-if="unreadCount > 0"
              class="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] flex items-center justify-center
                     bg-red-500 text-white text-[10px] font-bold rounded-full px-1 leading-none"
            >
              {{ unreadCount > 9 ? '9+' : unreadCount }}
            </span>
          </button>

          <!-- Notifications Dropdown -->
          <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 scale-95 -translate-y-1"
            enter-to-class="opacity-100 scale-100 translate-y-0"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100 scale-100 translate-y-0"
            leave-to-class="opacity-0 scale-95 -translate-y-1"
          >
            <div
              v-if="showNotifications"
              class="absolute right-0 mt-2 w-80 sm:w-96 bg-white dark:bg-gray-800 rounded-xl shadow-xl
                     border border-gray-200 dark:border-gray-700 overflow-hidden z-50"
            >
              <div class="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-700">
                <span class="text-sm font-semibold text-gray-900 dark:text-white">Notifiche</span>
                <button
                  v-if="notifications.length > 0"
                  @click="markAllRead"
                  class="text-xs text-blue-600 dark:text-blue-400 hover:underline"
                >
                  Segna tutte come lette
                </button>
              </div>

              <div class="max-h-80 overflow-y-auto">
                <div v-if="loadingNotifications" class="py-8 text-center text-sm text-gray-400">
                  Caricamento...
                </div>
                <div v-else-if="notifications.length === 0" class="py-8 text-center">
                  <svg class="w-8 h-8 mx-auto text-gray-300 dark:text-gray-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                  <p class="text-sm text-gray-400 dark:text-gray-500">Nessuna notifica</p>
                </div>
                <template v-else>
                  <button
                    v-for="notif in notifications"
                    :key="`${notif._source}-${notif.id}`"
                    @click="openNotification(notif)"
                    class="w-full text-left px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors border-b border-gray-50 dark:border-gray-700/50 last:border-0"
                    :class="{ 'bg-blue-50/50 dark:bg-blue-900/10': !notif.is_read }"
                  >
                    <div class="flex items-start gap-3">
                      <div :class="['p-1.5 rounded-lg flex-shrink-0 mt-0.5', getNotifBgClass(notif)]">
                        <span class="text-sm">{{ getNotifIcon(notif) }}</span>
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                          <span class="text-sm font-medium text-gray-900 dark:text-white truncate">
                            {{ getNotifLabel(notif) }}
                          </span>
                          <span
                            v-if="!notif.is_read"
                            class="w-2 h-2 bg-blue-500 rounded-full flex-shrink-0"
                          />
                        </div>
                        <p class="text-xs text-gray-600 dark:text-gray-300 mt-0.5 line-clamp-2">{{ notif.content || notif.title }}</p>
                        <span class="text-[10px] text-gray-400 mt-1 block">{{ formatTimeAgo(notif.created_at) }}</span>
                      </div>
                    </div>
                  </button>
                </template>
              </div>

              <!-- View all link -->
              <router-link
                to="/notifications"
                @click="closeDropdowns"
                class="block text-center px-4 py-2.5 text-sm font-medium text-blue-600 dark:text-blue-400
                       hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors
                       border-t border-gray-100 dark:border-gray-700"
              >
                Vedi tutte
              </router-link>
            </div>
          </Transition>
        </div>

        <!-- Avatar + User Menu (desktop) -->
        <div class="relative">
          <!-- Mobile: direct link to settings -->
          <router-link
            to="/settings"
            class="md:hidden flex items-center rounded-full transition-all"
            :class="$route.path === '/settings' ? 'ring-2 ring-blue-500 ring-offset-2 dark:ring-offset-gray-800' : ''"
            aria-label="Impostazioni"
          >
            <img
              v-if="authStore.avatarUrl"
              :src="authStore.avatarUrl"
              :alt="authStore.user?.name"
              class="w-9 h-9 rounded-full object-cover"
            />
            <div
              v-else
              class="w-9 h-9 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium"
            >
              {{ userInitials }}
            </div>
          </router-link>

          <!-- Desktop: dropdown trigger -->
          <button
            @click="toggleUserMenu"
            class="hidden md:flex items-center gap-2 rounded-xl pl-1 pr-2 py-1 transition-all hover:bg-gray-100 dark:hover:bg-gray-700"
            :class="showUserMenu ? 'bg-gray-100 dark:bg-gray-700' : ''"
          >
            <img
              v-if="authStore.avatarUrl"
              :src="authStore.avatarUrl"
              :alt="authStore.user?.name"
              class="w-9 h-9 rounded-full object-cover"
            />
            <div
              v-else
              class="w-9 h-9 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium"
            >
              {{ userInitials }}
            </div>
            <span class="flex items-center gap-1 text-sm text-gray-700 dark:text-gray-300">
              {{ authStore.user?.name?.split(' ')[0] }}
              <svg class="w-3.5 h-3.5 text-gray-400 transition-transform" :class="{ 'rotate-180': showUserMenu }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </span>
          </button>

          <!-- User Dropdown Menu -->
          <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 scale-95 -translate-y-1"
            enter-to-class="opacity-100 scale-100 translate-y-0"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100 scale-100 translate-y-0"
            leave-to-class="opacity-0 scale-95 -translate-y-1"
          >
            <div
              v-if="showUserMenu"
              class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-xl shadow-xl
                     border border-gray-200 dark:border-gray-700 py-1 z-50"
            >
              <router-link
                to="/settings"
                @click="showUserMenu = false"
                class="flex items-center gap-3 px-4 py-2.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                Impostazioni
              </router-link>
              <div class="border-t border-gray-100 dark:border-gray-700 my-1" />
              <button
                @click="handleLogout"
                class="flex items-center gap-3 px-4 py-2.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors w-full text-left"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                </svg>
                Logout
              </button>
            </div>
          </Transition>
        </div>
      </div>
    </div>
  </nav>

  <!-- Click-outside overlay for dropdowns -->
  <div
    v-if="showNotifications || showUserMenu"
    class="fixed inset-0 z-20"
    @click="closeDropdowns"
  />

  <!-- Mobile Bottom Navigation — iOS Floating Pill Tab Bar (4 items) -->
  <div
    class="md:hidden fixed bottom-0 left-0 right-0 z-40"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <nav class="tab-bar-pill mx-3 mb-4" role="tablist">
      <div class="flex items-stretch">
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          role="tab"
          class="tab-item flex-1 flex flex-col items-center justify-center py-3.5 gap-1 relative transition-colors"
          :class="isActive(link.path)
            ? 'text-blue-500 dark:text-blue-400'
            : 'text-gray-500 dark:text-gray-400 active:text-gray-700 dark:active:text-gray-200'"
          :aria-selected="isActive(link.path)"
          :aria-label="link.label"
        >
          <!-- Active pill highlight -->
          <span
            v-if="isActive(link.path)"
            class="active-pill"
          />

          <!-- Icon -->
          <svg class="w-[26px] h-[26px] relative z-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" :stroke-width="isActive(link.path) ? '2.2' : '1.6'" stroke-linecap="round" stroke-linejoin="round">
            <!-- Home -->
            <template v-if="link.id === 'home'">
              <path d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" :fill="isActive(link.path) ? 'currentColor' : 'none'" :fill-opacity="isActive(link.path) ? '0.2' : '0'" />
            </template>
            <!-- Expenses -->
            <template v-else-if="link.id === 'expenses'">
              <path d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" :fill="isActive(link.path) ? 'currentColor' : 'none'" :fill-opacity="isActive(link.path) ? '0.2' : '0'" />
            </template>
            <!-- Utilities -->
            <template v-else-if="link.id === 'utilities'">
              <path d="M13 10V3L4 14h7v7l9-11h-7z" :fill="isActive(link.path) ? 'currentColor' : 'none'" :fill-opacity="isActive(link.path) ? '0.2' : '0'" />
            </template>
            <!-- Projects -->
            <template v-else-if="link.id === 'projects'">
              <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" :fill="isActive(link.path) ? 'currentColor' : 'none'" :fill-opacity="isActive(link.path) ? '0.2' : '0'" />
            </template>
          </svg>

          <!-- Label -->
          <span class="text-[10px] font-semibold leading-none relative z-10">{{ link.shortLabel }}</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useDarkMode } from '@/composables/useDarkMode'
import { communicationsAPI, notificationsAPI, utilitiesAPI } from '@/api/client'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { isDark, themeMode, toggleDarkMode } = useDarkMode()

// When Navbar toggle is clicked, also persist to settings backend
function handleToggleDarkMode() {
  toggleDarkMode()
  // Sync to settings store + backend
  settingsStore.updateSettings({ theme: themeMode.value })
}

const navLinks = [
  { path: '/',          label: 'Dashboard',  shortLabel: 'Home',     id: 'home' },
  { path: '/expenses',  label: 'Spese',      shortLabel: 'Spese',    id: 'expenses' },
  { path: '/utilities', label: 'Servizi',     shortLabel: 'Servizi',  id: 'utilities' },
  { path: '/projects',  label: 'Progetti',   shortLabel: 'Progetti', id: 'projects' },
]

// Notifications
const showNotifications = ref(false)
const notifications = ref([])
const unreadCount = ref(0)
const loadingNotifications = ref(false)

// User menu
const showUserMenu = ref(false)

let pollInterval = null

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function getUtilityIcon(type) {
  const icons = {
    electricity: '\u26A1', gas: '\uD83D\uDD25', water: '\uD83D\uDCA7', waste: '\u267B\uFE0F',
    internet: '\uD83C\uDF10', insurance: '\uD83D\uDEE1\uFE0F', affitto: '\uD83C\uDFE0', mutuo: '\uD83C\uDFE6'
  }
  return icons[type] || '\uD83D\uDCEC'
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
    if (notif.type === 'join_request') return '\uD83D\uDC64'
    if (notif.type === 'expense_shared') return '\uD83D\uDCB3'
    return '\uD83D\uDD14'
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

function getNotifLabel(notif) {
  if (notif._source === 'notification') return notif.title || 'Notifica'
  return notif.utility?.provider || 'Servizio'
}

function formatTimeAgo(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now - d) / 1000)
  if (diff < 60) return 'Ora'
  if (diff < 3600) return `${Math.floor(diff / 60)} min fa`
  if (diff < 86400) return `${Math.floor(diff / 3600)} ore fa`
  if (diff < 604800) return `${Math.floor(diff / 86400)} giorni fa`
  return d.toLocaleDateString(settingsStore.language === 'en' ? 'en-US' : 'it-IT', { day: 'numeric', month: 'short' })
}

async function fetchUnreadCount() {
  try {
    const [commRes, notifRes] = await Promise.all([
      communicationsAPI.getUnreadCount(),
      notificationsAPI.getUnreadCount(),
    ])
    unreadCount.value = (commRes.data.count || 0) + (notifRes.data.count || 0)
  } catch {
    // Silent fail
  }
}

async function fetchNotifications() {
  loadingNotifications.value = true
  try {
    const [commRes, notifRes] = await Promise.all([
      communicationsAPI.getAll({ limit: 10 }),
      notificationsAPI.getAll({ limit: 10 }),
    ])
    const comms = (commRes.data || []).map(c => ({ ...c, _source: 'communication' }))
    const notifs = (notifRes.data || []).map(n => ({ ...n, _source: 'notification' }))
    notifications.value = [...comms, ...notifs]
      .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
      .slice(0, 20)
  } catch {
    notifications.value = []
  } finally {
    loadingNotifications.value = false
  }
}

function toggleNotifications() {
  showUserMenu.value = false
  showNotifications.value = !showNotifications.value
  if (showNotifications.value) {
    fetchNotifications()
  }
}

function toggleUserMenu() {
  showNotifications.value = false
  showUserMenu.value = !showUserMenu.value
}

function closeDropdowns() {
  showNotifications.value = false
  showUserMenu.value = false
}

function openNotification(notif) {
  // Mark as read
  if (!notif.is_read) {
    if (notif._source === 'notification') {
      notificationsAPI.markRead(notif.id)
    } else if (notif.utility) {
      utilitiesAPI.markCommunicationRead(notif.utility.id, notif.id)
    }
    notif.is_read = true
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  }
  // Navigate based on type
  if (notif._source === 'notification') {
    if (notif.type === 'join_request') {
      router.push('/settings?tab=family')
    } else if (notif.type === 'expense_shared') {
      router.push('/expenses')
    }
  } else if (notif.utility_id) {
    router.push(`/utilities/${notif.utility_id}`)
  }
  closeDropdowns()
}

async function markAllRead() {
  const promises = []
  for (const notif of notifications.value) {
    if (!notif.is_read) {
      if (notif._source === 'notification') {
        promises.push(notificationsAPI.markRead(notif.id))
      } else if (notif.utility) {
        promises.push(utilitiesAPI.markCommunicationRead(notif.utility.id, notif.id))
      }
      notif.is_read = true
    }
  }
  await Promise.allSettled(promises)
  unreadCount.value = 0
}

function handleLogout() {
  showUserMenu.value = false
  authStore.logout()
  router.push('/login')
}

onMounted(() => {
  fetchUnreadCount()
  // Poll every 60 seconds
  pollInterval = setInterval(fetchUnreadCount, 60000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
/* ─── Liquid Glass floating pill ─── */
.tab-bar-pill {
  position: relative;
  border-radius: 9999px;
  overflow: hidden;

  /* Glass material: translucent + heavy blur + saturation boost */
  background: rgba(245, 245, 248, 0.55);
  backdrop-filter: saturate(200%) blur(40px) brightness(1.05);
  -webkit-backdrop-filter: saturate(200%) blur(40px) brightness(1.05);

  /* Outer shadow for depth + inset glow for glass edge refraction */
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.5),
    0 4px 24px rgba(0, 0, 0, 0.08),
    0 1px 3px rgba(0, 0, 0, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.6),
    inset 0 -1px 0 rgba(0, 0, 0, 0.04);
}

/* Specular highlight gradient overlay — simulates light hitting glass surface */
.tab-bar-pill::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(
    180deg,
    rgba(255, 255, 255, 0.35) 0%,
    rgba(255, 255, 255, 0.05) 40%,
    rgba(0, 0, 0, 0.02) 100%
  );
  pointer-events: none;
  z-index: 1;
}

/* ─── Dark mode ─── */
.dark .tab-bar-pill {
  background: rgba(28, 28, 32, 0.55);
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.1),
    0 4px 24px rgba(0, 0, 0, 0.35),
    0 1px 3px rgba(0, 0, 0, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    inset 0 -1px 0 rgba(0, 0, 0, 0.15);
}

.dark .tab-bar-pill::before {
  background: linear-gradient(
    180deg,
    rgba(255, 255, 255, 0.08) 0%,
    rgba(255, 255, 255, 0.02) 40%,
    rgba(0, 0, 0, 0.05) 100%
  );
}

/* ─── Active tab pill highlight ─── */
.active-pill {
  position: absolute;
  inset: 5px 4px;
  border-radius: 9999px;
  background: rgba(59, 130, 246, 0.14);
  box-shadow: inset 0 0 8px rgba(59, 130, 246, 0.08);
}

.dark .active-pill {
  background: rgba(96, 165, 250, 0.18);
  box-shadow: inset 0 0 8px rgba(96, 165, 250, 0.1);
}

/* ─── Tap feedback ─── */
.tab-item {
  -webkit-tap-highlight-color: transparent;
}

.tab-item:active {
  transform: scale(0.92);
  transition: transform 0.1s ease;
}

/* ─── Line clamp ─── */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
