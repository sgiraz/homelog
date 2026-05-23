import { defineConfig } from 'vitepress'

// HomeLog user documentation — docs.homelog.dev.
// Served from a custom subdomain, so the site lives at the root (base: '/').
// Bilingual: English at /, Italian at /it/.
export default defineConfig({
  title: 'HomeLog Docs',
  description: 'How to use HomeLog — self-hosted home expense & utilities manager for families.',
  lastUpdated: true,
  // Local install URLs aren't reachable at build time — don't fail on them.
  ignoreDeadLinks: [/^https?:\/\/localhost/],
  head: [
    ['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }],
    ['meta', { name: 'theme-color', content: '#D9531E' }],
    // First-visit default to the browser language: only on the English home,
    // once per session (so manually switching back to English doesn't loop).
    ['script', {}, `try{
      var p = location.pathname;
      if ((p === '/' || p === '/index.html') &&
          !sessionStorage.getItem('hl-lang-redirect') &&
          (navigator.language || '').toLowerCase().indexOf('it') === 0) {
        sessionStorage.setItem('hl-lang-redirect', '1');
        location.replace('/it/');
      }
    }catch(e){}`],
  ],
  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/sgiraz/homelog' },
    ],
    search: { provider: 'local' },
  },
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/getting-started', activeMatch: '/guide/' },
          { text: 'Live Demo', link: 'https://demo.homelog.dev' },
          { text: 'Website', link: 'https://homelog.dev' },
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Introduction', link: '/guide/getting-started' },
                { text: 'Self-Hosting', link: '/guide/self-hosting' },
              ],
            },
            {
              text: 'Using HomeLog',
              items: [
                { text: 'Expenses & Splitting', link: '/guide/expenses' },
                { text: 'Utilities & Bills', link: '/guide/utilities' },
                { text: 'PDF Bill Templates', link: '/guide/pdf-templates' },
                { text: 'Projects', link: '/guide/projects' },
              ],
            },
          ],
        },
        editLink: {
          pattern: 'https://github.com/sgiraz/homelog/edit/main/docs-site/:path',
          text: 'Edit this page on GitHub',
        },
        footer: {
          message: '🚧 Work in progress · Released under the AGPL-3.0 License.',
          copyright: 'HomeLog — self-hosted home management.',
        },
      },
    },
    it: {
      label: 'Italiano',
      lang: 'it-IT',
      link: '/it/',
      themeConfig: {
        nav: [
          { text: 'Guida', link: '/it/guide/getting-started', activeMatch: '/it/guide/' },
          { text: 'Demo live', link: 'https://demo.homelog.dev' },
          { text: 'Sito', link: 'https://homelog.dev' },
        ],
        sidebar: {
          '/it/guide/': [
            {
              text: 'Per iniziare',
              items: [
                { text: 'Introduzione', link: '/it/guide/getting-started' },
                { text: 'Self-hosting', link: '/it/guide/self-hosting' },
              ],
            },
            {
              text: 'Usare HomeLog',
              items: [
                { text: 'Spese e divisione', link: '/it/guide/expenses' },
                { text: 'Utenze e bollette', link: '/it/guide/utilities' },
                { text: 'Template PDF', link: '/it/guide/pdf-templates' },
                { text: 'Progetti', link: '/it/guide/projects' },
              ],
            },
          ],
        },
        editLink: {
          pattern: 'https://github.com/sgiraz/homelog/edit/main/docs-site/:path',
          text: 'Modifica questa pagina su GitHub',
        },
        footer: {
          message: '🚧 Lavori in corso · Rilasciato sotto licenza AGPL-3.0.',
          copyright: 'HomeLog — gestione domestica self-hosted.',
        },
        docFooter: { prev: 'Precedente', next: 'Successivo' },
        outline: { label: 'In questa pagina' },
        lastUpdated: { text: 'Ultimo aggiornamento' },
        langMenuLabel: 'Cambia lingua',
        returnToTopLabel: 'Torna su',
        sidebarMenuLabel: 'Menu',
        darkModeSwitchLabel: 'Tema',
      },
    },
  },
})
