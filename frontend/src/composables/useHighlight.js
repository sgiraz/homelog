import { ref, nextTick, watch, onScopeDispose } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// useHighlight wires up the `?highlight=<id>` query-param pattern used by
// global search results. Views that list rows call it to:
//   1. Read a target id from the URL.
//   2. Scroll that row into view once it's rendered (observes `source`).
//   3. Keep it flash-highlighted for ~2s, then strip the query param so a
//      reload does not re-highlight.
//
// Usage inside a list view:
//
//   const { highlightId, isHighlighted, registerRow } = useHighlight({
//     source: () => store.items,
//     getId: (item) => item.id,
//   })
//   <div :ref="(el) => registerRow(item.id, el)"
//        :class="{ 'search-flash': isHighlighted(item.id) }"> ... </div>
//
// Add the CSS once in the view's scoped style block:
//
//   .search-flash { animation: search-flash 2.2s ease-out; }
//   @keyframes search-flash {
//     0%   { box-shadow: 0 0 0 3px rgba(59,130,246,.5); background: rgba(59,130,246,.12); }
//     70%  { box-shadow: 0 0 0 3px rgba(59,130,246,.3); background: rgba(59,130,246,.04); }
//     100% { box-shadow: 0 0 0 0 transparent; background: transparent; }
//   }
export function useHighlight({ source, getId = (x) => x.id } = {}) {
  const route = useRoute()
  const router = useRouter()

  const highlightId = ref(parseTarget(route.query.highlight))
  const rowEls = new Map()
  let clearTimer = null
  // Tracks the id we've already scrolled/flashed so the source watcher
  // doesn't re-fire on every list mutation. Reset when the flash finishes
  // so the same row can be deep-linked again later.
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
      // Another deep-link may have taken over while we were flashing; only
      // clean up if we're still showing the id this timer was scheduled for.
      if (highlightId.value !== id) return
      highlightId.value = null
      handledId = null
      // Drop the query param so a reload does not replay the highlight.
      const q = { ...route.query }
      delete q.highlight
      router.replace({ path: route.path, query: q, hash: route.hash })
    }, 2300)
  }

  // Keep highlightId in sync with the URL. Views cached behind <keep-alive>
  // (App.vue) don't remount when the query changes, so the composable setup
  // only runs once — without this watcher a second search-result click would
  // leave highlightId stuck on the previous target.
  watch(
    () => parseTarget(route.query.highlight),
    (id) => {
      if (id !== highlightId.value) highlightId.value = id
    },
  )

  // Scroll/flash whenever we have a target AND the source contains a matching
  // row. Both inputs can change independently: the URL changes (new deep
  // link) and the list fills in (initial async fetch, or caller appending via
  // ensureExpense). The `handledId === id` guard prevents re-triggering on
  // unrelated source mutations (filter change, delete, pagination).
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
