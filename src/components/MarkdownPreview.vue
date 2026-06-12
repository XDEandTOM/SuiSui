<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue"
import { marked } from "marked"
import hljs from "highlight.js/lib/core"
import "highlight.js/lib/languages/javascript"
import "highlight.js/lib/languages/typescript"
import "highlight.js/lib/languages/python"
import "highlight.js/lib/languages/go"
import "highlight.js/lib/languages/bash"
import "highlight.js/lib/languages/json"
import "highlight.js/lib/languages/xml"
import "highlight.js/lib/languages/css"
import "highlight.js/lib/languages/sql"
import "highlight.js/lib/languages/yaml"
import "highlight.js/lib/languages/dockerfile"
import "highlight.js/lib/languages/markdown"
import { useTheme } from "vuetify"
import hljsDark from "highlight.js/styles/github-dark.min.css?url"
import hljsLight from "highlight.js/styles/github.min.css?url"

const theme = useTheme()
const isDark = computed(() => theme.global.name.value === "dark")

function loadHighlightTheme(dark: boolean) {
  const id = "hljs-theme"
  let link = document.getElementById(id) as HTMLLinkElement | null
  if (!link) {
    link = document.createElement("link")
    link.id = id
    link.rel = "stylesheet"
    document.head.appendChild(link)
  }
  link.href = dark ? hljsDark : hljsLight
}

onMounted(() => loadHighlightTheme(theme.global.name.value === "dark"))
watch(isDark, (v) => loadHighlightTheme(v))

const renderer = new marked.Renderer()
let todoIndex = 0
renderer.code = ({ text, lang }) => {
  let highlighted: string
  try {
    if (lang && hljs.getLanguage(lang)) highlighted = hljs.highlight(text, { language: lang }).value
    else highlighted = hljs.highlightAuto(text).value
  } catch { highlighted = text }
  const langAttr = lang ? ` class="language-${lang}"` : ""
  const encoded = encodeURIComponent(text)
  const lines = text.split("\n").length
  return `<div class="code-block-wrapper">
    <div class="code-header"><span class="code-dot red"></span><span class="code-dot yellow"></span><span class="code-dot green"></span>${lang ? `<span class="code-lang">${lang}</span>` : ""}</div>
    <div class="code-body"><div class="code-gutter">${Array.from({length: lines}, (_, i) => `<span class="code-line-num">${i + 1}</span>`).join("")}</div>
    <pre><code${langAttr}>${highlighted}</code></pre></div>
    <button class="copy-btn" data-code="${encoded}"><svg style="pointer-events:none" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>
  </div>`
}
renderer.listitem = ({ text, task, checked }) => {
  if (task) {
    const idx = todoIndex++
    const checkedAttr = checked ? ' checked' : ''
    return `<li class="todo-item"><label class="todo-label"><input type="checkbox" class="todo-checkbox" data-todo-idx="${idx}"${checkedAttr}><span class="todo-checkmark"></span><span class="todo-text${checked ? ' done' : ''}">${text}</span></label></li>`
  }
  return `<li>${text}</li>`
}

marked.setOptions({ breaks: true, gfm: true })

const emit = defineEmits<{ "todo-toggle": [idx: number] }>()
const props = defineProps<{ content: string; searchQuery?: string }>()
const zoomedImage = ref("")

function highlightText(text: string, query: string): string {
  if (!text) return text
  if (!query || !query.trim()) return text
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const re = new RegExp(`(${escaped})`, "gi")
  // Split into image URLs and text, only highlight text parts
  return text.split(/(!\[.*?\]\(.*?\))/g).map(seg => {
    if (seg.startsWith("![")) return seg // skip image markdown
    return seg.replace(re, "<mark>$1</mark>")
  }).join("")
}

