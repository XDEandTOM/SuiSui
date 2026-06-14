<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from "vue"
import { useDisplay } from "vuetify"
import { useNotesStore, type Note } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
import { authFetch } from "@/utils/api"
import NoteCard from "@/components/NoteCard.vue"
import Heatmap from "@/components/Heatmap.vue"
import InlineEditor from "@/components/InlineEditor.vue"
import SidePanel from "@/components/SidePanel.vue"

defineProps<{ mobileHeatmap: boolean }>()
const emit = defineEmits<{ "close-heatmap": [] }>()

const store = useNotesStore()
const auth = useAuthStore()
const { mobile } = useDisplay()
const isMobile = mobile

// Simple local search input bound to store
const localSearch = ref("")
const selectedTag = ref("")
const selectedDay = ref("")
const viewMode = ref<"list" | "timeline">("list")

const TAG_COLORS = ["primary", "teal", "orange", "pink", "indigo", "brown", "cyan", "deep-purple", "amber", "blue-grey"]
function tagColor(tag: string) {
  let h = 0; for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) | 0
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}

const siteIcp = ref("")
const versionText = ref("")
const icpLink = "https://beian.miit.gov.cn/#/Integrated/index"


const showShortcuts = ref(false)



const zoomedUpload = ref("")
const showTrash = ref(false)
const deletedNotes = ref<Note[]>([])
const editorRef = ref<InstanceType<typeof InlineEditor> | null>(null)
const newNotesCount = ref(0)
let lastActionAt = 0
let pollingTimer: ReturnType<typeof setInterval> | null = null

watch(showTrash, v => { if (v) fetchDeletedNotes() })

function onNoteSubmitted() {
  lastActionAt = Date.now()
}


const timelineGroups = computed(() => {
  const groups: { date: string; notes: Note[] }[] = []
  const today = new Date()
  for (const note of store.notes) {
    const d = new Date(note.createdAt)
    let label: string
    const diffDays = Math.floor((today.getTime() - d.getTime()) / 86400000)
    if (diffDays === 0) label = "今天"
    else if (diffDays === 1) label = "昨天"
    else label = d.toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" })
    const last = groups[groups.length - 1]
    if (last && last.date === label) last.notes.push(note)
    else groups.push({ date: label, notes: [note] })
  }
  return groups
})

interface OutlineHeading { level: number; text: string; noteId: string }
const outline = computed(() => {
  const result: OutlineHeading[] = []
  for (const note of store.notes) {
    const matches = note.content.matchAll(/^(#{1,6})\s+(.+)$/gm)
    for (const m of matches) {
      result.push({ level: m[1].length, text: m[2].trim(), noteId: note.id })
    }
  }
  return result
})

function scrollToNote(noteId: string) {
  document.querySelector(`[data-note-id="${noteId}"]`)?.scrollIntoView({ behavior: "smooth", block: "start" })
}

// Scroll sentinel
const scrollSentinel = ref<HTMLDivElement | null>(null)
let scrollObserver: IntersectionObserver | null = null





onMounted(async () => {
  await store.fetchNotes(true)
  await loadSiteIcp()
  fetchVersion()
  setupInfiniteScroll()
  document.addEventListener("keydown", handleGlobalKeydown)
  startPolling()
})
onBeforeUnmount(() => {
  zoomedUpload.value = ""
  if (scrollObserver) scrollObserver.disconnect()
  document.removeEventListener("keydown", handleGlobalKeydown)
  stopPolling()
})

function setupInfiniteScroll() {
  if (scrollObserver) scrollObserver.disconnect()
  scrollObserver = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && store.hasMore && !store.loadingMore) {
      store.fetchNotes(false)
    }
  }, { rootMargin: "600px" })
  nextTick(() => {
    if (scrollSentinel.value) scrollObserver?.observe(scrollSentinel.value)
  })
}

// Reactive filter watchers — reset and re-fetch when filter changes
watch([localSearch, selectedTag, selectedDay], () => {
  store.searchQuery = localSearch.value
  store.selectedTag = selectedTag.value
  store.selectedDay = selectedDay.value
  store.fetchNotes(true)
  // Reconnect scroll observer after DOM update
  nextTick(setupInfiniteScroll)
})

// Re-observe sentinel after notes change (e.g. after appending)
watch(() => store.notes.length, () => {
  nextTick(setupInfiniteScroll)
})

