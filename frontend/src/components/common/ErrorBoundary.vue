<template>
  <div v-if="hasError" class="p-8 text-center">
    <div class="text-6xl mb-4">⚠️</div>
    <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
      Qualcosa è andato storto
    </h2>
    <p class="text-gray-500 dark:text-gray-400 mb-6 max-w-md mx-auto">
      {{ errorMessage || 'Si è verificato un errore imprevisto.' }}
    </p>
    <button
      @click="retry"
      class="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
    >
      Riprova
    </button>
  </div>
  <slot v-else />
</template>

<script setup>
import { ref, onErrorCaptured } from 'vue'

const hasError = ref(false)
const errorMessage = ref(null)

onErrorCaptured((err) => {
  hasError.value = true
  errorMessage.value = err.message
  console.error('[ErrorBoundary]', err)
  return false
})

function retry() {
  hasError.value = false
  errorMessage.value = null
}
</script>
