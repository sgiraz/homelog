<template>
  <!-- Expanded, the list is capped and scrolls: a household with thirty
       categories must not turn the card into a page-long column. -->
  <ul
    class="space-y-0.5"
    :class="expanded ? 'max-h-[22rem] overflow-y-auto overflow-x-hidden pr-1' : ''"
  >
    <li v-for="(row, i) in rows" :key="row.key">
      <component
        :is="row.clickable ? 'button' : 'div'"
        :type="row.clickable ? 'button' : undefined"
        class="w-full text-left px-2 py-2 rounded-lg transition-colors"
        :class="row.clickable ? 'hover:bg-surface-2 active:bg-surface-3 cursor-pointer' : ''"
        :aria-label="row.clickable ? row.aria : undefined"
        @click="row.clickable && emit('slice-click', i)"
      >
        <div class="flex items-baseline justify-between gap-3">
          <span class="text-sm text-ink truncate">{{ row.label }}</span>
          <span class="text-sm tabular-nums shrink-0">
            <span class="font-medium text-ink">{{ formatCurrency(row.amount) }}</span>
            <span class="text-ink-muted ml-1.5">{{ shareLabel(row.share) }}</span>
          </span>
        </div>
        <div class="h-2 mt-1.5 rounded-full bg-surface-2 overflow-hidden">
          <div
            class="h-full rounded-full"
            :style="{ width: barWidth(row.amount), backgroundColor: row.color }"
          ></div>
        </div>
      </component>
    </li>
  </ul>

  <!-- The expander is the only way past the default limit: a long tail is a
       deliberate second look, not something to dump on every visit. -->
  <button
    v-if="totalCount > rows.length || expanded"
    type="button"
    class="mt-2 ml-2 text-sm text-accent-soft hover:underline"
    @click="emit('update:expanded', !expanded)"
  >
    {{ expanded ? t('common.actions.showLess') : t('common.actions.showAll', { count: totalCount }) }}
  </button>
</template>

<script setup>
defineOptions({ name: 'CategoryBars' })

import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  /**
   * Rows in the order they are drawn, biggest first:
   * { key, label, amount, share (0-100), color, clickable, aria }
   */
  rows: {
    type: Array,
    required: true
  },
  formatCurrency: {
    type: Function,
    required: true
  },
  /** How many categories exist in total, folded ones included. */
  totalCount: {
    type: Number,
    required: true
  },
  expanded: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['slice-click', 'update:expanded'])

// Bars are scaled to the largest row, not to the total: the job here is
// ranking, and against the total everything below the leader collapses into
// slivers. Share of the total is the number on the right, which reads more
// precisely than any bar length would.
const maxAmount = computed(() => Math.max(...props.rows.map((r) => r.amount), 0))

function barWidth(amount) {
  if (maxAmount.value <= 0) return '0%'
  // A row that exists must be visible, however small its amount.
  return `${Math.max((amount / maxAmount.value) * 100, 1.5)}%`
}

/** A non-zero share must never read "0%". */
function shareLabel(share) {
  return share > 0 && share < 1 ? '<1%' : `${Math.round(share)}%`
}
</script>
