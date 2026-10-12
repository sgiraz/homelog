<template>
  <!-- Nothing to do is not news: one quiet line, no card. -->
  <div
    v-if="!items.length"
    class="flex items-center gap-2 px-1 text-sm min-w-0"
    :title="t('dashboard.attention.allClearSub')"
  >
    <svg class="w-4 h-4 shrink-0 text-positive-soft" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
    </svg>
    <span class="font-medium text-ink shrink-0">{{ t('dashboard.attention.allClear') }}</span>
    <span class="text-ink-muted truncate">· {{ emptyDetail || t('dashboard.attention.allClearSub') }}</span>
  </div>

  <!-- A bar, not a panel: the most urgent item is always on screen (items come
       sorted by urgency), the rest wait one tap away. The urgency colour lives
       in each row's chip, so the bar itself needs no header. -->
  <Card
    v-else
    role="region"
    :aria-label="t('dashboard.attention.title')"
    class="px-3 py-1 sm:px-4 border-l-4 border-l-accent"
  >
    <ul :id="listId">
      <li
        v-for="(item, idx) in visibleItems"
        :key="item.key"
        class="flex items-center gap-3 py-2"
        :class="{ 'border-t border-line/60': idx > 0 }"
      >
        <span :class="['w-8 h-8 rounded-lg grid place-items-center shrink-0', toneChip[item.tone]]">
          <component :is="icons[item.icon]" />
        </span>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium text-ink truncate">{{ item.title }}</div>
          <div class="text-xs truncate" :class="toneText[item.tone] || 'text-ink-muted'">{{ item.detail }}</div>
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
        <button
          v-if="idx === 0 && hiddenCount > 0"
          type="button"
          class="shrink-0 inline-flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-semibold text-ink-muted hover:text-ink hover:bg-surface-2 transition-colors"
          :aria-expanded="false"
          :aria-controls="listId"
          @click="expanded = true"
        >
          {{ t(`dashboard.attention.${hiddenCount === 1 ? 'more_one' : 'more_other'}`, { n: hiddenCount }) }}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </li>
    </ul>

    <div v-if="expanded && items.length > 1" class="flex justify-end pb-2">
      <button
        type="button"
        class="inline-flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-semibold text-ink-muted hover:text-ink hover:bg-surface-2 transition-colors"
        :aria-expanded="true"
        :aria-controls="listId"
        @click="expanded = false"
      >
        {{ t('dashboard.attention.showLess') }}
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 15l7-7 7 7" />
        </svg>
      </button>
    </div>
  </Card>
</template>

<script setup>
defineOptions({ name: 'AttentionPanel' })

import { ref, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from '@/components/common/Card.vue'

const { t } = useI18n()

const props = defineProps({
  // [{ key, tone, icon, title, detail, action, to? }] — to omitted → emits 'action'
  items: {
    type: Array,
    default: () => []
  },
  // Optional micro-line shown beside "all clear"
  emptyDetail: {
    type: String,
    default: ''
  }
})

defineEmits(['action'])

const listId = 'attention-items'
const expanded = ref(false)

const visibleItems = computed(() => (expanded.value ? props.items : props.items.slice(0, 1)))
const hiddenCount = computed(() => props.items.length - visibleItems.value.length)

// Tone → chip color classes (the left status chip carries the urgency colour)
const toneChip = {
  danger: 'bg-danger/10 text-danger-soft',
  warn: 'bg-warning/10 text-warning-soft',
  info: 'bg-info/10 text-info-soft',
  positive: 'bg-positive/10 text-positive-soft',
  accent: 'bg-accent-soft/15 text-accent-soft',
}

// Tone → status-line color. Only urgent/actionable tones tint the detail text;
// 'info' (e.g. readings) stays muted so it doesn't read like a link.
const toneText = {
  danger: 'text-danger-soft',
  warn: 'text-warning-soft',
  positive: 'text-positive',
  accent: 'text-accent-soft',
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
</style>
