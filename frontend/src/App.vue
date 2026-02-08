<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <Navbar v-if="authStore.isAuthenticated" />

    <main :class="authStore.isAuthenticated ? 'max-w-7xl mx-auto p-6' : ''">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import Navbar from '@/components/layout/Navbar.vue'

const authStore = useAuthStore()
const settingsStore = useSettingsStore()

// Load user settings when authenticated
if (authStore.isAuthenticated) {
  settingsStore.loadSettings()
}

// Also watch for login/logout
watch(() => authStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    settingsStore.loadSettings()
  } else {
    settingsStore.$reset()
  }
})
</script>
