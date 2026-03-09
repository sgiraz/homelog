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
            :class="$route.path === link.path
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
            :aria-current="$route.path === link.path ? 'page' : undefined"
          >
            {{ link.label }}
          </router-link>
        </div>
      </div>

      <!-- Right: Actions -->
      <div class="flex items-center gap-2">
        <!-- User initials (desktop only) -->
        <div class="hidden md:flex items-center gap-2 mr-1">
          <div
            class="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium"
            :title="authStore.user?.name"
          >
            {{ userInitials }}
          </div>
        </div>

        <!-- Dark mode toggle -->
        <button
          @click="toggleDarkMode"
          class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          :aria-label="isDark ? 'Passa a modalità chiara' : 'Passa a modalità scura'"
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

        <!-- Logout (desktop) -->
        <button
          @click="handleLogout"
          class="hidden md:block px-3 py-2 rounded-lg text-sm font-medium text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          Logout
        </button>
      </div>
    </div>
  </nav>

  <!-- Mobile Bottom Navigation (Apple HIG Tab Bar) -->
  <nav
    class="md:hidden fixed bottom-0 left-0 right-0 z-40 tab-bar-glass"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <div class="flex items-stretch">
      <router-link
        v-for="link in navLinks"
        :key="link.path"
        :to="link.path"
        class="flex-1 flex flex-col items-center justify-center pt-2 pb-1.5 gap-1 min-h-[50px] transition-colors relative"
        :class="isActive(link.path)
          ? 'text-blue-600 dark:text-blue-400'
          : 'text-gray-400 dark:text-gray-500 active:text-gray-600 dark:active:text-gray-300'"
        :aria-current="isActive(link.path) ? 'page' : undefined"
        :aria-label="link.label"
      >
        <!-- Icon -->
        <svg class="w-[22px] h-[22px] transition-transform" :class="isActive(link.path) ? 'scale-105' : ''" viewBox="0 0 24 24" fill="none" stroke="currentColor" :stroke-width="isActive(link.path) ? '2.2' : '1.8'">
          <path v-if="link.id === 'home' && isActive(link.path)" stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" fill="currentColor" fill-opacity="0.15" />
          <path v-else-if="link.id === 'home'" stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />

          <path v-else-if="link.id === 'expenses' && isActive(link.path)" stroke-linecap="round" stroke-linejoin="round" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" fill="currentColor" fill-opacity="0.15" />
          <path v-else-if="link.id === 'expenses'" stroke-linecap="round" stroke-linejoin="round" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />

          <path v-else-if="link.id === 'utilities' && isActive(link.path)" stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" fill="currentColor" fill-opacity="0.15" />
          <path v-else-if="link.id === 'utilities'" stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />

          <path v-else-if="link.id === 'projects' && isActive(link.path)" stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" fill="currentColor" fill-opacity="0.15" />
          <path v-else-if="link.id === 'projects'" stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />

          <path v-else-if="link.id === 'settings' && isActive(link.path)" stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066zM15 12a3 3 0 11-6 0 3 3 0 016 0z" fill="currentColor" fill-opacity="0.15" />
          <path v-else-if="link.id === 'settings'" stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066zM15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>

        <!-- Label -->
        <span class="text-[10px] font-semibold leading-none tracking-tight">{{ link.shortLabel }}</span>
      </router-link>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { isDark, toggleDarkMode } = useDarkMode()

const navLinks = [
  { path: '/',          label: 'Dashboard',    shortLabel: 'Home',      id: 'home' },
  { path: '/expenses',  label: 'Spese',        shortLabel: 'Spese',     id: 'expenses' },
  { path: '/utilities', label: 'Utenze',       shortLabel: 'Utenze',    id: 'utilities' },
  { path: '/projects',  label: 'Progetti',     shortLabel: 'Progetti',  id: 'projects' },
  { path: '/settings',  label: 'Impostazioni', shortLabel: 'Impostaz.', id: 'settings' },
]

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.tab-bar-glass {
  background: rgba(249, 250, 251, 0.82);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-top: 0.5px solid rgba(0, 0, 0, 0.12);
}

:root.dark .tab-bar-glass,
.dark .tab-bar-glass {
  background: rgba(17, 24, 39, 0.82);
  border-top-color: rgba(255, 255, 255, 0.08);
}
</style>