async function loadSiteIcp() {
  try {
    const r = await fetch("/api/settings")
    if (r.ok) {
      const s = await r.json()
      siteIcp.value = s.site_icp || ""
    }
  } catch { /* ignore */ }
}

function openGithub() { window.open("https://github.com/Linraintong/SuiSui", "_blank") }

async function fetchVersion() {
  try {
    const r = await fetch("https://api.github.com/repos/Linraintong/SuiSui/releases/latest")
    if (r.ok) {
      const d = await r.json()
      versionText.value = d.tag_name || ""
    }
  } catch { versionText.value = "" }
}


async function fetchDeletedNotes() {
  try {
    const res = await authFetch(`/api/notes/trash?username=${auth.userName}`)
    if (res.ok) { deletedNotes.value = await res.json() }
  } catch { console.warn("deletedNotes fetch failed") }
}
async function restoreNote(id: string) {
  try {
    const res = await authFetch(`/api/notes/${id}/restore?username=${auth.userName}`,{method:"PATCH"})
    if (res.ok) { deletedNotes.value = deletedNotes.value.filter(n=>n.id!==id); await store.fetchNotes(true) }
  } catch { console.warn("restoreNote failed") }
}
async function deleteForever(id: string) {
  try {
    const res = await authFetch(`/api/notes/${id}/hard-delete?username=${auth.userName}`,{method:"DELETE"})
    if (res.ok) { deletedNotes.value = deletedNotes.value.filter(n => n.id !== id) }
  } catch { console.warn("deleteForever failed") }
}

function handleEdit(memo: Note) {
  editorRef.value?.handleEdit(memo)
}

function handleGlobalKeydown(e: KeyboardEvent) {
  // Don't override when typing in inputs
  const tag = (e.target as HTMLElement)?.tagName
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return
  if (e.ctrlKey || e.metaKey || e.altKey) return

  if (e.key === "/") {
    e.preventDefault()
    // Focus search input
    const searchEl = document.querySelector<HTMLInputElement>('[data-search-input] input')
    searchEl?.focus()
  } else if (e.key === "?") {
    showShortcuts.value = !showShortcuts.value
  } else if (e.key === "Escape" && showShortcuts.value) {
    showShortcuts.value = false
  }
}

function startPolling() {
  const evtSource = new EventSource('/api/events')
  evtSource.addEventListener('note', () => {
    if (Date.now() - lastActionAt < 3000) return
    fetch('/api/notes?limit=1&offset=0').then(r => r.json()).then(data => {
      if (data.total > store.total && store.total > 0) {
        newNotesCount.value = data.total - store.total
      }
    }).catch(() => {})
  })
  pollingTimer = setInterval(async () => {
    if (Date.now() - lastActionAt < 3000) return
    try {
      const res = await fetch(`/api/notes?limit=1&offset=0`)
      if (res.ok) {
        const data = await res.json()
        if (data.total > store.total && store.total > 0) {
          newNotesCount.value = data.total - store.total
        }
      }
    } catch { /* ignore polling errors */ }
  }, 15000)
}

function stopPolling() {
  if (pollingTimer) { clearInterval(pollingTimer); pollingTimer = null }
}

function refreshNotes() {
  store.fetchNotes(true)
  newNotesCount.value = 0
  // Also update total via a lightweight fetch
  fetch('/api/notes?limit=1&offset=0').then(r => r.json()).then(data => {
    if (data.total) store.total = data.total
  }).catch(() => {})
}

async function movePinnedNote(note: Note, dir: "up" | "down") {
  const pinned = store.notes.filter(n => n.pinned)
  const idx = pinned.findIndex(n => n.id === note.id)
  if (idx === -1) return
  const swap = idx + (dir === "up" ? -1 : 1)
  if (swap < 0 || swap >= pinned.length) return
  ;[pinned[idx], pinned[swap]] = [pinned[swap], pinned[idx]]
  const order = pinned.map(n => n.id)
  await authFetch("/api/notes/reorder", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ order }),
  })
  await store.fetchNotes(true)
}
</script>

