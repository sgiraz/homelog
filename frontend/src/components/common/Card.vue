<template>
  <div :class="[
    'bg-surface rounded-xl border border-line',
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
