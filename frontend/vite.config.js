import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  define: {
    __APP_VERSION__: JSON.stringify(process.env.APP_VERSION || 'dev'),
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 5173,
    sourcemapIgnoreList: false,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true
      },
      '/avatars': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true
      },
      '/uploads': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    target: 'esnext',
    // Vite 8 defaults to Oxc for minification; esbuild is optional and would
    // require installing it as an explicit dep.
    sourcemap: true,
    rollupOptions: {
      output: {
        // Vite 8 uses Rolldown, which only accepts the function form of
        // manualChunks — the object shorthand from Vite 7 is no longer valid.
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (/node_modules[\\/](vue|vue-router|pinia|@vue)[\\/]/.test(id)) {
            return 'vue-vendor'
          }
          if (/node_modules[\\/](chart\.js|vue-chartjs)[\\/]/.test(id)) {
            return 'chart-vendor'
          }
        },
      },
    }
  }
})
