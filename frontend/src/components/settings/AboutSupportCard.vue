<template>
  <Card class="p-4 sm:p-6">
    <div class="flex items-start gap-3">
      <div class="text-2xl leading-none mt-0.5">🏠</div>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('settings.about.title') }}</h3>
          <span v-if="appVersion" class="text-xs text-gray-400 dark:text-gray-500 tabular-nums">v{{ appVersion }}</span>
          <a
            v-if="updateAvailable"
            :href="latestUrl || links.github"
            target="_blank"
            rel="noopener"
            class="text-xs font-medium text-amber-600 dark:text-amber-400 hover:underline"
          >
            {{ t('settings.about.updateAvailable') }}
          </a>
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ t('settings.about.tagline') }}</p>
        <div class="flex flex-wrap gap-2 mt-3">
          <a
            :href="links.github"
            target="_blank"
            rel="noopener"
            class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium
                   bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200
                   hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M12 .5C5.37.5 0 5.78 0 12.29c0 5.21 3.44 9.63 8.21 11.19.6.11.82-.25.82-.56 0-.28-.01-1.02-.02-2-3.34.71-4.04-1.58-4.04-1.58-.55-1.37-1.34-1.74-1.34-1.74-1.09-.73.08-.72.08-.72 1.21.08 1.84 1.22 1.84 1.22 1.07 1.8 2.81 1.28 3.5.98.11-.76.42-1.28.76-1.57-2.67-.3-5.47-1.31-5.47-5.84 0-1.29.47-2.34 1.24-3.17-.12-.3-.54-1.52.12-3.17 0 0 1.01-.32 3.3 1.21a11.6 11.6 0 0 1 3-.39c1.02 0 2.05.13 3 .39 2.29-1.53 3.3-1.21 3.3-1.21.66 1.65.24 2.87.12 3.17.77.83 1.24 1.88 1.24 3.17 0 4.54-2.81 5.54-5.49 5.83.43.36.81 1.08.81 2.18 0 1.57-.01 2.84-.01 3.23 0 .31.22.68.83.56A12.02 12.02 0 0 0 24 12.29C24 5.78 18.63.5 12 .5z" />
            </svg>
            {{ t('settings.about.star') }}
          </a>
          <a
            :href="links.kofi"
            target="_blank"
            rel="noopener"
            class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium
                   bg-pink-100 dark:bg-pink-900/40 text-pink-700 dark:text-pink-300
                   hover:bg-pink-200 dark:hover:bg-pink-900/60 transition-colors"
          >
            ❤️ {{ t('settings.about.donate') }}
          </a>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useDemoMode } from '@/composables/useDemoMode'
import { LINKS as links } from '@/config/links'
import Card from '@/components/common/Card.vue'

const { t } = useI18n()
// Version/update info comes from the single /version fetch done at app startup.
const { appVersion, updateAvailable, latestUrl } = useDemoMode()
</script>
