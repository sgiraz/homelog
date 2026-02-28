import { ref } from 'vue'

// Module-level shared state — all useDarkMode() callers share the same ref
const isDark = ref(false)

// Initialize from localStorage / system preference (runs once at module import)
if (typeof window !== 'undefined') {
  const stored = localStorage.getItem('theme')
  if (stored === 'dark') {
    isDark.value = true
    document.documentElement.classList.add('dark')
  } else if (stored === 'light') {
    isDark.value = false
  } else {
    // No preference stored — follow system
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    isDark.value = prefersDark
    if (prefersDark) document.documentElement.classList.add('dark')
  }
}

export function useDarkMode() {
  function toggleDarkMode() {
    isDark.value = !isDark.value
    if (isDark.value) {
      document.documentElement.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    }
  }

  return { isDark, toggleDarkMode }
}
