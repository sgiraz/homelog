// ─────────────────────────────────────────────────────────────────────────
// Color-theme registry — the single source of truth for the theme LIST.
//
// Adding or removing a theme touches exactly two places: its palette block in
// `src/assets/styles/main.css` ([data-theme="id"], or :root/.dark for the
// default) and one entry here. The picker, the settings store, validation and
// the anti-flash snippet in index.html all derive from this list.
//
// `swatch` is [light canvas, dark canvas, accent] hexes for the picker preview
// — showing both backgrounds makes warm vs cool obvious at a glance. Changing
// the DEFAULT theme's swatch also means updating the theme-color metas in
// index.html, which hardcode it.
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
