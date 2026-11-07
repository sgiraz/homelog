<template>
  <Transition
    enter-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0"
    leave-active-class="transition-opacity duration-150"
    leave-to-class="opacity-0"
  >
    <div
      v-if="isWaking"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    >
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="demo-wakeup-title"
        aria-describedby="demo-wakeup-message"
        class="w-full max-w-sm rounded-xl border border-line bg-surface p-6 shadow-lg text-center"
      >
        <svg class="mx-auto h-8 w-8 animate-spin text-accent" fill="none" viewBox="0 0 24 24" aria-hidden="true">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
        <h2 id="demo-wakeup-title" class="mt-4 text-base font-semibold text-ink">
          {{ t('demo.wakeup.title') }}
        </h2>
        <p id="demo-wakeup-message" class="mt-2 text-sm text-ink-soft">
          {{ t('demo.wakeup.message') }}
        </p>
        <button
          type="button"
          class="mt-5 px-4 py-2 rounded-lg text-sm font-medium text-ink-soft hover:bg-surface-2"
          @click="retry"
        >
          {{ t('demo.wakeup.retry') }}
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useDemoWakeup } from '@/composables/useDemoWakeup'

const { t } = useI18n()
const { isWaking } = useDemoWakeup()

function retry() {
  window.location.reload()
}
</script>
