<template>
  <div :class="[
    'bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700',
    'backdrop-blur-xl bg-opacity-90',
    className
  ]">
    <slot />
  </div>
</template>

<script setup>
defineProps({
  className: String
})
</script>
