<template>
  <Card
    class="p-4 sm:p-5 border-l-4 border-l-accent transition-colors"
    :class="{ 'attention-warn': items.length }"
  >
    <!-- Header: warm "home brief" identity + dynamic subtitle -->
    <div class="flex items-center gap-3 pb-3 mb-3 border-b border-line/60">
      <span class="w-9 h-9 rounded-xl grid place-items-center shrink-0 bg-accent-soft/15 text-accent-soft">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
        </svg>
      </span>
      <div class="min-w-0">
        <h2 class="text-sm font-semibold text-ink leading-tight">{{ t('dashboard.attention.title') }}</h2>
        <p class="text-xs text-ink-muted leading-tight">
          {{ items.length
            ? t(`dashboard.attention.${items.length === 1 ? 'subtitle_one' : 'subtitle_other'}`, { n: items.length })
            : t('dashboard.attention.allClear') }}
        </p>
      </div>
    </div>

    <!-- Calm "all clear" state -->
    <div v-if="!items.length" class="flex items-start gap-3 py-1">
      <span class="w-8 h-8 rounded-lg grid place-items-center shrink-0 bg-positive/15 text-positive">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
      </span>
      <div class="min-w-0">
        <div class="text-sm text-ink">{{ t('dashboard.attention.allClearSub') }}</div>
        <div v-if="emptyDetail" class="text-xs text-ink-faint mt-0.5">{{ emptyDetail }}</div>
      </div>
    </div>

    <!-- Concrete items (3 on mobile, 5 on desktop) -->
    <template v-else>
      <ul class="divide-y divide-line/60">
        <li
          v-for="(item, idx) in items"
          :key="item.key"
          class="flex items-center gap-3 py-2.5"
          :class="{ 'hidden md:flex': idx >= 3 && !showAllMobile }"
        >
          <span :class="['w-8 h-8 rounded-lg grid place-items-center shrink-0', toneChip[item.tone]]">
            <component :is="icons[item.icon]" />
          </span>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium text-ink truncate">{{ item.title }}</div>
            <div class="text-xs text-ink-muted truncate">{{ item.detail }}</div>
          </div>
          <component
            :is="item.to ? 'router-link' : 'button'"
            :to="item.to"
            type="button"
            class="action-pill shrink-0"
            @click="!item.to && $emit('action', item)"
          >
            {{ item.action }}
          </component>
        </li>
      </ul>

      <!-- Mobile-only overflow reveal -->
      <button
        v-if="items.length > 3 && !showAllMobile"
        type="button"
        class="md:hidden mt-2 text-xs font-medium text-ink-muted hover:text-ink"
        @click="showAllMobile = true"
      >
        {{ t(`dashboard.attention.${items.length - 3 === 1 ? 'more_one' : 'more_other'}`, { n: items.length - 3 }) }}
      </button>
    </template>
  </Card>
</template>

<script setup>
defineOptions({ name: 'AttentionPanel' })

import { ref, h } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'

const { t } = useI18n()

defineProps({
  // [{ key, tone, icon, title, detail, action, to? }] — to omitted → emits 'action'
  items: {
    type: Array,
    default: () => []
  },
  // Optional micro-line shown under the "all clear" state
  emptyDetail: {
    type: String,
    default: ''
  }
})

defineEmits(['action'])

const showAllMobile = ref(false)

// Tone → chip color classes
const toneChip = {
  danger: 'bg-red-100 text-red-600 dark:bg-red-900/40 dark:text-red-300',
  warn: 'bg-amber-100 text-amber-600 dark:bg-amber-900/40 dark:text-amber-300',
  info: 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300',
  positive: 'bg-positive/15 text-positive',
  accent: 'bg-accent-soft/15 text-accent-soft',
}

// Small inline icon set keyed by item.icon
function svg(path) {
  return {
    render() {
      return h('svg', { class: 'w-4 h-4', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
        h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: path })
      ])
    }
  }
}

const icons = {
  bill: svg('M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z'),
  balance: svg('M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z'),
  reading: svg('M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z'),
  project: svg('M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z'),
}
</script>

<style scoped>
.action-pill {
  display: inline-flex;
  align-items: center;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.375rem 0.75rem;
  border-radius: 0.5rem;
  background: rgb(243 244 246); /* gray-100 */
  color: rgb(55 65 81); /* gray-700 */
  transition: background-color 0.15s ease;
  cursor: pointer;
}
.action-pill:hover {
  background: rgb(229 231 235); /* gray-200 */
}
.dark .action-pill {
  background: rgb(55 65 81); /* gray-700 */
  color: rgb(229 231 235); /* gray-200 */
}
.dark .action-pill:hover {
  background: rgb(75 85 99); /* gray-600 */
}

/* Attention state: warm amber wash over the whole panel + amber left border.
   Scoped (→ [data-v]) so it beats the Card's bg-surface utility without
   needing !important or relying on Tailwind v4 utility ordering. */
.attention-warn {
  background-color: rgb(255 251 235); /* amber-50 */
  border-left-color: rgb(245 158 11); /* amber-500 */
}
.dark .attention-warn {
  background-color: rgb(120 53 15 / 0.12); /* warm amber wash */
  border-left-color: rgb(217 119 6); /* amber-600 */
}
</style>
