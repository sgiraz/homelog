<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <button
      v-for="preset in PERIOD_PRESETS"
      :key="preset.key"
      type="button"
      class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
      :class="active === preset.key
        ? 'bg-accent/10 text-accent-soft'
        : 'text-ink-soft hover:bg-surface-2'"
      @click="emit('select', preset)"
    >
      {{ t(`dashboard.period.${preset.key}`) }}
    </button>
    <!-- Shown only once the dates stop matching a preset: it is a state, not a
         choice, so it reads as a label and opens the filters instead. -->
    <button
      v-if="active === null"
      type="button"
      class="px-3 py-1.5 rounded-lg text-sm font-medium bg-accent/10 text-accent-soft"
      @click="emit('open-filters')"
    >
      {{ t('dashboard.period.custom') }}
    </button>
  </div>
</template>

<script setup>
defineOptions({ name: 'PeriodChips' })

import { useI18n } from 'vue-i18n'
import { PERIOD_PRESETS } from '@/utils/periodPresets'

const { t } = useI18n()

defineProps({
  /** Key of the preset matching the active dates, or null when custom. */
  active: {
    type: String,
    default: null
  }
})

const emit = defineEmits(['select', 'open-filters'])
</script>
