// ─────────────────────────────────────────────────────────────────────────
// Color-theme registry — the single source of truth for the theme LIST.
//
// To add or remove a theme you touch exactly TWO places:
//   1) its palette block in `src/assets/styles/main.css`  ([data-theme="id"])
//   2) one entry here (id + 3 preview-swatch colors)
//
// Everything else — the picker, the settings store, validation, the default,
// and the anti-flash snippet in index.html — derives from this list, so no
// other file needs editing.
//
// `swatch` is [light canvas, dark canvas, accent] hexes for the picker
// preview — showing both backgrounds makes warm vs cool themes obvious at a
// glance. The real palettes live in main.css.
// ─────────────────────────────────────────────────────────────────────────
export const THEMES = [
  { id: 'paper',  swatch: ['#FBF6EC', '#1A1612', '#D9531E'] },
  { id: 'slate',  swatch: ['#EAEDF3', '#0F131A', '#D9531E'] },
  { id: 'forest', swatch: ['#F2F4EE', '#141813', '#2E6B4E'] },
  { id: 'ocean',  swatch: ['#F0F5F7', '#0D1620', '#0E7C86'] },
  { id: 'plum',   swatch: ['#F8F1F0', '#1A1318', '#9A3B63'] },
]

export const THEME_IDS = THEMES.map(t => t.id)
export const DEFAULT_THEME = 'slate'

export function isValidTheme(id) {
  return THEME_IDS.includes(id)
}
