<template>
  <div class="flex flex-col h-full">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-3 py-2 bg-gray-100 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 rounded-t-lg">
      <!-- Page navigation -->
      <div class="flex items-center gap-2">
        <button
          @click="prevPage"
          :disabled="currentPage <= 1"
          class="p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          title="Pagina precedente"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300 min-w-[80px] text-center">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          @click="nextPage"
          :disabled="currentPage >= totalPages"
          class="p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          title="Pagina successiva"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      <!-- Zoom controls -->
      <div class="flex items-center gap-2">
        <button
          @click="zoomOut"
          :disabled="zoom <= 0.5"
          class="p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          title="Riduci"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
          </svg>
        </button>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300 min-w-[50px] text-center">
          {{ Math.round(zoom * 100) }}%
        </span>
        <button
          @click="zoomIn"
          :disabled="zoom >= 2"
          class="p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          title="Ingrandisci"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </button>
        <button
          @click="resetZoom"
          class="p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors text-xs font-medium text-gray-600 dark:text-gray-400"
          title="Reset zoom"
        >
          Fit
        </button>
      </div>

      <!-- Legend -->
      <div class="flex items-center gap-3 text-xs">
        <span class="flex items-center gap-1">
          <span class="w-3 h-3 rounded bg-green-400/60 border border-green-500"></span>
          Valuta
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-3 rounded bg-purple-400/60 border border-purple-500"></span>
          Data
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-3 rounded bg-blue-400/60 border border-blue-500"></span>
          Numero
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-3 rounded bg-yellow-400/60 border border-yellow-500"></span>
          Simbolo
        </span>
      </div>
    </div>

    <!-- PDF View Container -->
    <div
      ref="containerRef"
      class="flex-1 overflow-auto bg-gray-200 dark:bg-gray-900 p-4"
      @wheel.ctrl.prevent="handleWheel"
    >
      <div
        v-if="currentPageData"
        class="relative mx-auto shadow-lg"
        :style="{
          width: (currentPageData.image_width * zoom) + 'px',
          height: (currentPageData.image_height * zoom) + 'px'
        }"
      >
        <!-- PDF Image Background -->
        <img
          :src="apiBaseUrl + currentPageData.image_url"
          :alt="`Pagina ${currentPage}`"
          class="absolute inset-0 w-full h-full opacity-40 pointer-events-none select-none"
          draggable="false"
        />

        <!-- Words Overlay -->
        <div
          v-for="word in currentPageWords"
          :key="word.id"
          :class="[
            'absolute flex items-center justify-center cursor-grab transition-all group/word',
            'border rounded font-mono leading-none whitespace-nowrap',
            'hover:ring-2 hover:ring-blue-500 hover:z-20',
            getWordColorClass(word.type),
            selectedWordId === word.id ? 'ring-2 ring-blue-500 z-10' : '',
            mappedWordIds.has(word.id) ? 'opacity-30' : ''
          ]"
          :style="getWordStyle(word)"
          draggable="true"
          @dragstart="handleDragStart($event, word)"
          @dragend="handleDragEnd"
          @click="$emit('word-click', word)"
          @mouseenter="hoveredWord = word"
          @mouseleave="hoveredWord = null"
        >
          <span class="overflow-hidden text-ellipsis pointer-events-none">{{ word.text }}</span>

          <!-- Hover Tooltip -->
          <div
            class="hidden group-hover/word:block absolute left-0 bottom-full mb-1 z-50 pointer-events-none"
            style="min-width: 200px"
          >
            <div class="bg-gray-900 text-white text-xs rounded-lg shadow-xl p-2.5 space-y-1.5">
              <!-- Zoomed text -->
              <p class="text-sm font-bold font-mono truncate" :class="getTooltipTextColor(word.type)">
                {{ word.text }}
              </p>
              <hr class="border-gray-700">
              <!-- Properties -->
              <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                <span class="text-gray-400">Tipo</span>
                <span class="font-medium">{{ getTypeLabel(word.type) }}</span>
                <span class="text-gray-400">Pag.</span>
                <span>{{ (word.page || 0) + 1 }}</span>
                <span class="text-gray-400">Pos.</span>
                <span>x:{{ Math.round(word.x) }} y:{{ Math.round(word.y) }}</span>
                <span class="text-gray-400">Dim.</span>
                <span>{{ Math.round(word.width) }}x{{ Math.round(word.height) }}</span>
              </div>
              <!-- Neighbors -->
              <div v-if="hoveredWordNeighbors" class="pt-1 border-t border-gray-700">
                <p class="text-gray-400 mb-0.5">Contesto</p>
                <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                  <span v-if="hoveredWordNeighbors.left" class="text-gray-500">sx</span>
                  <span v-if="hoveredWordNeighbors.left" class="font-mono truncate">{{ hoveredWordNeighbors.left.text }}</span>
                  <span v-if="hoveredWordNeighbors.right" class="text-gray-500">dx</span>
                  <span v-if="hoveredWordNeighbors.right" class="font-mono truncate">{{ hoveredWordNeighbors.right.text }}</span>
                  <span v-if="hoveredWordNeighbors.above" class="text-gray-500">sopra</span>
                  <span v-if="hoveredWordNeighbors.above" class="font-mono truncate">{{ hoveredWordNeighbors.above.text }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center h-64">
        <svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- No data state -->
      <div v-if="!loading && !currentPageData" class="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p>Nessun PDF caricato</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { classifyToken, TokenType, findTokenNeighbors } from '@/utils/tokenizer'

const props = defineProps({
  pages: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  },
  mappedWordIds: {
    type: Set,
    default: () => new Set()
  },
  selectedWordId: {
    type: String,
    default: null
  }
})

