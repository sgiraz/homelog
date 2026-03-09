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

  <!-- Mobile Bottom Navigation — iOS Floating Pill Tab Bar -->
  <div
    class="md:hidden fixed bottom-0 left-0 right-0 z-40"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <nav class="tab-bar-pill mx-3 mb-2" role="tablist">
      <div class="flex items-stretch">
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          role="tab"
          class="tab-item flex-1 flex flex-col items-center justify-center py-3 gap-1 relative transition-colors"
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
            <!-- Settings -->
            <template v-else-if="link.id === 'settings'">
              <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066zM15 12a3 3 0 11-6 0 3 3 0 016 0z" :fill="isActive(link.path) ? 'currentColor' : 'none'" :fill-opacity="isActive(link.path) ? '0.15' : '0'" />
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
</style>
