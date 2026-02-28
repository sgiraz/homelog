<template>
  <Teleport to="body">
    <Transition name="confirm-fade">
      <div
        v-if="state.show"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-[200] p-4"
        @click.self="handleCancel"
      >
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl max-w-sm w-full p-6">
          <h3 v-if="state.title" class="text-lg font-bold text-gray-900 dark:text-white mb-2">
            {{ state.title }}
          </h3>
          <p class="text-gray-600 dark:text-gray-400 mb-6">{{ state.message }}</p>
          <div class="flex gap-3 justify-end">
            <button
              @click="handleCancel"
              class="px-4 py-2 rounded-lg font-medium text-gray-700 dark:text-gray-300
                     bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600
                     transition-colors"
            >
              {{ state.cancelText }}
            </button>
            <button
              @click="handleConfirm"
              :class="[
                'px-4 py-2 rounded-lg font-medium text-white transition-colors',
                state.variant === 'danger'
                  ? 'bg-red-600 hover:bg-red-700'
                  : 'bg-blue-600 hover:bg-blue-700'
              ]"
            >
              {{ state.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { useConfirm } from '@/composables/useConfirm'

const { state, handleConfirm, handleCancel } = useConfirm()
</script>

<style scoped>
.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.15s ease;
}
.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
</style>
