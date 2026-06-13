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
  const encoded = encodeURIComponent(text)
  return `<div class="code-block-wrapper">
    <div class="code-header"><span class="code-dot red"></span><span class="code-dot yellow"></span><span class="code-dot green"></span><span class="code-lang">${lang || ""}</span><button class="copy-btn" data-code="${encoded}" title="复制代码"><svg style="pointer-events:none" viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button></div>
    <div class="code-body"><pre><code class="language-${lang || ""}">${highlighted}</code></pre></div>
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
const expandedGrids = ref<Set<string>>(new Set())
const loadedRepos = ref<Set<string>>(new Set())
const githubRepos = ref<Record<string, { name: string; fullName: string; description: string; stars: number; language: string; license: string; url: string }>>({})

function extractGitHubRepos(text: string): string[] {
  const matches = text.matchAll(/https?:\/\/github\.com\/([\w.-]+)\/([\w.-]+)\b/g)
  const repos = new Set<string>()
  for (const m of matches) repos.add(`${m[1]}/${m[2]}`)
  return [...repos]
}

async function fetchGitHubRepos() {
  const repos = extractGitHubRepos(props.content)
  if (!repos.length) return
  for (const fullName of repos) {
    if (githubRepos.value[fullName]) continue
    try {
      const res = await fetch(`/api/gh/repos/${fullName}`)
      if (!res.ok) { console.warn(`GitHub API ${fullName}: ${res.status}`); continue }
      const d = await res.json()
      githubRepos.value = { ...githubRepos.value, [fullName]: {
        name: d.name, fullName: d.full_name, description: d.description || "",
        stars: d.stargazers_count, language: d.language || "",
        license: d.license?.spdx_id || "", url: d.html_url,
      }}
      loadedRepos.value = new Set(loadedRepos.value).add(fullName)
    } catch { console.warn("failed silently") }
  }
}

watch(() => props.content, () => { githubRepos.value = {}; fetchGitHubRepos() }, { immediate: false })
onMounted(fetchGitHubRepos)

const LANG_COLORS: Record<string, string> = {
  TypeScript: "#3178C6", JavaScript: "#F7DF1E", Python: "#3572A5", Go: "#00ADD8",
  Rust: "#DEA584", Java: "#B07219", C: "#555555", "C++": "#F34B7D", "C#": "#178600",
  Ruby: "#701516", PHP: "#4F5D95", Swift: "#FFAC45", Kotlin: "#A97BFF",
  Dart: "#00B4AB", Lua: "#000080", Shell: "#89E051", HTML: "#E34F26", CSS: "#563D7C",
  Vue: "#41B883", Solid: "#2C4F7C", Svelte: "#FF3E00", Scala: "#C22D40",
  Elixir: "#6E4A7E", Clojure: "#DB5855", Haskell: "#5E5086", R: "#198CE7",
}
function langColor(lang: string): string {
  return LANG_COLORS[lang] || "rgba(var(--v-theme-on-surface), 0.3)"
}

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
  // Ensure reactivity when repos are loaded or grids expanded
  void loadedRepos.value.has(""); void expandedGrids.value.has("");
  try {
    // Strip loaded GitHub URLs from content
    let content = props.content
    for (const repo of loadedRepos.value) {
      content = content.replace(new RegExp(`https?://github\\.com/${repo.replace("/", "\\/")}\\b`, "g"), "").trim()
    }
    let html = marked(highlightText(content, props.searchQuery || ""), { renderer }) as string
    let idx = 0
    html = html.replace(/((?:<p><img[^>]*><\/p>\s*)+)/g, (match) => {
      const images = match.match(/<img[^>]*>/g)
      if (!images || images.length < 2) return match
      const id = "g" + idx++
      const maxVisible = window.innerWidth <= 640 ? 3 : 4
      const isExpanded = expandedGrids.value.has(id)
      const visible = isExpanded ? images : images.slice(0, maxVisible)
      const remaining = images.length - maxVisible
      return `<div class="img-grid" data-id="${id}">
        ${visible.map((img, i) => {
          const isLast = i === maxVisible - 1 && remaining > 0 && !isExpanded
          return `<div class="img-grid-cell${isLast ? ' img-grid-overflow' : ''}">
            ${img}${isLast ? `<div class="img-grid-overlay" data-id="${id}"><span class="img-grid-more">+${remaining}</span></div>` : ''}
          </div>`
        }).join("")}
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
  // Expand collapsed image grid
  const cell = target.closest(".img-grid-overflow") as HTMLElement
  if (cell) {
    const grid = cell.closest(".img-grid") as HTMLElement
    const id = grid?.getAttribute("data-id")
    if (id) { expandedGrids.value.add(id); expandedGrids.value = new Set(expandedGrids.value) }
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
  <div class="markdown-body" @click="handleClick" v-html="rendered" /><!-- eslint-disable-line vue/no-v-html -->
  <div v-for="(repo, key) in githubRepos" :key="key" class="gh-card">
    <a :href="repo.url" target="_blank" rel="noopener" class="gh-card-inner">
      <div class="gh-card-header">
        <span class="gh-icon"><v-icon size="small">mdi-github</v-icon></span>
        <span class="gh-name">{{ repo.name }}</span>
      </div>
      <div v-if="repo.description" class="gh-desc">{{ repo.description }}</div>
      <div class="gh-meta">
        <span v-if="repo.language" class="gh-meta-item"><span class="gh-lang-dot" :style="{ background: langColor(repo.language) }" />{{ repo.language }}</span>
        <span class="gh-meta-item">⭐ {{ repo.stars >= 1000 ? (repo.stars / 1000).toFixed(1) + 'k' : repo.stars }}</span>
        <span v-if="repo.license" class="gh-meta-item">{{ repo.license }}</span>
      </div>
    </a>
  </div>
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
.markdown-body :deep(h4),.markdown-body :deep(h5),.markdown-body :deep(h6) { margin: .3em 0 .15em; line-height: 1.3; }
.markdown-body :deep(h1) { font-size: 1.4em; }
.markdown-body :deep(h2) { font-size: 1.25em; }
.markdown-body :deep(h3) { font-size: 1.1em; }
.markdown-body :deep(p) { margin: .2em 0; }
.markdown-body :deep(p:has(img)) { margin: .1em 0; }
.markdown-body :deep(ul),.markdown-body :deep(ol) { padding-left: 1.4em; margin: .2em 0; }
.markdown-body :deep(blockquote) { border-left: 3px solid rgba(var(--v-theme-primary),.5); padding-left: .75em; margin: .25em 0; opacity: .85; }
.markdown-body :deep(code) { background: rgba(var(--v-theme-on-surface),.08); border-radius: 4px; padding: .15em .4em; font-size: .9em; font-family: var(--code-font); }
.markdown-body :deep(pre) { background: rgba(var(--v-theme-on-surface),.05); border-radius: 8px; padding: .75em 1em; overflow-x: auto; margin: .5em 0; }
.markdown-body :deep(pre code) { background: none; padding: 0; font-size: .85em; font-family: var(--code-font); }
.markdown-body :deep(table) { border-collapse: collapse; width: 100%; margin: .5em 0; }
.markdown-body :deep(th),.markdown-body :deep(td) { border: 1px solid rgba(var(--v-theme-on-surface),.15); padding: .4em .6em; text-align: left; }
.markdown-body :deep(img) { max-width: 100%; max-height: 200px; border-radius: 12px; cursor: zoom-in; box-shadow: 0 2px 8px rgba(0,0,0,0.06); transition: transform 0.2s, box-shadow 0.2s; border: 1px solid rgba(var(--v-theme-on-surface), 0.04); object-fit: contain; background: rgba(var(--v-theme-on-surface), 0.02); }
.markdown-body :deep(img:hover) { transform: scale(1.02); box-shadow: 0 6px 24px rgba(0,0,0,0.1); }
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
.markdown-body :deep(.code-lang) { font-size: 0.7rem; opacity: 0.3; text-transform: uppercase; letter-spacing: 0.5px; flex: 1; }
.markdown-body :deep(.copy-btn) {
  width: 24px; height: 24px; border-radius: 5px; border: none;
  background: transparent; color: rgba(var(--v-theme-on-surface), 0.3);
  cursor: pointer; transition: all 0.15s; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
}
.markdown-body :deep(.copy-btn:hover) { background: rgba(var(--v-theme-on-surface), 0.06); color: rgba(var(--v-theme-on-surface), 0.7); }
.markdown-body :deep(.copy-btn svg) { pointer-events: none; }
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
.markdown-body :deep(.copy-btn:hover) { background: rgba(var(--v-theme-primary), 0.08); color: rgb(var(--v-theme-primary)); border-color: rgba(var(--v-theme-primary), 0.3); }
.markdown-body :deep(.carousel-wrap) { position: relative; margin: .5em 0; border-radius: 8px; overflow: hidden; background: rgba(var(--v-theme-on-surface), 0.03); }
.markdown-body :deep(.img-grid) { display: grid; grid-template-columns: repeat(4, 1fr); gap: 4px; margin: .2em 0; }
.markdown-body :deep(.img-grid-cell) { overflow: hidden; border-radius: 12px; aspect-ratio: 1; }
.markdown-body :deep(.img-grid-cell img) { width: 100%; height: 100%; object-fit: cover; cursor: zoom-in; transition: transform 0.2s; border-radius: 12px; }
.markdown-body :deep(.img-grid-cell img:hover) { transform: scale(1.03); }
.markdown-body :deep(.img-grid-overflow) { position: relative; cursor: pointer; }
.markdown-body :deep(.img-grid-overlay) {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  background: rgba(0,0,0,0.12);
  border-radius: 12px; transition: background 0.2s; cursor: pointer;
}
.markdown-body :deep(.img-grid-overlay:hover) { background: rgba(0,0,0,0.45); }
.markdown-body :deep(.img-grid-more) { color: #fff; font-size: 1.5rem; font-weight: 700; }
@media (max-width: 640px) { .markdown-body :deep(.img-grid) { grid-template-columns: repeat(3, 1fr); } }

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
/* GitHub repo card */
.gh-card { margin: .5em 0; }
.gh-card-inner {
  display: flex; flex-direction: column; gap: 6px;
  padding: 12px 16px; border-radius: 12px; text-decoration: none;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background: rgba(var(--v-theme-surface), 0.5);
  backdrop-filter: blur(8px); transition: all 0.15s;
  color: rgb(var(--v-theme-on-surface));
}
.gh-card-inner:hover { border-color: rgba(var(--v-theme-primary), 0.2); background: rgba(var(--v-theme-surface), 0.65); }
.gh-card-header { display: flex; align-items: center; gap: 6px; }
.gh-icon { opacity: 0.4; display: flex; }
.gh-name { font-weight: 600; font-size: 0.9rem; }
.gh-desc { font-size: 0.8rem; opacity: 0.7; line-height: 1.4; }
.gh-meta { display: flex; align-items: center; gap: 12px; font-size: 0.75rem; opacity: 0.5; }
.gh-meta-item { display: flex; align-items: center; gap: 4px; }
.gh-lang-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }

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
