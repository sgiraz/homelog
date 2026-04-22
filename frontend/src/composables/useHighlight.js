import { ref, nextTick, watch, onScopeDispose } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export function useHighlight({ source, getId = (x) => x.id } = {}) {
  const route = useRoute()
  const router = useRouter()

  const highlightId = ref(parseTarget(route.query.highlight))
  const rowEls = new Map()
  let clearTimer = null
  // prevents re-triggering on unrelated source mutations; reset after flash so the same id can be deep-linked again
  let handledId = null

  function parseTarget(raw) {
    if (raw == null || raw === '') return null
    const n = Number(raw)
    return Number.isFinite(n) ? n : null
  }

  function registerRow(id, el) {
    if (el) rowEls.set(id, el)
    else rowEls.delete(id)
  }

  function isHighlighted(id) {
    return highlightId.value != null && Number(id) === highlightId.value
  }

  async function trigger(id) {
    highlightId.value = id
    handledId = id
    await nextTick()
    const el = rowEls.get(id)
    if (el && typeof el.scrollIntoView === 'function') {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
    if (clearTimer) clearTimeout(clearTimer)
    clearTimer = setTimeout(() => {
      // a newer deep-link may have taken over; only clean up our own id
      if (highlightId.value !== id) return
      highlightId.value = null
      handledId = null
      const q = { ...route.query }
      delete q.highlight
      router.replace({ path: route.path, query: q, hash: route.hash })
    }, 2300)
  }

  // keep-alive (App.vue) suppresses remount on query changes; watch syncs highlightId across navigations
  watch(
    () => parseTarget(route.query.highlight),
    (id) => {
      if (id !== highlightId.value) highlightId.value = id
    },
  )

  if (typeof source === 'function') {
    watch(
      [() => highlightId.value, source],
      ([id, items]) => {
        if (id == null || id === handledId) return
        if (!items || items.length === 0) return
        const hit = items.some((it) => Number(getId(it)) === id)
        if (hit) trigger(id)
      },
      { immediate: true },
    )
  }

  onScopeDispose(() => {
    if (clearTimer) clearTimeout(clearTimer)
  })

  return { highlightId, isHighlighted, registerRow, trigger }
}
