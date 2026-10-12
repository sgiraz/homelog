<template>
  <!-- One fixed layout whatever the state: presets and dates are both always
       on screen, so picking one never moves the other. A single row with a
       pointer; on a phone the two groups stack, because 16px dates (the size
       Safari does not zoom on) and two segments do not fit in one line. -->
  <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">
    <!-- Segmented control (HIG): equal segments, one sliding thumb, a radio
         group to assistive tech. No segment is selected for a custom range. -->
    <div
      role="radiogroup"
      :aria-label="t('dashboard.period.label')"
      class="relative grid grid-cols-2 w-full sm:inline-grid sm:w-auto p-0.5 rounded-[10px] bg-surface-2"
      @keydown="onKeydown"
    >
      <span
        aria-hidden="true"
        class="absolute top-0.5 bottom-0.5 left-0.5 rounded-lg bg-surface dark:bg-surface-3 shadow-sm ring-1 ring-black/5 dark:ring-white/10 motion-safe:transition motion-safe:duration-200 motion-safe:ease-out"
        :class="activeIndex < 0 ? 'opacity-0' : 'opacity-100'"
        :style="{ width: 'calc((100% - 4px) / 2)', transform: `translateX(${Math.max(activeIndex, 0) * 100}%)` }"
      />
      <button
        v-for="(preset, i) in PERIOD_PRESETS"
        :key="preset.key"
        :ref="(el) => { buttons[i] = el }"
        type="button"
        role="radio"
        :aria-checked="mode === preset.key"
        :tabindex="i === Math.max(activeIndex, 0) ? 0 : -1"
        class="relative z-10 min-w-0 h-11 sm:h-9 px-3 sm:px-5 rounded-lg text-[13px] font-semibold truncate transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
        :class="mode === preset.key ? 'text-ink' : 'text-ink-soft hover:text-ink'"
        @click="choose(preset.key)"
      >
        {{ t(`dashboard.period.${preset.key}`) }}
      </button>
    </div>

    <DateRangeField
      :from="from"
      :to="to"
      :from-label="t('dashboard.period.from')"
      :to-label="t('dashboard.period.to')"
      @update:range="(range) => emit('update:range', range)"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'PeriodPicker' })

import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { PERIOD_PRESETS } from '@/utils/periodPresets'
import DateRangeField from '@/components/common/DateRangeField.vue'

const { t } = useI18n()

const props = defineProps({
  /** A preset key, or 'custom'. */
  mode: {
    type: String,
    required: true
  },
  /** Active range, "YYYY-MM-DD". */
  from: {
    type: String,
    required: true
  },
  to: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['select', 'update:range'])

const buttons = []
const activeIndex = computed(() => PERIOD_PRESETS.findIndex((p) => p.key === props.mode))

function choose(key) {
  if (key !== props.mode) emit('select', key)
}

// Radio group keyboard model: arrows move the selection and the focus together.
function onKeydown(event) {
  const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key]
  if (!step) return
  event.preventDefault()
  const count = PERIOD_PRESETS.length
  const next = (Math.max(activeIndex.value, 0) + step + count) % count
  choose(PERIOD_PRESETS[next].key)
  buttons[next]?.focus()
}

</script>

