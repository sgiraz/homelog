<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <Navbar v-if="authStore.isAuthenticated" />

    <main :class="authStore.isAuthenticated ? 'max-w-7xl mx-auto p-6' : ''">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useAuthStore } from '@/stores/auth'
import Navbar from '@/components/layout/Navbar.vue'

const authStore = useAuthStore()
</script>