const rendered = computed(() => {
  todoIndex = 0
  try {
    let html = marked(highlightText(props.content, props.searchQuery || ""), { renderer }) as string
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
  // Todo checkbox toggle
  const checkbox = target.closest('input[type="checkbox"][data-todo-idx]') as HTMLInputElement
  if (checkbox) {
    const idx = parseInt(checkbox.getAttribute("data-todo-idx") || "0", 10)
    emit("todo-toggle", idx)
    return
  }
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
  const raw = copyBtn.getAttribute("data-code")
  if (!raw) return
  const code = decodeURIComponent(raw)
  function doCopy(text: string) {
    try {
      if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
        return navigator.clipboard.writeText(text)
      }
    } catch { /* clipboard API not supported */ }
    const ta = document.createElement("textarea")
    ta.value = text; ta.style.position = "fixed"; ta.style.opacity = "0"
    document.body.appendChild(ta); ta.select()
    try { document.execCommand("copy") } catch { /* fallback failed */ }
    document.body.removeChild(ta)
    return Promise.resolve()
  }
  doCopy(code).then(() => {
    copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M20 6L9 17l-5-5\"/></svg>"
    setTimeout(() => { copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
  }).catch(() => {
    copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"#ef4444\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><line x1=\"18\" y1=\"6\" x2=\"6\" y2=\"18\"/><line x1=\"6\" y1=\"6\" x2=\"18\" y2=\"18\"/></svg>"
    setTimeout(() => { copyBtn.innerHTML = "<svg viewBox=\"0 0 24 24\" width=\"14\" height=\"14\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><rect x=\"9\" y=\"9\" width=\"13\" height=\"13\" rx=\"2\" ry=\"2\"/><path d=\"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1\"/></svg>" }, 2000)
  })
}
</script>

<template>
  <div class="markdown-body" @click="handleClick" v-html="rendered" />
  <teleport to="body">
    <div v-if="zoomedImage" class="zoom-overlay" @click="zoomedImage = ''">
      <button class="zoom-close-btn" @click.stop="zoomedImage = ''">
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
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
.markdown-body :deep(code) { background: rgba(var(--v-theme-on-surface),.08); border-radius: 4px; padding: .15em .4em; font-size: .9em; font-family: var(--code-font); }
.markdown-body :deep(pre) { background: rgba(var(--v-theme-on-surface),.05); border-radius: 8px; padding: .75em 1em; overflow-x: auto; margin: .5em 0; }
.markdown-body :deep(pre code) { background: none; padding: 0; font-size: .85em; font-family: var(--code-font); }
.markdown-body :deep(table) { border-collapse: collapse; width: 100%; margin: .5em 0; }
.markdown-body :deep(th),.markdown-body :deep(td) { border: 1px solid rgba(var(--v-theme-on-surface),.15); padding: .4em .6em; text-align: left; }
.markdown-body :deep(img) { max-width: 100%; max-height: 220px; border-radius: 10px; cursor: zoom-in; box-shadow: 0 2px 8px rgba(0,0,0,0.06); transition: transform 0.2s, box-shadow 0.2s; }
.markdown-body :deep(img:hover) { transform: scale(1.01); box-shadow: 0 4px 16px rgba(0,0,0,0.1); }
.markdown-body :deep(a) { color: rgb(var(--v-theme-primary)); text-decoration: none; transition: text-decoration 0.15s, opacity 0.15s; }
.markdown-body :deep(a:hover) { text-decoration: underline; opacity: 0.85; }
.markdown-body :deep(.code-block-wrapper) { position: relative; margin: .5em 0; border-radius: 10px; overflow: hidden; border: 1px solid rgba(var(--v-theme-on-surface), 0.06); }
.markdown-body :deep(.code-header) {
  display: flex; align-items: center; gap: 6px; padding: 8px 12px;
  background: rgba(var(--v-theme-on-surface), 0.03);
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.04);
}
.markdown-body :deep(.code-dot) { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.markdown-body :deep(.code-dot.red) { background: #ff5f57; }
.markdown-body :deep(.code-dot.yellow) { background: #febc2e; }
.markdown-body :deep(.code-dot.green) { background: #28c840; }
.markdown-body :deep(.code-lang) { margin-left: auto; font-size: 0.7rem; opacity: 0.35; text-transform: uppercase; letter-spacing: 0.5px; }
.markdown-body :deep(.code-body) { display: flex; }
.markdown-body :deep(.code-gutter) {
  display: flex; flex-direction: column; align-items: flex-end; padding: 12px 8px;
  background: rgba(var(--v-theme-on-surface), 0.02); user-select: none;
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.04);
}
.markdown-body :deep(.code-line-num) {
  font-size: 0.72em; line-height: 1.6; color: rgba(var(--v-theme-on-surface), 0.2);
  font-family: var(--code-font);
}
.markdown-body :deep(.code-body pre) { background: none; border-radius: 0; margin: 0; padding: 12px 0; }
.markdown-body :deep(pre) { margin: 0; }
.markdown-body :deep(.copy-btn) {
  position: absolute; top: 4px; right: 4px; width: 28px; height: 28px;
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

.markdown-body :deep(mark) {
  background: rgba(var(--v-theme-warning), 0.35);
  color: inherit;
  border-radius: 3px;
  padding: 0 2px;
}

/* Todo list styling */
.markdown-body :deep(ul:has(.todo-item)) {
  padding-left: 0;
  list-style: none;
}
.markdown-body :deep(.todo-item) {
  list-style: none;
  margin: 2px 0;
  padding-left: 0;
}
.markdown-body :deep(.todo-label) {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  position: relative;
}
.markdown-body :deep(.todo-checkbox) {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}
.markdown-body :deep(.todo-checkmark) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 2px solid rgba(var(--v-theme-on-surface), 0.25);
  border-radius: 4px;
  flex-shrink: 0;
  transition: all 0.15s;
  position: relative;
}
.markdown-body :deep(.todo-checkbox:checked + .todo-checkmark) {
  background: rgb(var(--v-theme-primary));
  border-color: rgb(var(--v-theme-primary));
  transform: scale(0.9);
  animation: checkPop 0.25s ease;
}
@keyframes checkPop {
  0% { transform: scale(0.9); }
  50% { transform: scale(1.15); }
  100% { transform: scale(1); }
}
.markdown-body :deep(.todo-checkbox:checked + .todo-checkmark::after) {
  content: "";
  position: absolute;
  left: 4px;
  top: 1px;
  width: 5px;
  height: 9px;
  border: solid #fff;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}
.markdown-body :deep(.todo-checkbox:not(:checked) + .todo-checkmark:hover) {
  border-color: rgba(var(--v-theme-primary), 0.5);
}
.markdown-body :deep(.todo-text.done) {
  text-decoration: line-through;
  opacity: 0.55;
}
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
