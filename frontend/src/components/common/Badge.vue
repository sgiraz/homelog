<template>
  <span
    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap"
    :class="VARIANTS[variant]"
  >
    <slot />
  </span>
</template>

<script setup>
defineOptions({ name: 'Badge' })

import { BADGE_VARIANTS } from '@/utils/badgeVariants'

/**
 * Status chip. Every badge in the app goes through here so a state has one
 * colour wherever it appears — the hand-rolled classes it replaces had
 * "pending" as amber in one view and yellow in another.
 */
const VARIANTS = BADGE_VARIANTS

defineProps({
  variant: {
    type: String,
    default: 'neutral',
    validator: (value) => value in BADGE_VARIANTS
  }
})
</script>
