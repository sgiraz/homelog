<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Skip to main content (accessibility) -->
    <a
      href="#main-content"
      class="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4
             bg-blue-600 text-white px-4 py-2 rounded-lg z-50 text-sm font-medium"
    >
      {{ $t('nav.skipToContent') }}
    </a>

    <Navbar v-if="authStore.isAuthenticated" />

    <main
      id="main-content"
      :class="authStore.isAuthenticated ? 'max-w-7xl mx-auto p-6 pb-24 md:pb-6' : ''"
      tabindex="-1"
    >
      <router-view v-slot="{ Component, route }">
        <keep-alive :include="cachedViews">
          <component :is="Component" :key="route.path" />
        </keep-alive>
      </router-view>
    </main>

    <Toast />
    <ConfirmDialog />
    <PwaUpdatePrompt />
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { setI18nLocale } from '@/i18n'
import Navbar from '@/components/layout/Navbar.vue'
import Toast from '@/components/common/Toast.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import PwaUpdatePrompt from '@/components/common/PwaUpdatePrompt.vue'

const authStore = useAuthStore()
const settingsStore = useSettingsStore()

// Cache main views to avoid re-mount flicker on navigation
const cachedViews = ['DashboardView', 'ExpensesView', 'UtilitiesView', 'ProjectsView', 'SettingsView']

if (authStore.isAuthenticated) {
  settingsStore.loadSettings()
  settingsStore.loadHouseholdSettings()
}

watch(() => authStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    settingsStore.loadSettings()
    settingsStore.loadHouseholdSettings()
  } else {
    settingsStore.$reset()
  }
})

watch(
  () => settingsStore.language,
  (lang) => setI18nLocale(lang),
  { immediate: true }
)
</script>

