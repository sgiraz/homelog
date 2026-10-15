import { watch } from 'vue'

// Cookieless usage stats for a public demo instance only. Nothing loads unless
// the server runs in demo mode and its operator set DEMO_GOATCOUNTER_SITE,
// reported by /version as analytics_site: the site code belongs to whoever runs
// the instance, so it can never be baked into the bundle. GoatCounter sets no
// cookies and stores no personal data (see the privacy policy, demo section).

let scriptRequested = false

function loadScript(site, onReady) {
  if (scriptRequested) return
  scriptRequested = true
  const script = document.createElement('script')
  script.async = true
  script.src = 'https://gc.zgo.at/count.js'
  script.dataset.goatcounter = `https://${site}.goatcounter.com/count`
  // We call count() ourselves on every route change (incl. the first), so
  // disable GoatCounter's own onload pageview to avoid double-counting.
  script.dataset.goatcounterSettings = JSON.stringify({ no_onload: true })
  script.onload = onReady
  document.head.appendChild(script)
}

function countPageview(path) {
  window.goatcounter?.count({ path })
}

// Raw browser language, e.g. "fr": deliberately not run through
// detectBrowserLocale(). Unsupported languages are the signal we want (demand),
// not the "it"/"en" the app fell back to.
function browserLanguage() {
  const tag = navigator.languages?.[0] || navigator.language || ''
  return tag.toLowerCase().split('-')[0] || 'unknown'
}

// Call once at startup (App.vue), after the router is available. No-ops
// until/unless /version names an analytics site.
export function initAnalytics(router, analyticsSite) {
  const stop = watch(analyticsSite, (site) => {
    if (!site) return
    loadScript(site, () => {
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
