import { ref, watch } from 'vue'
import { THEME_IDS, DEFAULT_THEME } from '@/config/themes'

/**
 * Color-theme system (palette), orthogonal to light/dark mode.
 * Applies the chosen theme as `data-theme` on <html>; the CSS variable blocks
 * in main.css ([data-theme="…"] / .dark[data-theme="…"]) do the rest, so the
 * whole UI re-colours from one attribute. Shared module-level state.
 */

const colorTheme = ref(DEFAULT_THEME)

function applyColorTheme() {
  if (typeof document !== 'undefined') {
    document.documentElement.dataset.theme = colorTheme.value
  }
}

// Read the persisted theme before the API responds (matches the pre-paint
// snippet in index.html so there's no flash).
if (typeof window !== 'undefined') {
  const stored = localStorage.getItem('colorTheme')
  if (stored && THEME_IDS.includes(stored)) colorTheme.value = stored
  applyColorTheme()
}

watch(colorTheme, () => {
  applyColorTheme()
  try { localStorage.setItem('colorTheme', colorTheme.value) } catch { /* storage unavailable */ }
})

export function useTheme() {
  /** Set the active palette. Invalid ids are ignored (falls back to current). */
  function setColorTheme(id) {
    if (THEME_IDS.includes(id)) colorTheme.value = id
  }
  return { colorTheme, setColorTheme }
}
