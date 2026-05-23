import DefaultTheme from 'vitepress/theme'
import Layout from './Layout.vue'
import './custom.css'

// Extend the default theme with a persistent "work in progress" banner.
export default {
  extends: DefaultTheme,
  Layout,
}
