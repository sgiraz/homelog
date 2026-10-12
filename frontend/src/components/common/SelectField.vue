<template>
  <!-- The frame a <select> wears so a row of filters reads as a row of
       identical controls: same border, height and radius as DateRangeField,
       label inside, one chevron. The select itself stays in the caller's
       markup, so v-model keeps its option values untouched. -->
  <label
    class="field-shell flex items-center gap-2 min-w-0 h-11 sm:h-[38px] px-3 rounded-[10px] border border-line bg-surface cursor-pointer focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 focus-within:ring-offset-canvas transition-shadow"
    :class="disabled ? 'opacity-50 cursor-not-allowed' : ''"
  >
    <span class="text-[13px] font-medium text-ink-muted whitespace-nowrap">{{ label }}</span>
    <span class="relative flex items-center min-w-0 flex-1 gap-1">
      <slot />
      <svg class="pointer-events-none shrink-0 w-4 h-4 text-ink-faint" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
      </svg>
    </span>
  </label>
</template>

<script setup>
defineOptions({ name: 'SelectField' })

defineProps({
  label: {
    type: String,
    required: true
  },
  /** Mirrors the inner select's own disabled state, which it keeps. */
  disabled: {
    type: Boolean,
    default: false
  }
})
</script>

<style scoped>
/* The bordered label is the visible control; the native select inside is bare,
   and the focus ring moves to the frame. 16px on a phone so Safari does not
   zoom on focus. A transparent select would leave Chrome drawing its popup
   white, so color-scheme (inherited by the select) and the option colours put
   the native list on the app's theme instead of the OS one. */
.field-shell {
  color-scheme: light;
}
.dark .field-shell {
  color-scheme: dark;
}
:deep(select) {
  appearance: none;
  -webkit-appearance: none;
  min-width: 0;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: rgb(var(--c-ink));
  font-size: 1rem;
  font-weight: 600;
  line-height: 1.375;
  text-overflow: ellipsis;
  cursor: inherit;
}
:deep(select option) {
  background-color: rgb(var(--c-surface));
  color: rgb(var(--c-ink));
}
:deep(select:focus),
:deep(select:focus-visible) {
  outline: none;
  /* The frame wears the focus ring; the bare select must not draw a second
     one inside it (main.css rings every :focus-visible). */
  box-shadow: none;
}
:deep(select:disabled) {
  cursor: not-allowed;
}
@media (min-width: 640px) {
  :deep(select) {
    font-size: 0.875rem;
  }
}
</style>
