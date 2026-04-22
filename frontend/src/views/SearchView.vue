<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 pb-safe">
    <!-- Header: back + search input (Apple HIG pattern) -->
    <div
      class="sticky top-0 z-20 bg-white/85 dark:bg-gray-800/85 backdrop-blur-xl
             border-b border-gray-200 dark:border-gray-700"
      style="padding-top: env(safe-area-inset-top)"
    >
      <div class="max-w-3xl mx-auto flex items-center gap-2 px-3 py-2.5">
        <button
          @click="goBack"
          class="p-2 -ml-1 rounded-lg text-gray-600 dark:text-gray-300
                 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          aria-label="Indietro"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <!-- Search input -->
        <div class="flex-1 relative">
          <span class="absolute inset-y-0 left-3 flex items-center text-gray-400 pointer-events-none">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M21 21l-4.35-4.35M10.5 18a7.5 7.5 0 100-15 7.5 7.5 0 000 15z" />
            </svg>
          </span>
          <input
            ref="inputEl"
            v-model="q"
            type="search"
            inputmode="search"
            enterkeyhint="search"
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            placeholder="Cerca spese, bollette, progetti, servizi"
            class="w-full pl-9 pr-9 py-3 rounded-xl text-base
                   bg-gray-100 dark:bg-gray-700/60 text-gray-900 dark:text-white
                   placeholder-gray-400 dark:placeholder-gray-400
                   border border-transparent
                   focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white dark:focus:bg-gray-700"
          />
          <button
            v-if="q.length > 0"
            type="button"
            @click="clearQuery"
            class="absolute inset-y-0 right-2 flex items-center px-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
            aria-label="Cancella"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Body -->
    <div class="max-w-3xl mx-auto px-3 py-4">
      <!-- Empty prompt (nothing typed) -->
      <div v-if="!q.trim()" class="text-center py-16 text-gray-500 dark:text-gray-400">
        <svg class="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M21 21l-4.35-4.35M10.5 18a7.5 7.5 0 100-15 7.5 7.5 0 000 15z" />
        </svg>
        <p class="text-sm">Inizia a scrivere per cercare</p>
        <p class="text-xs mt-1 text-gray-400">Spese, bollette, progetti, servizi</p>
      </div>

      <!-- Loading -->
      <div v-else-if="loading && hits.length === 0" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-6 w-6 border-2 border-blue-500 border-t-transparent" />
      </div>

      <!-- No results -->
      <div
        v-else-if="!loading && hits.length === 0 && q.trim().length >= 1"
        class="text-center py-16 text-gray-500 dark:text-gray-400"
      >
        <p class="text-sm">Nessun risultato per "{{ q.trim() }}"</p>
      </div>

      <!-- Results grouped by entity type -->
      <div v-else class="space-y-5">
        <section v-for="group in groupedHits" :key="group.type">
          <h2 class="px-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
            {{ group.label }}
            <span class="ml-1 text-gray-400">({{ group.hits.length }})</span>
          </h2>
          <ul class="bg-white dark:bg-gray-800 rounded-2xl overflow-hidden divide-y divide-gray-100 dark:divide-gray-700 shadow-sm">
            <li v-for="hit in group.hits" :key="`${hit.entity_type}-${hit.entity_id}`">
              <button
                type="button"
                class="w-full text-left px-4 py-3 flex items-start gap-3 hover:bg-gray-50 dark:hover:bg-gray-700/40 active:bg-gray-100 dark:active:bg-gray-700 transition-colors"
                @click="openHit(hit)"
              >
                <div :class="['flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center', iconBgClass(hit.entity_type), iconTextClass(hit.entity_type)]">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <!-- Expense: credit card -->
                    <path v-if="hit.entity_type === 'expense'"
                      stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8"
                      d="M3 10h18M5 6h14a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2z" />
                    <!-- Bill: document -->
                    <path v-else-if="hit.entity_type === 'bill'"
                      stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8"
                      d="M9 12h6m-6 4h6m-6-8h6M7 20h10a2 2 0 002-2V6a2 2 0 00-2-2H7a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    <!-- Project: folder -->
                    <path v-else-if="hit.entity_type === 'project'"
                      stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8"
                      d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    <!-- Utility: lightning -->
                    <path v-else-if="hit.entity_type === 'utility'"
                      stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8"
                      d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-white truncate">
                    {{ hit.title || '(senza titolo)' }}
                  </p>
                  <p
                    v-if="hit.snippet"
                    class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-2"
                    v-html="renderSnippet(hit.snippet)"
                  />
                </div>
                <svg class="flex-shrink-0 w-4 h-4 text-gray-300 dark:text-gray-600 mt-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            </li>
          </ul>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { searchAPI } from '@/api/client'

