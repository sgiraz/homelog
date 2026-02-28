<template>
  <div class="text-center py-16 px-4">
    <div class="text-6xl mb-4 select-none">{{ icon }}</div>
    <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
      {{ title }}
    </h3>
    <p class="text-gray-500 dark:text-gray-400 mb-6 max-w-md mx-auto text-sm leading-relaxed">
      {{ description }}
    </p>
    <slot name="action">
      <button
        v-if="actionText"
        @click="$emit('action')"
        class="inline-flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
      >
        {{ actionText }}
      </button>
    </slot>
  </div>
</template>

<script setup>
defineProps({
  icon:        { type: String, default: '📭' },
  title:       { type: String, required: true },
  description: { type: String, required: true },
  actionText:  { type: String, default: null },
})

defineEmits(['action'])
</script>
