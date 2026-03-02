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

  <!-- Mobile Bottom Navigation -->
  <nav
    class="md:hidden fixed bottom-0 left-0 right-0 z-40
           bg-white/95 dark:bg-gray-900/95 backdrop-blur-xl
           border-t border-gray-200 dark:border-gray-700"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <div class="flex items-stretch">
      <router-link
        v-for="link in navLinks"
        :key="link.path"
        :to="link.path"
        class="flex-1 flex flex-col items-center justify-center py-2 gap-0.5 min-h-[56px] transition-colors"
        :class="$route.path === link.path
          ? 'text-blue-600 dark:text-blue-400'
          : 'text-gray-500 dark:text-gray-400'"
        :aria-current="$route.path === link.path ? 'page' : undefined"
        :aria-label="link.label"
      >
        <component :is="'span'" class="text-xl leading-none" aria-hidden="true">{{ link.icon }}</component>
        <span class="text-[11px] font-medium leading-tight">{{ link.shortLabel }}</span>
      </router-link>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'

const router = useRouter()
const authStore = useAuthStore()
const { isDark, toggleDarkMode } = useDarkMode()

const navLinks = [
  { path: '/',          label: 'Dashboard',    shortLabel: 'Home',       icon: '🏠' },
  { path: '/expenses',  label: 'Spese',         shortLabel: 'Spese',      icon: '💰' },
  { path: '/utilities', label: 'Utenze',        shortLabel: 'Utenze',     icon: '⚡' },
  { path: '/projects',  label: 'Progetti',      shortLabel: 'Progetti',   icon: '📁' },
  { path: '/settings',  label: 'Impostazioni',  shortLabel: 'Impostaz.',  icon: '⚙️' },
]

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>
