<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Skip to main content (accessibility) -->
    <a
      href="#main-content"
      class="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4
             bg-blue-600 text-white px-4 py-2 rounded-lg z-50 text-sm font-medium"
    >
      Vai al contenuto principale
    </a>

    <Navbar v-if="authStore.isAuthenticated" />

    <main id="main-content" :class="authStore.isAuthenticated ? 'max-w-7xl mx-auto p-6' : ''" tabindex="-1">
      <router-view v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </router-view>
    </main>

    <Toast />
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import Navbar from '@/components/layout/Navbar.vue'
import Toast from '@/components/common/Toast.vue'

const authStore = useAuthStore()
const settingsStore = useSettingsStore()

if (authStore.isAuthenticated) {
  settingsStore.loadSettings()
}

watch(() => authStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    settingsStore.loadSettings()
  } else {
    settingsStore.$reset()
  }
})
</script>

<style>
.page-enter-active,
.page-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.page-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
