<template>
  <Teleport to="body">
    <Transition name="confirm-fade">
      <div
        v-if="state.show"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-[200] p-4"
        @click.self="handleCancel"
      >
        <div class="bg-surface rounded-2xl shadow-xl max-w-sm w-full p-6">
          <h3 v-if="state.title" class="text-lg font-bold text-ink mb-2">
            {{ state.title }}
          </h3>
          <p class="text-ink-soft mb-6">{{ state.message }}</p>
          <div class="flex gap-3 justify-end">
            <button
              @click="handleCancel"
              class="px-4 py-3 rounded-lg font-medium text-ink-soft
                     bg-surface-2 hover:bg-surface-3
                     transition-colors"
            >
              {{ state.cancelText || t('common.actions.cancel') }}
            </button>
            <button
              @click="handleConfirm"
              :class="[
                'px-4 py-3 rounded-lg font-medium text-white transition-colors',
                state.variant === 'danger'
                  ? 'bg-red-600 hover:bg-red-700'
                  : 'bg-blue-600 hover:bg-blue-700'
              ]"
            >
              {{ state.confirmText || t('common.actions.confirm') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useConfirm } from '@/composables/useConfirm'

const { t } = useI18n()
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
