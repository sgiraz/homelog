<template>
  <nav class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 lg:px-6 py-4">
    <div class="flex items-center justify-between max-w-7xl mx-auto">
      <!-- Left side: Logo + Navigation -->
      <div class="flex items-center gap-8">
        <router-link to="/" class="text-xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span class="text-2xl">HomeLog</span>
        </router-link>

        <!-- Desktop Navigation -->
        <div class="hidden md:flex items-center gap-1">
          <router-link
            to="/"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            Dashboard
          </router-link>
          <router-link
            to="/expenses"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/expenses'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            Spese
          </router-link>
          <router-link
            to="/balance"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/balance'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            Bilancio
          </router-link>
          <router-link
            to="/utilities"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/utilities'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            Utenze
          </router-link>
          <router-link
            to="/projects"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/projects'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            Progetti
          </router-link>
          <router-link
            to="/settings"
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="$route.path === '/settings'
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </router-link>
        </div>
      </div>

      <!-- Right side: User + Actions -->
      <div class="flex items-center gap-4">
        <!-- User info (desktop) -->
        <div class="hidden md:flex items-center gap-3">
          <div class="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
            {{ userInitials }}
          </div>
          <span class="text-sm text-gray-600 dark:text-gray-400">
            {{ authStore.user?.name || 'User' }}
          </span>
        </div>

        <Button variant="secondary" @click="handleLogout" class="text-sm">
          Logout
        </Button>

        <!-- Mobile menu button -->
        <button
          @click="mobileMenuOpen = !mobileMenuOpen"
          class="md:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          <svg class="w-6 h-6 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path v-if="!mobileMenuOpen" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Mobile Navigation -->
    <div v-if="mobileMenuOpen" class="md:hidden mt-4 pb-4 border-t border-gray-200 dark:border-gray-700 pt-4">
      <div class="flex flex-col gap-2">
        <router-link
          to="/"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium"
          :class="$route.path === '/'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          Dashboard
        </router-link>
        <router-link
          to="/expenses"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium"
          :class="$route.path === '/expenses'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          Spese
        </router-link>
        <router-link
          to="/balance"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium"
          :class="$route.path === '/balance'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          Bilancio
        </router-link>
        <router-link
          to="/utilities"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium"
          :class="$route.path === '/utilities'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          Utenze
        </router-link>
        <router-link
          to="/projects"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium"
          :class="$route.path === '/projects'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          Progetti
        </router-link>
        <router-link
          to="/settings"
          @click="mobileMenuOpen = false"
          class="px-3 py-2 rounded-lg text-sm font-medium flex items-center gap-2"
          :class="$route.path === '/settings'
            ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-600 dark:text-gray-300'"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          Impostazioni
        </router-link>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/common/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

const mobileMenuOpen = ref(false)

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>
