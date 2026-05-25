import { defineConfig } from '@vite-pwa/assets-generator/config'

// Generates ONLY the transparent pwa-*.png icons + favicon.ico from the rounded
// adaptive logo.svg (padding 0 = full-bleed rounded tile, transparent corners).
//
// maskable + apple-touch are intentionally DISABLED here (sizes: []) — they are
// rendered by `scripts/build-logo.mjs` from a clean full-bleed square so iOS 26
// can apply its own Liquid Glass on a neutral canvas. Re-enabling them here would
// clobber those with sheen/rim baked-in versions. Run build-logo.mjs first.
export default defineConfig({
  preset: {
    transparent: {
      sizes: [64, 192, 512],
      favicons: [[48, 'favicon.ico']],
      padding: 0,
    },
    maskable: { sizes: [] },
    apple: { sizes: [] },
  },
  images: ['public/logo.svg'],
})
