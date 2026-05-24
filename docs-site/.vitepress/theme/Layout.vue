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

<!--
  Unscoped on purpose: the banner reserves layout space through VitePress's
  --vp-layout-top-height variable, which the fixed nav, sidebar and content all
  read to offset themselves. Without it the nav sits at top:0 over the banner.
-->
<style>
:root {
  --vp-layout-top-height: 36px;
}
.wip-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: var(--vp-z-index-layout-top);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4em;
  height: var(--vp-layout-top-height);
  padding: 0 16px;
  font-size: 13px;
  line-height: 1.25;
  text-align: center;
  color: #fbf6ec;
  background: linear-gradient(90deg, #d9531e, #b23f12);
}
.wip-banner a {
  color: #fbf6ec;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
  white-space: nowrap;
}
/* On phones the message wraps to two lines — give it the room it needs. */
@media (max-width: 768px) {
  :root {
    --vp-layout-top-height: 56px;
  }
  .wip-banner {
    font-size: 12px;
  }
}
</style>
