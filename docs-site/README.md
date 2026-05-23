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

## Deploy on GitHub Pages

The docs are published to GitHub Pages (Render's free plan caps custom domains at
2, used by the demo + landing). Deployment is automatic via
[`.github/workflows/deploy-docs.yml`](../.github/workflows/deploy-docs.yml),
which builds this site and publishes it on every push to `main` that touches
`docs-site/**`.

One-time setup:

1. GitHub repo → **Settings → Pages → Source: GitHub Actions**.
2. The custom domain `docs.homelog.dev` is set via `public/CNAME` (committed).
3. On Cloudflare, add a `CNAME` record: `docs` → `sgiraz.github.io`.

Because the site is served from a custom subdomain, `base` stays `/`. The same
build also works on Render, Cloudflare Pages or Netlify if ever needed.

## Structure

- `index.md` — home page (hero + feature cards).
- `guide/*.md` — the documentation pages.
- `.vitepress/config.mjs` — nav, sidebar, theme config.
- `.vitepress/theme/` — ember accent + the work-in-progress banner.