<template>
  <div class="notes-layout" :class="{ mobile: isMobile }">
    <div class="side-col">
      <SidePanel
        :search="localSearch"
        :selected-tag="selectedTag"
        :selected-day="selectedDay"
        :all-tags="store.allTags"
        :version-text="versionText"
        :outline="outline"
        @update:search="localSearch = $event"
        @update:selected-tag="selectedTag = $event"
        @update:selected-day="selectedDay = $event"
        @scroll-to-note="scrollToNote" />
    </div>

    <div class="main-col">
      <v-dialog :model-value="mobileHeatmap" max-width="400" scrollable persistent transition="dialog-bottom-transition"
        @update:model-value="v => !v && emit('close-heatmap')">
        <v-card class="rounded-xl pa-4 heatmap-dialog-card">
          <div class="d-flex align-center mb-3">
            <span class="text-subtitle-2 font-weight-medium">活动日历</span>
            <v-spacer /><v-btn icon="mdi-close" size="small" variant="text" @click="emit('close-heatmap')" />
          </div>
          <v-text-field v-model="localSearch" prepend-inner-icon="mdi-magnify" label="搜索备忘..." variant="outlined" hide-details density="compact" clearable class="mb-3 rounded-search search-border" />
          <Heatmap class="mb-4" style="border-color:#424242 !important" @select-day="selectedDay = $event; emit('close-heatmap')" />
          <v-card variant="outlined" class="rounded-xl pa-4 side-card">
            <div class="d-flex align-center ga-2 mb-3"><span class="text-subtitle-2 font-weight-medium">标签</span></div>
            <div class="d-flex flex-wrap ga-1">
              <v-chip v-for="[tag] in store.allTags" :key="tag" size="x-small" class="tag-chip"
                :color="tagColor(tag)"
                :variant="selectedTag === tag ? 'flat' : 'outlined'" @click="selectedTag = selectedTag === tag ? '' : tag; emit('close-heatmap')">
                #{{ tag }}
              </v-chip>
              <div v-if="!store.allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
            </div>
          </v-card>
          <div v-if="versionText" class="d-flex justify-center mt-2">
            <v-chip size="x-small" variant="tonal" color="primary" class="version-chip" style="cursor:pointer" prepend-icon="mdi-github" @click="openGithub">
              {{ versionText }}
            </v-chip>
          </div>
        </v-card>
      </v-dialog>

      <v-dialog v-model="showTrash" max-width="500" scrollable>
        <v-card class="rounded-xl pa-4">
          <div class="d-flex align-center mb-3">
            <span class="text-subtitle-2 font-weight-medium">回收站</span>
            <v-spacer />
            <v-btn icon="mdi-close" size="x-small" variant="text" @click="showTrash = false" />
          </div>
          <div v-if="!deletedNotes.length" class="d-flex flex-column align-center py-4 text-medium-emphasis">
            <v-icon size="32" class="mb-2" color="rgba(var(--v-theme-on-surface),0.15)">mdi-delete-outline</v-icon>
            <span class="text-caption">回收站为空</span>
          </div>
          <div v-else class="d-flex flex-column ga-2">
            <div v-for="note in deletedNotes" :key="note.id" class="d-flex align-center ga-2 pa-2"
              style="border-bottom:1px solid rgba(var(--v-theme-on-surface),0.06)">
              <div class="flex-grow-1 text-caption" style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                {{ note.content?.replace(/!\[.*?\]\(.+?\)/g, "[图片]").substring(0, 60) }}
              </div>
              <v-btn icon="mdi-restore" size="x-small" variant="text" color="primary" title="恢复" @click="restoreNote(note.id)" />
              <v-btn icon="mdi-delete-forever" size="x-small" variant="text" color="error" title="永久删除" @click="deleteForever(note.id)" />
            </div>
          </div>
        </v-card>
      </v-dialog>

      <InlineEditor ref="editorRef" @submitted="onNoteSubmitted" @open-trash="showTrash = true" />
      <div v-if="!store.loaded" class="d-flex flex-column ga-3 px-1">
        <div v-for="i in 3" :key="i" class="skeleton-card">
          <div class="skeleton-row" style="width:65%" />
          <div class="skeleton-row" style="width:45%" />
          <div class="skeleton-row" style="width:85%" />
        </div>
      </div>
      <template v-else>
        <div v-if="selectedDay" class="date-filter-bar">
          <v-icon size="x-small" color="primary">mdi-calendar</v-icon>
          <span>{{ selectedDay }} 的碎片笔记</span>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="selectedDay = ''" />
        </div>
        <div class="view-bar">
          <div class="view-bar-btns">
            <button class="view-bar-btn" :class="{ active: viewMode === 'list' }" @click="viewMode = 'list'">
              <v-icon size="small">mdi-view-list</v-icon>
              <span>列表</span>
            </button>
            <button class="view-bar-btn" :class="{ active: viewMode === 'timeline' }" @click="viewMode = 'timeline'">
              <v-icon size="small">mdi-timeline</v-icon>
              <span>时间线</span>
            </button>
          </div>
        </div>
        <div v-if="newNotesCount > 0" class="new-notes-bar" @click="refreshNotes">
          <v-icon size="small" color="primary">mdi-arrow-up-circle</v-icon>
          <span>有 {{ newNotesCount }} 条新的碎片笔记</span>
        </div>
        <div v-if="store.notes.length === 0" class="empty-state">
          <div class="empty-illust">
            <template v-if="localSearch || selectedTag || selectedDay">
              <div class="empty-illust-inner search-empty">
                <v-icon size="40" class="empty-icon-main">mdi-magnify</v-icon>
                <div class="empty-icon-sub">
                  <v-icon size="18" color="error">mdi-emoticon-sad-outline</v-icon>
                </div>
              </div>
            </template>
            <template v-else>
              <div class="empty-illust-inner notes-empty">
                <v-icon size="44" class="empty-icon-main">mdi-pencil-box-multiple-outline</v-icon>
                <div class="empty-icon-sparkle"><v-icon size="12" color="warning">mdi-sparkles</v-icon></div>
                <div class="empty-icon-sparkle sparkle-2"><v-icon size="10" color="primary">mdi-sparkles</v-icon></div>
              </div>
            </template>
          </div>
          <p v-if="localSearch || selectedTag || selectedDay" class="empty-text-title">没有找到匹配的碎片笔记</p>
          <p v-else class="empty-text-title">还没有碎片笔记</p>
          <p v-if="!localSearch && !selectedTag && !selectedDay" class="empty-text-hint">点击上方编辑框，写下你的第一段记忆吧 ✨</p>
        </div>
        <Transition name="view-fade" mode="out-in">
          <div v-if="viewMode === 'list'" key="list" class="d-flex flex-column ga-4">
            <div v-for="(note, idx) in store.notes" :key="note.id" class="note-drag-wrapper"
              :style="{ animationDelay: `${idx * 0.05}s` }" :data-note-id="note.id">
              <NoteCard :memo="note" :search-query="localSearch" :logged-in="auth.isLoggedIn" @edit="handleEdit" @move-pin="movePinnedNote" />
            </div>
          </div>
          <div v-else key="timeline" class="timeline-view">
            <div v-for="(group, gi) in timelineGroups" :key="gi" class="timeline-group">
              <div class="timeline-date-label">{{ group.date }}</div>
              <div class="timeline-line" />
              <div v-for="(note, ni) in group.notes" :key="note.id" class="timeline-item"
                :style="{ animationDelay: `${(gi * 5 + ni) * 0.05}s` }">
                <div class="timeline-dot" />
                <NoteCard :memo="note" :search-query="localSearch" :logged-in="auth.isLoggedIn"
                  @edit="handleEdit" @move-pin="movePinnedNote" />
              </div>
            </div>
          </div>
        </Transition>
        <!-- Infinite scroll sentinel -->
        <div ref="scrollSentinel" class="scroll-sentinel">
          <div v-if="store.loadingMore" class="d-flex flex-column align-center py-4 ga-2">
            <v-progress-circular indeterminate size="32" color="primary" />
            <span class="text-caption text-medium-emphasis">加载中…</span>
          </div>
          <div v-else-if="!store.hasMore && store.notes.length > 0" class="scroll-end">
            <div class="scroll-end-line" />
            <span class="scroll-end-text">共 {{ store.total }} 条碎片笔记</span>
            <div class="scroll-end-line" />
          </div>
        </div>
      </template>
      <div v-if="siteIcp" class="text-center text-caption py-4 icp-text" style="opacity:0.6">
        <a :href="icpLink" target="_blank" rel="noopener" class="icp-link">{{ siteIcp }}</a>
      </div>
    </div>
  </div>

  <teleport to="body">
    <div v-if="zoomedUpload" class="zoom-overlay" @click="zoomedUpload = ''">
      <button class="zoom-close-btn" @click.stop="zoomedUpload = ''">
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
      </button>
      <img :src="zoomedUpload" class="zoom-img" @click.stop />
    </div>
  </teleport>

  <!-- Keyboard shortcuts dialog -->
  <v-dialog v-model="showShortcuts" max-width="360">
    <v-card class="rounded-xl pa-4">
      <div class="d-flex align-center mb-3">
        <span class="text-subtitle-2 font-weight-medium">⌨️ 快捷键</span>
        <v-spacer />
        <v-btn icon="mdi-close" size="x-small" variant="text" @click="showShortcuts = false" />
      </div>
      <div class="shortcut-list">
        <div class="shortcut-row"><kbd>/</kbd><span>聚焦搜索</span></div>
        <div class="shortcut-row"><kbd>?</kbd><span>显示此面板</span></div>
        <div class="shortcut-row"><kbd>Ctrl + Enter</kbd><span>发布/更新笔记</span></div>
        <div class="shortcut-row"><kbd>Ctrl + B</kbd><span>粗体</span></div>
        <div class="shortcut-row"><kbd>Ctrl + I</kbd><span>斜体</span></div>
        <div class="shortcut-row"><kbd>Escape</kbd><span>关闭面板</span></div>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.notes-layout { display: flex; gap: 28px; padding: 28px; max-width: 1200px; margin: 0 auto; align-items: flex-start; }