const emit = defineEmits(['word-click', 'drag-start', 'drag-end'])

// API base URL for images
const apiBaseUrl = ''  // same origin via nginx proxy

// State
const containerRef = ref(null)
const currentPage = ref(1)
const zoom = ref(1)

// Computed
const totalPages = computed(() => props.pages.length)

const currentPageData = computed(() => {
  if (props.pages.length === 0) return null
  return props.pages[currentPage.value - 1]
})

const currentPageWords = computed(() => {
  if (!currentPageData.value) return []

  // Process words and add type classification + unique id
  // Filter out punctuation and noise (QR codes, URLs, encoded data)
  return (currentPageData.value.words || []).map((word, index) => ({
    ...word,
    id: `${currentPage.value}-${word.lineIndex}-${word.wordIndex}-${index}`,
    type: classifyToken(word.text)
  })).filter(w => w.type && w.type !== TokenType.PUNCTUATION && w.type !== TokenType.NOISE)
})

// Methods
function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

function zoomIn() {
  if (zoom.value < 2) {
    zoom.value = Math.min(2, zoom.value + 0.25)
  }
}

function zoomOut() {
  if (zoom.value > 0.5) {
    zoom.value = Math.max(0.5, zoom.value - 0.25)
  }
}

function resetZoom() {
  zoom.value = 1
}

function handleWheel(event) {
  // Ctrl+scroll to zoom
  if (event.deltaY < 0) {
    zoomIn()
  } else {
    zoomOut()
  }
}

// Hovered word state for tooltip
const hoveredWord = ref(null)

const hoveredWordNeighbors = computed(() => {
  if (!hoveredWord.value) return null
  return findTokenNeighbors(hoveredWord.value, currentPageWords.value, true)
})

function getWordStyle(word) {
  // Calculate position and size based on coordinates and zoom
  const pageData = currentPageData.value
  if (!pageData) return {}

  // Scale factor from PDF coordinates to image pixels
  // pdftotext -bbox uses 72 DPI, pdftoppm uses 150 DPI
  const pdfDPI = 72
  const imageDPI = 150
  const scale = imageDPI / pdfDPI

  const widthPx = word.width * scale * zoom.value
  const heightPx = Math.max(word.height * scale * zoom.value, 14 * zoom.value)

  // Auto-fit font size: estimate based on box width and text length
  // Average monospace char width ≈ 0.6 × fontSize
  const charCount = word.text.length || 1
  const fittedByWidth = widthPx / (charCount * 0.62)
  const fittedByHeight = heightPx * 0.85
  const fittedSize = Math.min(fittedByWidth, fittedByHeight)
  const fontSize = Math.max(Math.min(fittedSize, 14 * zoom.value), 5)

  return {
    left: (word.x * scale * zoom.value) + 'px',
    top: (word.y * scale * zoom.value) + 'px',
    width: widthPx + 'px',
    height: heightPx + 'px',
    fontSize: fontSize + 'px'
  }
}

function getTypeLabel(type) {
  const labels = {
    [TokenType.CURRENCY]: 'Valuta',
    [TokenType.NUMBER]: 'Numero',
    [TokenType.DATE]: 'Data',
    [TokenType.MONTH]: 'Mese',
    [TokenType.SYMBOL]: 'Simbolo',
    [TokenType.TEXT]: 'Testo',
    [TokenType.POD]: 'POD',
    [TokenType.PDR]: 'PDR'
  }
  return labels[type] || type
}

function getTooltipTextColor(type) {
  switch (type) {
    case TokenType.CURRENCY: return 'text-green-400'
    case TokenType.DATE:
    case TokenType.MONTH: return 'text-purple-400'
    case TokenType.NUMBER: return 'text-blue-400'
    case TokenType.SYMBOL: return 'text-yellow-400'
    case TokenType.POD:
    case TokenType.PDR: return 'text-orange-400'
    default: return 'text-gray-200'
  }
}

function getWordColorClass(type) {
  switch (type) {
    case TokenType.CURRENCY:
      return 'bg-green-400/60 border-green-500 text-green-900 dark:text-green-100'
    case TokenType.DATE:
    case TokenType.MONTH:
      return 'bg-purple-400/60 border-purple-500 text-purple-900 dark:text-purple-100'
    case TokenType.NUMBER:
      return 'bg-blue-400/60 border-blue-500 text-blue-900 dark:text-blue-100'
    case TokenType.SYMBOL:
      return 'bg-yellow-400/60 border-yellow-500 text-yellow-900 dark:text-yellow-100'
    case TokenType.POD:
    case TokenType.PDR:
      return 'bg-orange-400/60 border-orange-500 text-orange-900 dark:text-orange-100'
    default:
      return 'bg-gray-300/60 border-gray-400 text-gray-800 dark:text-gray-200'
  }
}

function handleDragStart(event, word) {
  emit('drag-start', event, word)
  event.dataTransfer.setData('application/json', JSON.stringify(word))
  event.dataTransfer.effectAllowed = 'copy'
}

function handleDragEnd() {
  emit('drag-end')
}

// Watch for page changes and reset scroll
watch(currentPage, () => {
  if (containerRef.value) {
    containerRef.value.scrollTop = 0
  }
})

// Auto-fit zoom on mount
onMounted(() => {
  if (containerRef.value && currentPageData.value) {
    const containerWidth = containerRef.value.clientWidth - 32 // padding
    const pageWidth = currentPageData.value.image_width
    if (pageWidth > containerWidth) {
      zoom.value = Math.max(0.5, containerWidth / pageWidth)
    }
  }
})
</script>
