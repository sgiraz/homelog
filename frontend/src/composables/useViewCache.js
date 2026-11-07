import { ref, nextTick } from 'vue'

// Main views App.vue keeps alive between navigations (component names).
const CACHED_VIEWS = ['DashboardView', 'ExpensesView', 'UtilitiesView', 'ProjectsView', 'SettingsView']

const cachedViews = ref([...CACHED_VIEWS])

// Drops every cached view except `keep`, so each one mounts fresh on its next
// visit. Needed when state those views captured at mount goes stale, e.g. the
// current property is deleted or replaced. The view on screen must be in
// `keep`: pruning the active entry would unmount it on leave instead of caching.
function evictCachedViews(keep = []) {
  cachedViews.value = CACHED_VIEWS.filter((name) => keep.includes(name))
  // keep-alive prunes in a post-flush watcher; restore once that has run.
  nextTick(() => {
    cachedViews.value = [...CACHED_VIEWS]
  })
}

export function useViewCache() {
  return { cachedViews, evictCachedViews }
}
