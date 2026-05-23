<script setup>
import DefaultTheme from 'vitepress/theme'
import { useData } from 'vitepress'
import { computed } from 'vue'

const { Layout } = DefaultTheme
const { lang } = useData()
const isIt = computed(() => (lang.value || '').toLowerCase().startsWith('it'))
</script>

<template>
  <Layout>
    <template #layout-top>
      <div class="wip-banner">
        <template v-if="isIt">
          🚧 <strong>Lavori in corso</strong> — questa documentazione è ancora in fase di scrittura.
          <a href="https://github.com/sgiraz/homelog" target="_blank" rel="noopener">Contribuisci su GitHub →</a>
        </template>
        <template v-else>
          🚧 <strong>Work in progress</strong> — these docs are still being written.
          <a href="https://github.com/sgiraz/homelog" target="_blank" rel="noopener">Help out on GitHub →</a>
        </template>
      </div>
    </template>
  </Layout>
</template>

<style scoped>
.wip-banner {
  position: sticky;
  top: 0;
  z-index: 30;
  text-align: center;
  font-size: 13px;
  padding: 6px 16px;
  color: #fff;
  background: linear-gradient(90deg, #D9531E, #B23F12);
}
.wip-banner a {
  color: #fff;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
}
</style>
