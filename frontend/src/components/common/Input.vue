<template>
  <div class="min-w-0 w-full">
    <label v-if="label" :for="id" class="block text-sm text-ink-soft mb-1">
      {{ label }}
    </label>
    <input
      :id="id"
      :type="isDecimal ? 'text' : type"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :step="step"
      :min="min"
      :autocomplete="autocomplete"
      :inputmode="inputmode"
      :disabled="disabled"
      :pattern="isDecimal ? '[0-9]*[.,]?[0-9]*' : undefined"
      @input="handleInput"
      class="w-full min-w-0 max-w-full box-border px-3 py-3 border border-line rounded-lg
             bg-surface text-ink text-base
             focus:outline-none focus:ring-2 focus:ring-blue-500
             disabled:opacity-50 disabled:cursor-not-allowed disabled:bg-surface-2"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: [String, Number],
  label: String,
  type: { type: String, default: 'text' },
  placeholder: String,
  required: Boolean,
  id: String,
  step: String,
  min: String,
  autocomplete: String,
  inputmode: String,
  disabled: Boolean
})

const emit = defineEmits(['update:modelValue'])

const isDecimal = computed(() => props.inputmode === 'decimal')

function handleInput(e) {
  const raw = e.target.value
  if (isDecimal.value || props.type === 'number') {
    if (raw === '') {
      emit('update:modelValue', null)
    } else {
      const normalized = raw.replace(',', '.')
      const num = parseFloat(normalized)
      emit('update:modelValue', isNaN(num) ? raw : num)
    }
  } else {
    emit('update:modelValue', raw)
  }
}
</script>

<style scoped>
input[type="date"] {
  width: 0;
  min-width: 100%;
}
input[type="date"]::-webkit-date-and-time-value {
  min-height: 1.5em;
}
</style>
