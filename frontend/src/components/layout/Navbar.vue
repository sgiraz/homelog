<template>
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
          :title="isDark ? 'Passa a modalità chiara' : 'Passa a modalità scura'"
          :aria-label="isDark ? 'Passa a modalità chiara' : 'Passa a modalità scura'"
        >
          <!-- Sun icon (shown in dark mode) -->
          <svg v-if="isDark" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364-6.364l-.707.707M6.343 17.657l-.707.707M17.657 17.657l-.707-.707M6.343 6.343l-.707-.707M12 8a4 4 0 100 8 4 4 0 000-8z" />
          </svg>
          <!-- Moon icon (shown in light mode) -->
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

        <!-- Mobile hamburger -->
        <button
          @click="mobileMenuOpen = !mobileMenuOpen"
          class="md:hidden p-2 rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          :aria-label="mobileMenuOpen ? 'Chiudi menu' : 'Apri menu'"
          :aria-expanded="mobileMenuOpen"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path v-if="!mobileMenuOpen" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Mobile Navigation -->
    <Transition name="mobile-menu">
      <div
        v-if="mobileMenuOpen"
        class="md:hidden mt-3 pb-2 border-t border-gray-100 dark:border-gray-700 pt-3 space-y-1"
      >
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          @click="mobileMenuOpen = false"
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
          :class="$route.path === link.path
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          :aria-current="$route.path === link.path ? 'page' : undefined"
        >
          <span class="text-lg">{{ link.icon }}</span>
          <span>{{ link.label }}</span>
        </router-link>

        <div class="border-t border-gray-100 dark:border-gray-700 pt-2 mt-2 flex items-center justify-between px-3">
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ authStore.user?.name || 'Utente' }}
          </span>
          <button
            @click="handleLogout"
            class="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
          >
            Logout
          </button>
        </div>
      </div>
    </Transition>
  </nav>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'

const router = useRouter()
const authStore = useAuthStore()
const { isDark, toggleDarkMode } = useDarkMode()

const mobileMenuOpen = ref(false)

const navLinks = [
  { path: '/',          label: 'Dashboard',    icon: '🏠' },
  { path: '/expenses',  label: 'Spese',         icon: '💰' },
  { path: '/balance',   label: 'Bilancio',      icon: '⚖️' },
  { path: '/utilities', label: 'Utenze',        icon: '⚡' },
  { path: '/projects',  label: 'Progetti',      icon: '📁' },
  { path: '/settings',  label: 'Impostazioni',  icon: '⚙️' },
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

<style scoped>
.mobile-menu-enter-active,
.mobile-menu-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.mobile-menu-enter-from,
.mobile-menu-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