const router = useRouter()
const q = ref('')
const hits = ref([])
const loading = ref(false)
const inputEl = ref(null)

let debounceId = null
let activeController = null

const GROUP_LABELS = {
  expense: 'Spese',
  bill: 'Bollette',
  project: 'Progetti',
  utility: 'Servizi',
}

const ICON_BG = {
  expense: 'bg-emerald-100 dark:bg-emerald-900/30',
  bill: 'bg-amber-100 dark:bg-amber-900/30',
  project: 'bg-violet-100 dark:bg-violet-900/30',
  utility: 'bg-blue-100 dark:bg-blue-900/30',
}

const ICON_TEXT = {
  expense: 'text-emerald-600 dark:text-emerald-300',
  bill: 'text-amber-600 dark:text-amber-300',
  project: 'text-violet-600 dark:text-violet-300',
  utility: 'text-blue-600 dark:text-blue-300',
}

const groupedHits = computed(() => {
  const buckets = {}
  for (const h of hits.value) {
    if (!buckets[h.entity_type]) buckets[h.entity_type] = []
    buckets[h.entity_type].push(h)
  }
  // preserve a stable group ordering
  const order = ['expense', 'bill', 'utility', 'project']
  return order
    .filter((k) => buckets[k]?.length)
    .map((k) => ({ type: k, label: GROUP_LABELS[k] || k, hits: buckets[k] }))
})

function iconBgClass(type) {
  return ICON_BG[type] || 'bg-gray-100 dark:bg-gray-700'
}

function iconTextClass(type) {
  return ICON_TEXT[type] || 'text-gray-500 dark:text-gray-300'
}

function escapeHtml(s) {
  return (s || '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  })[c])
}

const SNIPPET_OPEN = String.fromCharCode(1)
const SNIPPET_CLOSE = String.fromCharCode(2)

function renderSnippet(s) {
  return escapeHtml(s)
    .split(SNIPPET_OPEN).join('<mark class="bg-yellow-200 dark:bg-yellow-500/40 text-inherit rounded px-0.5">')
    .split(SNIPPET_CLOSE).join('</mark>')
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/')
}

function clearQuery() {
  q.value = ''
  hits.value = []
  nextTick(() => inputEl.value?.focus())
}

async function runSearch(term) {
  if (activeController) activeController.abort()
  const controller = new AbortController()
  activeController = controller

  loading.value = true
  try {
    const { data } = await searchAPI.query(term, controller.signal)
    if (!controller.signal.aborted) {
      hits.value = data?.hits || []
    }
  } catch (err) {
    if (err.name !== 'CanceledError' && err.name !== 'AbortError') {
      hits.value = []
    }
  } finally {
    if (activeController === controller) {
      loading.value = false
      activeController = null
    }
  }
}

watch(q, (val) => {
  const term = val.trim()
  clearTimeout(debounceId)
  if (activeController) {
    activeController.abort()
    activeController = null
  }
  if (!term) {
    hits.value = []
    loading.value = false
    return
  }
  loading.value = true
  debounceId = setTimeout(() => runSearch(term), 250)
})

function openHit(hit) {
  // Uniform pattern: land on the list where the entity appears and flash the
  // row. Each destination reads `?highlight=<id>` via useHighlight. For
  // property-scoped views (utility, project) we also forward `property` so the
  // destination auto-switches property if the entity lives under a different
  // one than the currently selected.
  switch (hit.entity_type) {
    case 'expense':
      router.push({ path: '/expenses', query: { highlight: hit.entity_id } })
      break
    case 'bill':
      // parent_id is the utility the bill belongs to. The utility detail page
      // defaults to the Bollette tab, which is where the bill appears.
      if (hit.parent_id) {
        router.push({
          path: `/utilities/${hit.parent_id}`,
          query: { tab: 'bills', highlight: hit.entity_id },
        })
      } else {
        router.push('/utilities')
      }
      break
    case 'project': {
      const query = { highlight: hit.entity_id }
      if (hit.property_id) query.property = hit.property_id
      router.push({ path: '/projects', query })
      break
    }
    case 'utility': {
      const query = { highlight: hit.entity_id }
      if (hit.property_id) query.property = hit.property_id
      router.push({ path: '/utilities', query })
      break
    }
    default:
      router.push('/')
  }
}

onMounted(() => {
  nextTick(() => inputEl.value?.focus())
})
</script>

<style scoped>
.pb-safe {
  padding-bottom: calc(env(safe-area-inset-bottom) + 80px);
}
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
