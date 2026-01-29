<template>
  <button
    :type="type"
    :class="[
      'px-4 py-2 rounded-lg font-medium transition-colors inline-flex items-center justify-center',
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
  disabled: Boolean,
  type: {
    type: String,
    default: 'button'
  }
})

const variants = {
  primary: 'bg-blue-600 hover:bg-blue-700 text-white',
  secondary: 'bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-900 dark:text-white',
  danger: 'bg-red-600 hover:bg-red-700 text-white',
  success: 'bg-green-600 hover:bg-green-700 text-white'
}
</script>