.md-toolbar { overflow-x: auto !important; overflow-y: hidden; white-space: nowrap !important; -webkit-overflow-scrolling: touch; scrollbar-width: thin; }
.md-toolbar::-webkit-scrollbar { height: 3px; }
.notes-layout.mobile { flex-direction: column; padding: 12px; gap: 12px; }
.side-col { width: 280px; flex-shrink: 0; position: sticky; top: 24px; align-self: flex-start; }
.notes-layout.mobile .side-col { opacity: 0; pointer-events: none; max-height: 0; overflow: hidden; transition: opacity 0.3s, max-height 0.3s; }
.main-col { flex: 1; min-width: 0; }
.tag-chip { cursor: pointer; }
.tag-chip:hover { opacity: 0.9; }
.rounded-search :deep(.v-field) { border-radius: 12px !important; }
.heatmap-dialog-card { border-color: #424242 !important; }
.icp-link { color: inherit; text-decoration: none; }
.icp-link:hover { text-decoration: underline; }

.inline-editor { width: 100%; }
.tool-sep-sm {
  width: 1px; height: 16px;
  background: rgba(var(--v-theme-on-surface), 0.08);
  flex-shrink: 0;
}
.empty-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 56px 16px; gap: 12px;
}
.empty-illust { position: relative; width: 80px; height: 80px; display: flex; align-items: center; justify-content: center; }
.empty-illust-inner { position: relative; display: flex; align-items: center; justify-content: center; }
.empty-icon-main { opacity: 0.15; }
.notes-empty .empty-icon-main { opacity: 0.12; }
.empty-icon-sub { position: absolute; bottom: -4px; right: -12px; }
.empty-icon-sparkle { position: absolute; top: -6px; right: -4px; opacity: 0.4; }
.sparkle-2 { top: -2px; right: -20px; opacity: 0.3; }
.empty-text-title { font-size: 1rem; font-weight: 600; margin: 0; text-align: center; }
.empty-text-hint { font-size: 0.82rem; color: rgba(var(--v-theme-on-surface), 0.45); margin: 0; text-align: center; }
.date-filter-bar {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 14px;
  margin-bottom: 8px;
  background: rgba(var(--v-theme-primary), 0.06);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  border: 1px solid rgba(var(--v-theme-primary), 0.12);
  border-radius: 10px;
  font-size: 0.82rem;
  color: rgb(var(--v-theme-primary));
}
.note-drag-wrapper {
  transition: opacity 0.15s, box-shadow 0.15s;
  border-radius: 12px;
  animation: cardFadeIn 0.4s ease both;
}

