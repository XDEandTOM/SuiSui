<script setup lang="ts">
import { onMounted } from "vue"

onMounted(() => {
  document.querySelectorAll(".copy-btn").forEach(btn => {
    btn.addEventListener("click", () => {
      const code = btn.getAttribute("data-code")
      if (!code) return
      const clipboard = navigator.clipboard
      function doCopy(text: string) {
        if (clipboard) return clipboard.writeText(text)
        const ta = document.createElement("textarea")
        ta.value = text; ta.style.position = "fixed"; ta.style.opacity = "0"
        document.body.appendChild(ta); ta.select()
        try { document.execCommand("copy") } catch {}
        document.body.removeChild(ta)
        return Promise.resolve()
      }
      doCopy(code).then(() => {
        btn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M20 6L9 17l-5-5\"/></svg>"
        setTimeout(() => { btn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
      }).catch(() => {
        btn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"#ef4444\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><line x1=\"18\" y1=\"6\" x2=\"6\" y2=\"18\"/><line x1=\"6\" y1=\"6\" x2=\"18\" y2=\"18\"/></svg>"
        setTimeout(() => { btn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
      })
    })
  })
})
</script>

<template>
  <slot />
</template>
