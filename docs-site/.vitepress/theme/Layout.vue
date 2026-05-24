<script setup>
import DefaultTheme from 'vitepress/theme'
import { useData } from 'vitepress'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const { Layout } = DefaultTheme
const { lang } = useData()
const isIt = computed(() => (lang.value || '').toLowerCase().startsWith('it'))

// The banner is position:fixed (out of flow). VitePress offsets the nav, sidebar
// and content by --vp-layout-top-height, so we keep that variable in sync with
// the banner's *actual* rendered height. This way a message that wraps to two or
// three lines (narrow phones, longer locales) never overlaps the nav below it.
const banner = ref(null)
let ro = null
function syncHeight() {
  if (!banner.value) return
  const h = Math.ceil(banner.value.getBoundingClientRect().height)
  document.documentElement.style.setProperty('--vp-layout-top-height', h + 'px')
}
onMounted(() => {
  syncHeight()
  if (typeof ResizeObserver !== 'undefined') {
    ro = new ResizeObserver(syncHeight)
    ro.observe(banner.value)
  } else {
    window.addEventListener('resize', syncHeight)
  }
})
// Switching language changes the text length — remeasure after the DOM updates.
watch(lang, () => requestAnimationFrame(syncHeight))
onUnmounted(() => {
  if (ro) ro.disconnect()
  else window.removeEventListener('resize', syncHeight)
})
</script>

<template>
  <Layout>
    <template #layout-top>
      <div ref="banner" class="wip-banner">
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
  read to offset themselves. The exact value is set from JS (see syncHeight); the
  CSS defaults below just avoid a flash before hydration.
-->
<style>
:root {
  --vp-layout-top-height: 40px;
}
.wip-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: var(--vp-z-index-layout-top);
  padding: 7px 16px;
  font-size: 13px;
  line-height: 1.3;
  text-align: center;
  color: #fbf6ec;
  background: linear-gradient(90deg, #d9531e, #b23f12);
}
.wip-banner strong {
  font-weight: 700;
}
.wip-banner a {
  color: #fbf6ec;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
  white-space: nowrap;
}
/* Pre-hydration fallback for the taller wrapped message on phones. */
@media (max-width: 768px) {
  :root {
    --vp-layout-top-height: 60px;
  }
  .wip-banner {
    font-size: 12px;
    padding: 6px 12px;
  }
}
</style>