@keyframes cardFadeIn {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}
.scroll-sentinel {
  min-height: 80px;
}
.scroll-end {
  display: flex; align-items: center; gap: 12px; justify-content: center;
  padding: 16px 0; opacity: 0.3;
}
.scroll-end-line { flex: 1; max-width: 80px; height: 1px; background: rgba(var(--v-theme-on-surface), 0.15); }
.scroll-end-text { font-size: 0.72rem; white-space: nowrap; letter-spacing: 0.3px; }

/* Input focus glow */
:global(.v-field--focused) { box-shadow: 0 0 0 2px rgba(var(--v-theme-primary), 0.08); }

/* Timeline view */
.timeline-view { position: relative; }
.timeline-group { position: relative; padding-left: 28px; margin-bottom: 8px; }

/* View mode transition */
.view-fade-enter-active, .view-fade-leave-active { transition: opacity 0.15s ease; }
.view-fade-enter-from, .view-fade-leave-to { opacity: 0; }

/* New notes notification */
.new-notes-bar {
  display: flex; align-items: center; gap: 6px; justify-content: center;
  padding: 8px 14px; margin-bottom: 8px;
  background: rgba(var(--v-theme-primary), 0.06);
  border: 1px solid rgba(var(--v-theme-primary), 0.12);
  border-radius: 10px; cursor: pointer; font-size: 0.82rem;
  color: rgb(var(--v-theme-primary)); transition: background 0.15s;
}
.new-notes-bar:hover { background: rgba(var(--v-theme-primary), 0.1); }

