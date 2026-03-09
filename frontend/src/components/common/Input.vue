<template>
  <div>
    <label v-if="label" :for="id" class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
      {{ label }}
    </label>
    <input
      :id="id"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :step="step"
      :min="min"
      :autocomplete="autocomplete"
      :inputmode="inputmode"
      :disabled="disabled"
      @input="$emit('update:modelValue', type === 'number' ? ($event.target.value === '' ? null : parseFloat($event.target.value)) : $event.target.value)"
      class="w-full min-w-0 max-w-full box-border px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
             bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
             focus:outline-none focus:ring-2 focus:ring-blue-500
             disabled:opacity-50 disabled:cursor-not-allowed disabled:bg-gray-100 dark:disabled:bg-gray-700"
    />
  </div>
</template>

<script setup>
defineProps({
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

defineEmits(['update:modelValue'])
</script>
