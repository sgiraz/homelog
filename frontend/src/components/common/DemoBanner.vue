<template>
  <div
    v-if="isDemoMode"
    class="bg-indigo-600 text-white text-sm"
    role="status"
  >
    <div class="max-w-7xl mx-auto px-4 py-2 flex items-center gap-3">
      <span class="font-semibold flex items-center gap-1.5 flex-shrink-0">
        🎭 {{ t('demo.banner.label') }}
      </span>
      <span class="hidden sm:inline text-indigo-100 truncate">
        {{ t('demo.banner.description') }}
      </span>
      <button
        type="button"
        class="ml-auto flex-shrink-0 inline-flex items-center gap-1.5 rounded-lg
               bg-white/15 hover:bg-white/25 disabled:opacity-60
               px-3 py-1 font-medium transition-colors"
        :disabled="isResetting"
        @click="onReset"
      >
        <svg class="w-4 h-4" :class="{ 'animate-spin': isResetting }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {{ isResetting ? t('demo.banner.resetting') : t('demo.banner.reset') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useDemoMode } from '@/composables/useDemoMode'
import { useConfirm } from '@/composables/useConfirm'

const { t } = useI18n()
const { isDemoMode, isResetting, resetDemo } = useDemoMode()
const { confirm } = useConfirm()

async function onReset() {
  const ok = await confirm({
    title: t('demo.reset.confirmTitle'),
    message: t('demo.reset.confirmMessage'),
    confirmText: t('demo.reset.confirmButton'),
    variant: 'danger',
  })
  if (ok) resetDemo()
}
</script>
