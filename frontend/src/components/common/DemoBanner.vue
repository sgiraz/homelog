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
      <div class="ml-auto flex items-center gap-2 flex-shrink-0">
        <a
          :href="links.github"
          target="_blank"
          rel="noopener"
          class="inline-flex items-center gap-1.5 rounded-lg
                 bg-white/15 hover:bg-white/25 px-3 py-1 font-medium transition-colors"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 .5C5.37.5 0 5.78 0 12.29c0 5.21 3.44 9.63 8.21 11.19.6.11.82-.25.82-.56 0-.28-.01-1.02-.02-2-3.34.71-4.04-1.58-4.04-1.58-.55-1.37-1.34-1.74-1.34-1.74-1.09-.73.08-.72.08-.72 1.21.08 1.84 1.22 1.84 1.22 1.07 1.8 2.81 1.28 3.5.98.11-.76.42-1.28.76-1.57-2.67-.3-5.47-1.31-5.47-5.84 0-1.29.47-2.34 1.24-3.17-.12-.3-.54-1.52.12-3.17 0 0 1.01-.32 3.3 1.21a11.6 11.6 0 0 1 3-.39c1.02 0 2.05.13 3 .39 2.29-1.53 3.3-1.21 3.3-1.21.66 1.65.24 2.87.12 3.17.77.83 1.24 1.88 1.24 3.17 0 4.54-2.81 5.54-5.49 5.83.43.36.81 1.08.81 2.18 0 1.57-.01 2.84-.01 3.23 0 .31.22.68.83.56A12.02 12.02 0 0 0 24 12.29C24 5.78 18.63.5 12 .5z" />
          </svg>
          <span class="hidden sm:inline">{{ t('demo.banner.selfHost') }}</span>
        </a>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg
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
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useDemoMode } from '@/composables/useDemoMode'
import { useConfirm } from '@/composables/useConfirm'
import { LINKS as links } from '@/config/links'

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