/* Outline */
.outline-count {
  font-size: 0.65rem; background: rgba(var(--v-theme-on-surface), 0.06);
  padding: 0 6px; border-radius: 4px; color: rgba(var(--v-theme-on-surface), 0.4);
}
.outline-list { display: flex; flex-direction: column; gap: 1px; max-height: 240px; overflow-y: auto; }
.outline-item {
  display: flex; align-items: center; gap: 6px; padding: 4px 0;
  border-radius: 4px; cursor: pointer; transition: all 0.1s;
  font-size: 0.78rem; overflow: hidden;
}
.outline-item:hover { background: rgba(var(--v-theme-primary), 0.04); }
.outline-marker { flex-shrink: 0; height: 2px; border-radius: 1px; background: rgba(var(--v-theme-primary), 0.25); }
.outline-text { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.outline-more { font-size: 0.7rem; opacity: 0.35; padding: 2px 0; }

.timeline-date-label {
  font-size: 0.78rem; font-weight: 600; padding: 4px 0 8px;
  color: rgba(var(--v-theme-on-surface), 0.5);
  position: relative; z-index: 1;
}
.timeline-line {
  position: absolute; left: 10px; top: 24px; bottom: 0;
  width: 2px; background: rgba(var(--v-theme-primary), 0.12);
  border-radius: 1px;
}
.timeline-item {
  position: relative; margin-bottom: 8px;
  animation: cardFadeIn 0.35s ease both;
}
.timeline-dot {
  position: absolute; left: -22px; top: 16px;
  width: 8px; height: 8px; border-radius: 50%;
  background: rgb(var(--v-theme-primary)); opacity: 0.3;
  z-index: 1;
}
.view-toggle { border-radius: 8px; overflow: hidden; }
.view-toggle :deep(.v-btn) { border-radius: 0 !important; }
.view-bar {
  display: flex; align-items: center; margin-bottom: 8px;
}
.view-bar-btns {
  display: flex; border-radius: 8px; overflow: hidden;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}
.view-bar-btn {
  display: flex; align-items: center; gap: 4px; padding: 5px 14px;
  border: none; background: transparent; cursor: pointer;
  font-size: 0.78rem; color: rgba(var(--v-theme-on-surface), 0.5);
  transition: all 0.15s; font-family: inherit;
}
.view-bar-btn:not(:last-child) { border-right: 1px solid rgba(var(--v-theme-on-surface), 0.06); }
.view-bar-btn:hover { color: rgba(var(--v-theme-on-surface), 0.8); }
.view-bar-btn.active {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
}

/* Skeleton loading */
.skeleton-card {
  border-radius: 14px; padding: 16px;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.06);
  display: flex; flex-direction: column; gap: 10px;
}
.skeleton-row {
  height: 14px; border-radius: 6px;
  background: linear-gradient(90deg,
    rgba(var(--v-theme-on-surface), 0.06) 25%,
    rgba(var(--v-theme-on-surface), 0.12) 50%,
    rgba(var(--v-theme-on-surface), 0.06) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

@media (max-width: 768px) {
  .notes-layout.mobile { flex-direction: column; padding: 12px; gap: 8px; }
  .notes-layout.mobile .main-col { width: 100%; }
}
</style>

<style>
.shortcut-list { display: flex; flex-direction: column; gap: 8px; }
.shortcut-row { display: flex; align-items: center; gap: 12px; font-size: 0.85rem; }
.shortcut-row kbd {
  display: inline-block; min-width: 60px; padding: 3px 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 5px; font-size: 0.75rem; font-family: inherit;
  text-align: center; color: rgb(var(--v-theme-on-surface));
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
