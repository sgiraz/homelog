<template>
  <Teleport to="body">
    <Transition name="modal" appear>
      <div
        v-if="show"
        class="fixed inset-0 z-50"
        aria-modal="true"
        role="dialog"
        :aria-label="title"
      >
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/50" @click="$emit('close')" />

        <!-- Positioning: bottom on mobile, centered on desktop -->
        <div class="relative h-full flex items-end sm:items-center sm:justify-center sm:p-4">
          <div
            class="modal-content relative bg-white dark:bg-gray-800 w-full
                   rounded-t-2xl sm:rounded-2xl sm:shadow-xl
                   max-h-[92dvh] sm:max-h-[90vh] overflow-y-auto
                   sm:pb-0"
            :class="size === '2xl' ? 'sm:max-w-2xl' : 'sm:max-w-md'"
          >
            <!-- Drag handle (mobile only) -->
            <div class="sm:hidden flex justify-center pt-3 pb-1 sticky top-0 bg-white dark:bg-gray-800 z-10">
              <div class="w-10 h-1.5 bg-gray-300 dark:bg-gray-600 rounded-full" />
            </div>

            <!-- Header -->
            <div class="flex items-center justify-between px-6 pt-4 pb-4">
              <h3 class="text-xl font-bold text-gray-900 dark:text-white">{{ title }}</h3>
              <button
                @click="$emit('close')"
                class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300
                       rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                aria-label="Chiudi"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <!-- Content slot -->
            <div class="px-6 pb-6 pb-[calc(1.5rem+env(safe-area-inset-bottom))] sm:pb-6">
              <slot />
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'

const props = defineProps({
  show: { type: Boolean, default: true },
  title: { type: String, default: '' },
  size: { type: String, default: 'md' }
})

defineEmits(['close'])

onMounted(() => {
  document.body.style.overflow = 'hidden'
})

onUnmounted(() => {
  document.body.style.overflow = ''
})
</script>

<style scoped>
/* Backdrop fade */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

/* Mobile: slide up */
.modal-enter-active .modal-content {
  transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);
}
.modal-leave-active .modal-content {
  transition: transform 0.2s ease-in;
}
.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: translateY(100%);
}

/* Desktop: scale + fade */
@media (min-width: 640px) {
  .modal-enter-active .modal-content {
    transition: opacity 0.2s ease, transform 0.2s ease;
  }
  .modal-leave-active .modal-content {
    transition: opacity 0.15s ease, transform 0.15s ease;
  }
  .modal-enter-from .modal-content,
  .modal-leave-to .modal-content {
    opacity: 0;
    transform: scale(0.95);
  }
}
</style>
