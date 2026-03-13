import { ref, watch } from 'vue'

/**
 * Unified theme system — single source of truth for theme mode.
 * Supports 'auto' (follows OS), 'light', 'dark'.
 * All callers share the same reactive state (module-level refs).
 */

// The user's chosen mode: 'auto' | 'light' | 'dark'
const themeMode = ref('auto')

// Whether the page is actually in dark mode right now (derived)
const isDark = ref(false)

// OS preference media query
let mediaQuery = null

function applyTheme() {
  const shouldBeDark = themeMode.value === 'dark'
    || (themeMode.value === 'auto' && (mediaQuery?.matches ?? false))

  isDark.value = shouldBeDark
  if (shouldBeDark) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

// Initialize media query listener (runs once at module import)
if (typeof window !== 'undefined') {
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (themeMode.value === 'auto') {
      applyTheme()
    }
  })

  // Read initial value from localStorage (before settings load from API)
  const stored = localStorage.getItem('themeMode')
  if (stored === 'dark' || stored === 'light' || stored === 'auto') {
    themeMode.value = stored
  } else {
    // Migrate from old 'theme' key if present
    const oldStored = localStorage.getItem('theme')
    if (oldStored === 'dark' || oldStored === 'light') {
      themeMode.value = oldStored
    }
    // else stays 'auto'
  }

  applyTheme()
}

// Re-apply whenever themeMode changes
watch(themeMode, () => {
  applyTheme()
  localStorage.setItem('themeMode', themeMode.value)
})

export function useDarkMode() {
  /**
   * Set theme mode explicitly. Called from settings or on settings load.
   * @param {'auto'|'light'|'dark'} mode
   */
  function setTheme(mode) {
    if (mode === 'auto' || mode === 'light' || mode === 'dark') {
      themeMode.value = mode
    }
  }

  /**
   * Toggle between light and dark (used by Navbar button).
   * Switches to explicit light/dark, breaking out of 'auto'.
   */
  function toggleDarkMode() {
    themeMode.value = isDark.value ? 'light' : 'dark'
  }

  return { isDark, themeMode, setTheme, toggleDarkMode }
}
