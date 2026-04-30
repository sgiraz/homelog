<template>
  <Teleport to="body">
    <div class="fixed top-4 right-4 z-50 space-y-3 max-w-sm w-full pointer-events-none px-4 sm:px-0">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="[
            'p-4 rounded-xl shadow-lg border backdrop-blur-sm pointer-events-auto',
            getToastClass(toast.type)
          ]"
        >
          <div class="flex items-start gap-3">
            <span class="text-xl flex-shrink-0 mt-0.5">{{ getIcon(toast.type) }}</span>

            <div class="flex-1 min-w-0">
              <div v-if="toast.title" class="font-semibold text-sm mb-0.5">
                {{ toast.title }}
              </div>
              <div class="text-sm">{{ toast.message }}</div>
            </div>

            <button
              @click="removeToast(toast.id)"
              class="flex-shrink-0 opacity-60 hover:opacity-100 transition-opacity"
              :aria-label="t('common.toast.dismiss')"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Progress bar -->
          <div
            v-if="toast.duration > 0"
            class="mt-2 h-0.5 bg-black/10 dark:bg-white/10 rounded-full overflow-hidden"
          >
            <div
              class="h-full bg-black/20 dark:bg-white/30 rounded-full"
              :style="{ animationDuration: toast.duration + 'ms' }"
              style="animation-name: toast-progress; animation-timing-function: linear; animation-fill-mode: forwards;"
            ></div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const toasts = ref([])
let nextId = 0

function getToastClass(type) {
  const classes = {
    success: 'bg-green-50 dark:bg-green-900/90 border-green-200 dark:border-green-700 text-green-900 dark:text-green-100',
    error:   'bg-red-50 dark:bg-red-900/90 border-red-200 dark:border-red-700 text-red-900 dark:text-red-100',
    warning: 'bg-yellow-50 dark:bg-yellow-900/90 border-yellow-200 dark:border-yellow-700 text-yellow-900 dark:text-yellow-100',
    info:    'bg-blue-50 dark:bg-blue-900/90 border-blue-200 dark:border-blue-700 text-blue-900 dark:text-blue-100',
  }
  return classes[type] || classes.info
}

function getIcon(type) {
  return { success: '✅', error: '❌', warning: '⚠️', info: 'ℹ️' }[type] || 'ℹ️'
}

function addToast({ type = 'info', title, message, duration = 5000 }) {
  const id = nextId++
  toasts.value.push({ id, type, title, message, duration })
  if (duration > 0) setTimeout(() => removeToast(id), duration)
  return id
}

function removeToast(id) {
  const index = toasts.value.findIndex(t => t.id === id)
  if (index !== -1) toasts.value.splice(index, 1)
}

// Register global $toast helper
window.$toast = {
  success: (message, title) => addToast({ type: 'success', message, title }),
  error:   (message, title) => addToast({ type: 'error',   message, title }),
  warning: (message, title) => addToast({ type: 'warning', message, title }),
  info:    (message, title) => addToast({ type: 'info',    message, title }),
}

defineExpose({ addToast, removeToast })
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.95);
}

@keyframes toast-progress {
  from { width: 100%; }
  to   { width: 0%; }
}
</style>
