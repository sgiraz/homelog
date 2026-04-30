import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import globals from 'globals'

export default [
  // Global ignores
  { ignores: ['dist/**', 'node_modules/**', 'dev-dist/**'] },

  // Base JS recommended rules
  js.configs.recommended,

  // Vue 3 essential rules (flat config)
  ...pluginVue.configs['flat/essential'],

  // Browser globals for source files
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        __APP_VERSION__: 'readonly',
      },
    },
    rules: {
      'no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'vue/multi-word-component-names': 'off',
    },
  },

  // Node globals for config files and CLI scripts
  {
    files: ['vite.config.js', 'tailwind.config.js', 'postcss.config.js', 'scripts/**/*.{js,mjs}'],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
]
