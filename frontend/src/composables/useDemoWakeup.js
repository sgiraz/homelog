import { ref } from 'vue'

// The public demo runs on a free host that sleeps after a few idle minutes and
// takes up to a minute to wake. A returning visitor gets the app shell from the
// service worker instantly, while every API call hangs until the server is up,
// so the page renders half-empty. This module spots that case, tells the
// visitor to wait, and reloads once the server answers.
//
// "Is this the demo?" normally comes from /version, which is exactly the call
// that hangs on a cold start, so the answer from the previous visit is kept in
// localStorage. A first visit needs no help: no service worker is installed
// yet, and the host holds the page itself until the server is up.

const DEMO_FLAG_KEY = 'demoInstance'
const HEALTH_GRACE_MS = 1500
const POLL_INTERVAL_MS = 2000

export const SLOW_REQUEST_MS = 3000

const isWaking = ref(false)
let probing = false

function isKnownDemo() {
  try {
    return localStorage.getItem(DEMO_FLAG_KEY) === '1'
  } catch {
    return false
  }
}

// Records whether this origin is a demo instance, from the /version answer.
export function rememberDemoInstance(isDemo) {
  try {
    if (isDemo) localStorage.setItem(DEMO_FLAG_KEY, '1')
    else localStorage.removeItem(DEMO_FLAG_KEY)
  } catch {
    // Storage blocked: the notice just won't show.
  }
}

async function serverHealthy() {
  try {
    const res = await fetch('/health', { cache: 'no-store' })
    return res.ok
  } catch {
    return false
  }
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

// Called by the API client when a request is slow or fails with a gateway or
// network error. A slow request alone proves nothing (a PDF upload can take a
// while), so /health decides: an awake server answers it at once and nothing is
// shown; a sleeping one holds it, the notice appears, and the page reloads once
// the server is up so every view fetches its data again.
export function suspectColdStart() {
  if (probing || !isKnownDemo()) return
  probing = true
  const grace = setTimeout(() => { isWaking.value = true }, HEALTH_GRACE_MS)
  ;(async () => {
    while (!(await serverHealthy())) {
      isWaking.value = true
      await sleep(POLL_INTERVAL_MS)
    }
    clearTimeout(grace)
    if (isWaking.value) window.location.reload()
    else probing = false
  })()
}

export function useDemoWakeup() {
  return { isWaking }
}
