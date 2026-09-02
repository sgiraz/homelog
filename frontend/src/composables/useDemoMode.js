import { ref } from 'vue'
import { versionAPI, demoAPI } from '@/api/client'

// Module-level state — fetched once and shared across every component.
const isDemoMode = ref(false)
const isResetting = ref(false)
// Instance/version info, also sourced from the single /version call.
const appVersion = ref('')
const updateAvailable = ref(false)
const latestUrl = ref('')
// Memoized so concurrent callers (App.vue at startup, the settings store when
// it needs to know whether the account is shared) all await the same request.
let initPromise = null

// Credentials for the single shared demo account. Mirrors the backend
// constants in database/demo.go; used to prefill the login form.
export const DEMO_EMAIL = 'demo@homelog.app'
export const DEMO_PASSWORD = 'demo'

export function useDemoMode() {
  // Reads the public /version endpoint to learn whether this instance is a
  // demo. Safe to call before authentication. Runs at most once per page load.
  function initDemoMode() {
    if (initPromise) return initPromise
    initPromise = (async () => {
      try {
        const { data } = await versionAPI.check()
        isDemoMode.value = !!data.demo_mode
        appVersion.value = data.current && data.current !== 'dev' ? data.current : ''
        updateAvailable.value = !!data.update_available
        latestUrl.value = data.latest_url || ''
      } catch {
        // Version check is best-effort; default to non-demo on failure.
        isDemoMode.value = false
      }
    })()
    return initPromise
  }

  // Resets the dataset server-side, then reloads so every view re-fetches the
  // fresh seed. The demo account keeps ID 1, so the session stays valid.
  async function resetDemo() {
    if (isResetting.value) return
    isResetting.value = true
    try {
      await demoAPI.reset()
      window.location.reload()
    } catch {
      isResetting.value = false
      window.$toast?.error('Reset failed')
    }
  }

  return { isDemoMode, isResetting, appVersion, updateAvailable, latestUrl, initDemoMode, resetDemo }
}
