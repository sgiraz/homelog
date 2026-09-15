import { watch } from 'vue'

// Cookieless, privacy-first usage stats for the public demo instance only.
// Self-hosted installs never load this: GoatCounter is wired up exclusively
// when isDemoMode (from /version) is true, and it sets no cookies and stores
// no personal data — see docs-site privacy policy (demo section).
//
// TODO: replace with your GoatCounter site code (sign up free for open
// source projects at https://www.goatcounter.com/) before deploying.
const GOATCOUNTER_SITE = 'homelog-demo'

let scriptRequested = false

function loadScript(onReady) {
  if (scriptRequested) return
  scriptRequested = true
  const script = document.createElement('script')
  script.async = true
  script.src = 'https://gc.zgo.at/count.js'
  script.dataset.goatcounter = `https://${GOATCOUNTER_SITE}.goatcounter.com/count`
  // We call count() ourselves on every route change (incl. the first), so
  // disable GoatCounter's own onload pageview to avoid double-counting.
  script.dataset.goatcounterSettings = JSON.stringify({ no_onload: true })
  script.onload = onReady
  document.head.appendChild(script)
}

function countPageview(path) {
  window.goatcounter?.count({ path })
}

// Raw browser/OS language preference, e.g. "fr" — deliberately NOT run
// through detectBrowserLocale()'s supported-locale fallback: unsupported
// languages are exactly the signal we want (demand for a language HomeLog
// doesn't have yet), not the "it"/"en" the app fell back to for them.
function browserLanguage() {
  const tag = navigator.languages?.[0] || navigator.language || ''
  return tag.toLowerCase().split('-')[0] || 'unknown'
}

// Call once at startup (App.vue), after the router is available. No-ops
// until/unless isDemoMode becomes true.
export function initAnalytics(router, isDemoMode) {
  const stop = watch(isDemoMode, (enabled) => {
    if (!enabled) return
    loadScript(() => {
      countPageview(router.currentRoute.value.fullPath)
      trackEvent(`lang_${browserLanguage()}`)
    })
    router.afterEach((to) => countPageview(to.fullPath))
    stop()
  }, { immediate: true })
}

// Tracks a named feature-usage event (e.g. "expense_created"). Safe to call
// unconditionally from stores — it's a no-op outside demo mode or before the
// script has loaded.
export function trackEvent(name) {
  window.goatcounter?.count({ path: name, title: name, event: true })
}
