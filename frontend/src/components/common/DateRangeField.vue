<template>
  <!-- One bordered field split in two, the system date picker behind each half.
       Label above the value on a phone (each half stays a full-width 44pt
       target), beside it with a pointer. Shared by the dashboard period control
       and the expense filters so a date reads the same wherever it appears. -->
  <div class="grid grid-cols-2 sm:inline-grid rounded-[10px] border border-line bg-surface divide-x divide-line">
    <label
      v-for="field in FIELDS"
      :key="field"
      class="flex flex-col justify-center min-w-0 px-3 min-h-[44px] sm:min-h-0 sm:h-[38px] sm:flex-row sm:items-center sm:justify-start sm:gap-2 cursor-pointer relative focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 focus-within:ring-offset-canvas first:rounded-l-[9px] last:rounded-r-[9px] transition-shadow"
    >
      <span class="text-[11px] sm:text-[13px] font-medium leading-tight text-ink-muted">
        {{ field === 'from' ? fromLabel : toLabel }}
      </span>
      <input
        type="date"
        :value="field === 'from' ? from : to"
        :min="field === 'to' ? from || undefined : undefined"
        :max="field === 'from' ? to || undefined : undefined"
        class="date-field"
        @change="onDateChange(field, $event)"
      />
    </label>
  </div>
</template>

<script setup>
defineOptions({ name: 'DateRangeField' })

const FIELDS = ['from', 'to']

const props = defineProps({
  /** Range bounds, "YYYY-MM-DD"; empty means "unset" and is only reachable when clearable. */
  from: {
    type: String,
    default: ''
  },
  to: {
    type: String,
    default: ''
  },
  fromLabel: {
    type: String,
    required: true
  },
  toLabel: {
    type: String,
    required: true
  },
  /** Whether emptying a half is a valid state (a filter) or must be undone (a period). */
  clearable: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:range'])

// A cleared field (iOS has a Reset button in its picker) means nothing to a
// control that always needs a range: put the old value back. Bounds are hinted
// with min/max, but typed input can still cross them, so an inverted range is
// swapped, not rejected.
function onDateChange(field, event) {
  const value = event.target.value
  if (!value && !props.clearable) {
    event.target.value = props[field]
    return
  }
  let { from, to } = props
  if (field === 'from') from = value
  else to = value
  if (from && to && from > to) [from, to] = [to, from]
  emit('update:range', { from, to })
}
</script>

<style scoped>
/* The bordered group is the visible control; the native input inside is bare.
   16px on a phone so Safari does not zoom on focus; color-scheme makes the
   system picker follow the app theme instead of the OS one. */
.date-field {
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
  font-variant-numeric: tabular-nums;
  line-height: 1.375;
  color-scheme: light;
  cursor: pointer;
}
.dark .date-field {
  color-scheme: dark;
}
.date-field::-webkit-date-and-time-value {
  text-align: left;
  min-height: 1.375em;
}
.date-field:focus,
.date-field:focus-visible {
  outline: none;
  /* The frame wears the focus ring; the bare input must not draw a second one
     inside it (main.css rings every :focus-visible). */
  box-shadow: none;
}
@media (min-width: 640px) {
  .date-field {
    width: auto;
    font-size: 0.875rem;
  }
}
</style>
