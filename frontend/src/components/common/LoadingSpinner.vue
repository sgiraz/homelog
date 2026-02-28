<template>
  <div :class="['flex items-center justify-center', containerClass]">
    <div
      :class="[
        'animate-spin rounded-full border-2 border-t-transparent',
        sizeClass,
        colorClass
      ]"
      role="status"
      aria-label="Caricamento..."
    ></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: String,
    default: 'md',
    validator: (v) => ['sm', 'md', 'lg', 'xl'].includes(v)
  },
  color: {
    type: String,
    default: 'blue',
    validator: (v) => ['blue', 'gray', 'white'].includes(v)
  },
  containerClass: {
    type: String,
    default: ''
  }
})

const sizeClass = computed(() => ({
  sm: 'w-4 h-4',
  md: 'w-8 h-8',
  lg: 'w-12 h-12',
  xl: 'w-16 h-16',
})[props.size])

const colorClass = computed(() => ({
  blue:  'border-blue-600',
  gray:  'border-gray-400 dark:border-gray-500',
  white: 'border-white',
})[props.color])
</script>
