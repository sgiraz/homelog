<template>
  <Transition
    enter-active-class="transition ease-out duration-200"
    enter-from-class="translate-y-4 opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition ease-in duration-150"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-4 opacity-0"
  >
    <div
      v-if="needRefresh"
      role="alert"
      aria-live="polite"
      class="fixed bottom-20 md:bottom-4 left-4 right-4 md:left-auto md:right-4 md:max-w-sm
             bg-surface border border-line
             rounded-xl shadow-lg p-4 z-40 flex items-start gap-3"
    >
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-ink">
          {{ $t('common.pwa.updateAvailable') }}
        </p>
        <p class="mt-1 text-xs text-ink-soft">
          {{ $t('common.pwa.updateHint') }}
        </p>
      </div>
      <div class="flex flex-col gap-2 shrink-0">
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium"
          @click="apply"
        >
          {{ $t('common.pwa.update') }}
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-ink-soft text-sm hover:bg-surface-2"
          @click="dismiss"
        >
          {{ $t('common.pwa.later') }}
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRegisterSW } from 'virtual:pwa-register/vue'

const needRefresh = ref(false)
let updateSW = null

onMounted(() => {
  const reg = useRegisterSW({
    immediate: true,
    onNeedRefresh() {
      needRefresh.value = true
    },
  })
  updateSW = reg.updateServiceWorker
  needRefresh.value = reg.needRefresh.value
})

function apply() {
  if (updateSW) updateSW(true)
}

function dismiss() {
  needRefresh.value = false
}
</script>
