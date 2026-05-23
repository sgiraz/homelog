# Documentation — docs.homelog.dev

User documentation for HomeLog, built with [VitePress](https://vitepress.dev).
**Work in progress** (a persistent banner says so on every page).

## Local development

```bash
cd docs-site
npm install
npm run docs:dev      # local preview with hot reload
npm run docs:build    # production build → .vitepress/dist
```

## Deploy on Render (static site)

1. New → **Static Site** → connect the repo.
2. **Root Directory:** `docs-site`
3. **Build Command:** `npm install && npm run docs:build`
4. **Publish Directory:** `.vitepress/dist`
5. Add custom domain **docs.homelog.dev**.

Because the site is served from a custom subdomain, `base` stays `/` — the same
build works on GitHub Pages, Cloudflare Pages or Netlify.

## Structure

- `index.md` — home page (hero + feature cards).
- `guide/*.md` — the documentation pages.
- `.vitepress/config.mjs` — nav, sidebar, theme config.
- `.vitepress/theme/` — ember accent + the work-in-progress banner.
