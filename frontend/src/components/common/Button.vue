<template>
  <button
    :type="type"
    :class="[
      'rounded-lg font-medium transition-colors inline-flex items-center justify-center',
      sizeClasses[size],
      variants[variant],
      { 'opacity-50 cursor-not-allowed': disabled }
    ]"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>

<script setup>
defineProps({
  variant: {
    type: String,
    default: 'primary',
    validator: (v) => ['primary', 'secondary', 'danger', 'success'].includes(v)
  },
  size: {
    type: String,
    default: 'md',
    validator: (v) => ['md', 'sm'].includes(v)
  },
  disabled: Boolean,
  type: {
    type: String,
    default: 'button'
  }
})

const sizeClasses = {
  md: 'px-4 py-3',
  sm: 'px-3 py-2'
}

const variants = {
  primary: 'bg-ember hover:bg-ember-deep text-white',
  secondary: 'bg-surface-3 hover:bg-surface-2 text-ink',
  danger: 'bg-red-600 hover:bg-red-700 text-white',
  success: 'bg-green-600 hover:bg-green-700 text-white'
}
</script>
