<template>
  <BaseModal title="Ritaglia foto" @close="$emit('close')">
    <div class="space-y-5">
      <!-- Cropper area -->
      <div class="relative rounded-xl overflow-hidden bg-gray-950"
           :style="{ height: cropperHeight + 'px' }">
        <Cropper
          ref="cropperRef"
          :src="imageSrc"
          :stencil-props="{ aspectRatio: 1 }"
          :stencil-component="CircleStencil"
          :resize-image="{ adjustStencil: false }"
          image-restriction="stencil"
          class="h-full"
        />
      </div>

      <!-- Zoom control -->
      <div class="flex items-center gap-3 px-1">
        <button
          @click="adjustZoom(-0.15)"
          class="p-2 rounded-full text-gray-500 dark:text-gray-400
                 hover:bg-gray-100 dark:hover:bg-gray-700 active:bg-gray-200 dark:active:bg-gray-600
                 transition-colors"
          aria-label="Riduci zoom"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7" />
          </svg>
        </button>
        <input
          type="range"
          min="0"
          max="1"
          step="0.005"
          v-model.number="zoomLevel"
          @input="onSliderZoom"
          class="flex-1 h-1 bg-gray-200 dark:bg-gray-700 rounded-full appearance-none cursor-pointer
                 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-6
                 [&::-webkit-slider-thumb]:h-6 [&::-webkit-slider-thumb]:bg-white
                 [&::-webkit-slider-thumb]:border-2 [&::-webkit-slider-thumb]:border-blue-500
                 [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:cursor-pointer
                 [&::-webkit-slider-thumb]:shadow-md"
        />
        <button
          @click="adjustZoom(0.15)"
          class="p-2 rounded-full text-gray-500 dark:text-gray-400
                 hover:bg-gray-100 dark:hover:bg-gray-700 active:bg-gray-200 dark:active:bg-gray-600
                 transition-colors"
          aria-label="Aumenta zoom"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7" />
          </svg>
        </button>
      </div>

      <!-- Action buttons -->
      <div class="flex gap-3">
        <Button variant="secondary" class="flex-1" @click="$emit('close')">
          Annulla
        </Button>
        <Button class="flex-1" :disabled="saving" @click="handleSave">
          {{ saving ? 'Salvataggio...' : 'Salva' }}
        </Button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Cropper, CircleStencil } from 'vue-advanced-cropper'
import 'vue-advanced-cropper/dist/style.css'
import BaseModal from './BaseModal.vue'
import Button from './Button.vue'

defineProps({
  imageSrc: { type: String, required: true },
})

const emit = defineEmits(['close', 'cropped'])

const cropperRef = ref(null)
const zoomLevel = ref(0)
const saving = ref(false)
const windowWidth = ref(window.innerWidth)

// Responsive cropper height: taller on desktop, compact on mobile
const cropperHeight = computed(() => {
  return windowWidth.value >= 640 ? 380 : 280
})

function onResize() { windowWidth.value = window.innerWidth }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

let lastSliderValue = 0

function onSliderZoom() {
  const cropper = cropperRef.value
  if (!cropper) return
  const delta = zoomLevel.value - lastSliderValue
  if (Math.abs(delta) > 0.001) {
    // Convert slider delta to zoom factor: positive = zoom in, negative = zoom out
    cropper.zoom(1 + delta * 3)
  }
  lastSliderValue = zoomLevel.value
}

function adjustZoom(delta) {
  const newVal = Math.max(0, Math.min(1, zoomLevel.value + delta))
  const sliderDelta = newVal - zoomLevel.value
  zoomLevel.value = newVal
  const cropper = cropperRef.value
  if (cropper && Math.abs(sliderDelta) > 0.001) {
    cropper.zoom(1 + sliderDelta * 3)
  }
  lastSliderValue = zoomLevel.value
}

function handleSave() {
  const cropper = cropperRef.value
  if (!cropper) return

  saving.value = true
  const { canvas } = cropper.getResult()
  if (!canvas) {
    saving.value = false
    return
  }

  // Export at 512x512 for sharp rendering on retina displays
  const outCanvas = document.createElement('canvas')
  outCanvas.width = 512
  outCanvas.height = 512
  const ctx = outCanvas.getContext('2d')
  ctx.drawImage(canvas, 0, 0, 512, 512)

  outCanvas.toBlob(
    (blob) => {
      if (blob) {
        const file = new File([blob], 'avatar.jpg', { type: 'image/jpeg' })
        emit('cropped', file)
      } else {
        saving.value = false
      }
    },
    'image/jpeg',
    0.92
  )
}
</script>

<style>
/* Darken area outside the circle stencil */
.vue-advanced-cropper__background,
.vue-advanced-cropper__foreground {
  background: rgb(3 7 18) !important;
}
</style>
