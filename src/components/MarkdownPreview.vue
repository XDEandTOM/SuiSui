<script setup lang="ts">
import { ref, computed } from "vue"
import { marked } from "marked"
import hljs from "highlight.js"
import "highlight.js/styles/github.css"

const renderer = new marked.Renderer()
renderer.code = ({ text, lang }) => {
  let highlighted: string
  try {
    if (lang && hljs.getLanguage(lang)) highlighted = hljs.highlight(text, { language: lang }).value
    else highlighted = hljs.highlightAuto(text).value
  } catch { highlighted = text }
  const langAttr = lang ? ` class="language-${lang}"` : ""
  const escaped = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;")
  return `<div class="code-block-wrapper">
    <button class="copy-btn" data-code="${escaped}"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>
    <pre><code${langAttr}>${highlighted}</code></pre>
  </div>`
}

marked.setOptions({ breaks: true, gfm: true })

const props = defineProps<{ content: string }>()
const zoomedImage = ref("")

const rendered = computed(() => {
  try {
    let html = marked(props.content, { renderer }) as string
    let carouselIdx = 0
    html = html.replace(/((?:<p><img[^>]*><\/p>\s*)+)/g, (match) => {
      const images = match.match(/<img[^>]*>/g)
      if (!images || images.length < 2) return match
      const id = "c" + carouselIdx++
      return `<div class="carousel-wrap" data-id="${id}">
        <div class="carousel-track">${images.map((img, i) => `<div class="carousel-slide" data-idx="${i}">${img}</div>`).join("")}</div>
        <button class="carousel-btn carousel-prev" data-id="${id}">‹</button>
        <button class="carousel-btn carousel-next" data-id="${id}">›</button>
        <div class="carousel-dots">${images.map((_, i) => `<span class="carousel-dot" data-id="${id}" data-idx="${i}"></span>`).join("")}</div>
      </div>`
    })
    return html
  } catch { return props.content }
})

function handleClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  const img = target.closest("img") as HTMLImageElement
  if (img) { zoomedImage.value = img.src; return }
  const btn = target.closest(".carousel-btn") as HTMLElement
  if (btn) {
    const wrap = btn.closest(".carousel-wrap") as HTMLElement
    if (!wrap) return
    const track = wrap.querySelector(".carousel-track") as HTMLElement
    const slides = track.querySelectorAll(".carousel-slide")
    const current = Math.round(track.scrollLeft / track.clientWidth)
    const isNext = btn.classList.contains("carousel-next")
    const next = isNext ? Math.min(current + 1, slides.length - 1) : Math.max(current - 1, 0)
    track.scrollTo({ left: next * track.clientWidth, behavior: "smooth" })
    wrap.querySelectorAll(".carousel-dot").forEach((d, i) => { (d as HTMLElement).style.opacity = i === next ? "1" : "0.35" })
    return
  }
  const copyBtn = target.closest(".copy-btn") as HTMLElement
  if (!copyBtn) return
  const code = copyBtn.getAttribute("data-code")
  if (!code) return
  navigator.clipboard.writeText(code).then(() => {
    copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M20 6L9 17l-5-5\"/></svg>"
    setTimeout(() => { copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
  }).catch(() => {
    copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"#ef4444\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><line x1=\"18\" y1=\"6\" x2=\"6\" y2=\"18\"/><line x1=\"6\" y1=\"6\" x2=\"18\" y2=\"18\"/></svg>"
    setTimeout(() => { copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
  })
}
</script>

<template>
  <div class="markdown-body" v-html="rendered" @click="handleClick" />
  <teleport to="body">
    <div v-if="zoomedImage" class="zoom-overlay" @click="zoomedImage = ''">
      <button class="zoom-close-btn" @click.stop="zoomedImage = ''">
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <img :src="zoomedImage" class="zoom-img" @click.stop />
    </div>
  </teleport>
</template>

<style scoped>
.markdown-body { word-break: break-word; line-height: 1.6; }
.markdown-body :deep(h1),.markdown-body :deep(h2),.markdown-body :deep(h3),
.markdown-body :deep(h4),.markdown-body :deep(h5),.markdown-body :deep(h6) { margin: .5em 0 .25em; line-height: 1.3; }
.markdown-body :deep(h1) { font-size: 1.4em; }
.markdown-body :deep(h2) { font-size: 1.25em; }
.markdown-body :deep(h3) { font-size: 1.1em; }
.markdown-body :deep(p) { margin: .3em 0; }
.markdown-body :deep(ul),.markdown-body :deep(ol) { padding-left: 1.4em; margin: .3em 0; }
.markdown-body :deep(blockquote) { border-left: 3px solid rgba(var(--v-theme-primary),.5); padding-left: .75em; margin: .4em 0; opacity: .85; }
.markdown-body :deep(code) { background: rgba(var(--v-theme-on-surface),.08); border-radius: 4px; padding: .15em .4em; font-size: .9em; }
.markdown-body :deep(pre) { background: rgba(var(--v-theme-on-surface),.05); border-radius: 8px; padding: .75em 1em; overflow-x: auto; margin: .5em 0; }
.markdown-body :deep(pre code) { background: none; padding: 0; font-size: .85em; }
.markdown-body :deep(table) { border-collapse: collapse; width: 100%; margin: .5em 0; }
.markdown-body :deep(th),.markdown-body :deep(td) { border: 1px solid rgba(var(--v-theme-on-surface),.15); padding: .4em .6em; text-align: left; }
.markdown-body :deep(img) { max-width: 100%; max-height: 220px; border-radius: 6px; cursor: zoom-in; }
.markdown-body :deep(a) { color: rgb(var(--v-theme-primary)); }
.markdown-body :deep(.code-block-wrapper) { position: relative; margin: .5em 0; }
.markdown-body :deep(pre) { margin: 0; }
.markdown-body :deep(.copy-btn) {
  position: absolute; top: 6px; right: 6px; width: 28px; height: 28px;
  padding: 0; display: flex; align-items: center; justify-content: center;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.15); border-radius: 4px;
  background: rgb(var(--v-theme-surface)); color: rgba(var(--v-theme-on-surface), 0.6);
  cursor: pointer; opacity: 0; transition: opacity 0.2s;
}
.markdown-body :deep(.code-block-wrapper:hover .copy-btn) { opacity: 1; }
.markdown-body :deep(.copy-btn:hover) { background: rgba(var(--v-theme-primary), 0.08); color: rgb(var(--v-theme-primary)); border-color: rgba(var(--v-theme-primary), 0.3); }
.markdown-body :deep(.carousel-wrap) { position: relative; margin: .5em 0; border-radius: 8px; overflow: hidden; background: rgba(var(--v-theme-on-surface), 0.03); }
.markdown-body :deep(.carousel-track) { display: flex; overflow-x: auto; scroll-snap-type: x mandatory; scrollbar-width: none; }
.markdown-body :deep(.carousel-track::-webkit-scrollbar) { display: none; }
.markdown-body :deep(.carousel-slide) { flex: 0 0 100%; scroll-snap-align: start; display: flex; align-items: center; justify-content: center; }
.markdown-body :deep(.carousel-slide img) { max-height: 220px; width: 100%; object-fit: contain; cursor: zoom-in; }
.markdown-body :deep(.carousel-btn) {
  position: absolute; top: 50%; transform: translateY(-50%); z-index: 2;
  width: 32px; height: 32px; border-radius: 50%; border: none;
  background: rgba(0,0,0,0.45); color: #fff; font-size: 1.3rem;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; opacity: 0; transition: opacity 0.2s; line-height: 1;
}
.markdown-body :deep(.carousel-wrap:hover .carousel-btn) { opacity: 1; }
.markdown-body :deep(.carousel-btn:hover) { background: rgba(0,0,0,0.65); }
.markdown-body :deep(.carousel-prev) { left: 8px; }
.markdown-body :deep(.carousel-next) { right: 8px; }
.markdown-body :deep(.carousel-dots) { display: flex; gap: 6px; justify-content: center; padding: 8px; }
.markdown-body :deep(.carousel-dot) { width: 8px; height: 8px; border-radius: 50%; background: rgb(var(--v-theme-primary)); opacity: 0.35; cursor: pointer; transition: opacity 0.2s; }
</style>

<style>
.zoom-overlay {
  position: fixed; inset: 0; z-index: 9999;
  background: rgba(0,0,0,0.8);
  display: flex; align-items: center; justify-content: center;
  cursor: zoom-out;
}
.zoom-img { max-width: 90vw; max-height: 90vh; border-radius: 8px; object-fit: contain; cursor: default; }
.zoom-close-btn {
  position: fixed; top: 16px; right: 16px; width: 36px; height: 36px; border-radius: 50%;
  border: none; background: rgba(255,255,255,0.15); color: #fff;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; transition: background 0.2s; z-index: 10000;
}
.zoom-close-btn:hover { background: rgba(255,255,255,0.3); }
</style>
