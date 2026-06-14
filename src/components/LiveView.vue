<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue"

const loading = ref(true)
let dp: any = null

onMounted(async () => {
  // Load CSS
  const link = document.createElement("link")
  link.rel = "stylesheet"
  link.href = "/DPlayer.min.css"
  document.head.appendChild(link)

  // Load local scripts
  await loadScript("/hls.min.js")
  await loadScript("/DPlayer.min.js")

  // Fetch stream URL
  let url = ""
  try {
    const r = await fetch("/api/live/config")
    const d = await r.json()
    url = d.streamUrl || ""
  } catch { console.warn("failed to load live config") }

  loading.value = false
  if (!url) return

  // Init DPlayer with HLS
  const DPlayer = (window as any).DPlayer
  const Hls = (window as any).Hls
  dp = new DPlayer({
    container: document.getElementById("dplayer"),
    video: {
      url,
      type: "customHLS",
      customType: {
        customHLS(v: HTMLVideoElement) {
          const h = new Hls()
          h.loadSource(v.src)
          h.attachMedia(v)
        },
      },
    },
    autoplay: true,
    volume: 1,
    theme: "#1976D2",
  })
})

function loadScript(src: string): Promise<void> {
  return new Promise((resolve) => {
    const s = document.createElement("script")
    s.src = src
    s.onload = () => resolve()
    document.head.appendChild(s)
  })
}

onBeforeUnmount(() => {
  if (dp) { try { dp.destroy() } catch { console.warn("dp destroy failed") } }
})
</script>

<template>
  <div class="live-page">
    <div v-if="loading" class="loading-state">
      <v-progress-circular indeterminate color="primary" />
      <p class="text-body-2 text-medium-emphasis mt-2">加载直播...</p>
    </div>
    <div v-else-if="!dp" class="loading-state">
      <v-icon size="48" color="grey-darken-1">mdi-video-off-outline</v-icon>
      <p class="text-body-2 text-medium-emphasis mt-3">直播流未配置</p>
      <p class="text-caption text-medium-emphasis mt-1">请管理员在后台配置直播流地址</p>
    </div>
    <div v-else id="dplayer"></div>
  </div>
</template>

<style scoped>
.live-page { width: 100vw; height: 100vh; background: #000; overflow: hidden; }
.loading-state { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; gap: 4px; padding: 20px; }
#dplayer { width: 100%; height: 100%; }
</style>
