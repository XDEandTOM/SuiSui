<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from "vue"
import { useDisplay } from "vuetify"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
import { authFetch } from "@/utils/api"
import NoteCard from "@/components/NoteCard.vue"
import Heatmap from "@/components/Heatmap.vue"

const props = defineProps<{ mobileHeatmap: boolean }>()
const emit = defineEmits<{ "close-heatmap": [] }>()

const store = useNotesStore()
const auth = useAuthStore()
const { mobile } = useDisplay()
const isMobile = mobile
const searchQuery = ref("")
const selectedTag = ref("")
const selectedDay = ref("")

const siteIcp = ref("")
const versionText = ref("")
const icpLink = "https://beian.miit.gov.cn/#/Integrated/index"
const DRAFT_KEY = "suisui-draft"

const inlineContent = ref("")
const inlineTagsInput = ref<string[]>([])
const showInlineTags = ref(false)
const inlineUploading = ref(false)
const inlineTextarea = ref<HTMLTextAreaElement | null>(null)
const inlineFileInput = ref<HTMLInputElement | null>(null)
const uploadedImages = ref<string[]>([])
const editingNoteId = ref("")
const zoomedUpload = ref("")
const showTrash = ref(false)
const deletedNotes = ref([])
const hasDraft = computed(() => !!(inlineContent.value || uploadedImages.value.length))

function saveDraft() {
  if (!auth.isLoggedIn) return
  const draft = { content: inlineContent.value, tags: inlineTagsInput.value, images: uploadedImages.value, editingId: editingNoteId.value }
  try { localStorage.setItem(DRAFT_KEY, JSON.stringify(draft)) } catch {}
}

function restoreDraft() {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) return
    const draft = JSON.parse(raw)
    if (draft.content) inlineContent.value = draft.content
    if (draft.tags) { inlineTagsInput.value = typeof draft.tags === "string" ? draft.tags.split(/[,，]/).map(t => t.trim()).filter(Boolean) : draft.tags; showInlineTags.value = true }
    if (draft.images?.length) uploadedImages.value = draft.images
    if (draft.editingId) editingNoteId.value = draft.editingId
  } catch {}
}

function clearDraft() {
  localStorage.removeItem(DRAFT_KEY)
}

// Auto-save draft on any editor input (debounced)
let draftTimer: ReturnType<typeof setTimeout> | null = null
watch([inlineContent, inlineTagsInput, uploadedImages, editingNoteId], () => {
  if (draftTimer) clearTimeout(draftTimer)
  draftTimer = setTimeout(saveDraft, 500)
}, { deep: true })

onMounted(async () => {
  await store.fetchNotes(); await loadSiteIcp(); fetchVersion()
  restoreDraft()
})
onBeforeUnmount(() => { zoomedUpload.value = "" })

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

const allTags = computed(() => {
  const tagCount = new Map()
  for (const n of store.notes) {
    if (n.tags && Array.isArray(n.tags)) for (const t of n.tags) tagCount.set(t, (tagCount.get(t) || 0) + 1)
  }
  return [...tagCount.entries()].sort((a, b) => b[1] - a[1])
})

const filteredNotes = computed(() => {
  let list = store.notes
  if (searchQuery.value && searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(n => {
      const text = (n.content || "").replace(/!\[.*?\]\(.+?\)/g, "").toLowerCase()
      return text.includes(q) || (n.tags && Array.isArray(n.tags) && n.tags.some(t => t && t.toLowerCase().includes(q)))
    })
  }
  if (selectedTag.value) list = list.filter(n => n.tags && Array.isArray(n.tags) && n.tags.includes(selectedTag.value))
  if (selectedDay.value) {
    list = list.filter(n => {
      const d = new Date(n.createdAt)
      const ds = d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0")
      return ds === selectedDay.value
    })
  }
  return list
})

function insertMd(b,f,fb) {
  const el = document.querySelector(".inline-textarea")
  if (!el) { inlineContent.value += fb; return }
  const start = el.selectionStart, end = el.selectionEnd
  const t = inlineContent.value, sel = t.substring(start, end)
  inlineContent.value = t.slice(0,start) + b + (sel||fb) + f + t.slice(end)
  nextTick(() => { el.focus(); el.selectionStart = el.selectionEnd = start + b.length + (sel||fb).length })
}
function insertBold() { insertMd("**","**","粗体") }
function insertItalic() { insertMd("*","*","斜体") }
function insertHeading() { insertMd("\n## ","","标题") }
function insertCode() { insertMd("`","`","code") }
function insertLink() { insertMd("[","](url)","链接文字") }
function insertList() { insertMd("\n- ","","列表项") }
function insertOrderedList() { insertMd("\n1. ","","列表项") }
function insertQuote() { insertMd("\n> ","","引用") }
function insertStrikethrough() { insertMd("~~","~~","删除线") }
function insertHr() { insertMd("\n---\n","","") }
function insertTable() { insertMd("\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n| 内容 | 内容 | 内容 |","","") }
function insertTodo() { insertMd("\n- [ ] ","","待办事项") }
function insertCodeBlock() { insertMd("\n```\n","\n```\n","代码块") }

async function fetchDeletedNotes() {
  try {
    const res = await authFetch(`/api/notes/trash?username=${auth.userName}`)
    if (res.ok) {
      deletedNotes.value = await res.json()
    }
  } catch { }
}
async function restoreNote(id) {
  try {
    const res = await authFetch(`/api/notes/${id}/restore?username=${auth.userName}`,{method:"PATCH"})
    if (res.ok) { deletedNotes.value = deletedNotes.value.filter(n=>n.id!==id); await store.fetchNotes() }
  } catch { }
}
async function deleteForever(id: string) {
  try {
    const res = await authFetch(`/api/notes/${id}/hard-delete?username=${auth.userName}`,{method:"DELETE"})
    if (res.ok) {
      deletedNotes.value = deletedNotes.value.filter(n => n.id !== id)
    }
  } catch { }
}


function onInlineKeydown(e: KeyboardEvent) {
  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) submitInline()
}

async function submitInline() {
  if ((!inlineContent.value.trim() && !uploadedImages.value.length) || !auth.isLoggedIn) return
  const tags = Array.isArray(inlineTagsInput.value) ? inlineTagsInput.value.map(t => t.trim()).filter(Boolean) : []
  let content = inlineContent.value
  for (const url of uploadedImages.value) content += "\n\n![](" + url + ")"
  if (editingNoteId.value) {
    await store.updateNote(editingNoteId.value, content.trim(), tags, auth.userName)
    editingNoteId.value = ""
  } else {
    await store.addNote(content.trim(), tags, auth.userName)
  }
  inlineContent.value = ""
  inlineTagsInput.value = []
  uploadedImages.value = []
  showInlineTags.value = false
  clearDraft()
  nextTick(() => {
    const el = document.querySelector('.inline-textarea') as HTMLTextAreaElement
    if (el) { el.style.height = '' }
  })
}

function triggerInlineUpload() { inlineFileInput.value?.click() }

async function onInlineUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  if (files.some(f => f.size > 10 * 1024 * 1024)) { alert("图片大小不能超过 10MB"); input.value = ""; return }
  inlineUploading.value = true
  for (const file of files) {
    const fd = new FormData()
    fd.append("image", file)
    try {
      const res = await authFetch("/api/notes/upload", { method: "POST", body: fd })
      const data = await res.json()
      if (data.success) uploadedImages.value.push(data.url)
      else alert(data.error || "上传失败")
    } catch { alert("上传失败") }
  }
  inlineUploading.value = false
  input.value = ""
}

async function onInlineDrop(e: DragEvent) {
  const files = e.dataTransfer?.files
  if (!files || !files.length) return
  e.preventDefault()
  if (Array.from(files).some(f => f.size > 10 * 1024 * 1024)) { alert("图片大小不能超过 10MB"); return }
  inlineUploading.value = true
  for (const file of Array.from(files)) {
    if (!file.type.startsWith("image/")) continue
    const fd = new FormData()
    fd.append("image", file)
    try {
      const res = await authFetch("/api/notes/upload", { method: "POST", body: fd })
      const data = await res.json()
      if (data.success) uploadedImages.value.push(data.url)
      else alert(data.error || "上传失败")
    } catch { alert("上传失败") }
  }
  inlineUploading.value = false
}

async function onInlinePaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items
  if (!items) return
  for (let i = 0; i < items.length; i++) {
    if (items[i].type.startsWith("image/")) {
      e.preventDefault()
      const file = items[i].getAsFile()
      if (!file) continue
      if (file.size > 10 * 1024 * 1024) { alert("图片大小不能超过 10MB"); return }
      inlineUploading.value = true
      const fd = new FormData()
      fd.append("image", file)
      try {
        const res = await authFetch("/api/notes/upload", { method: "POST", body: fd })
        const data = await res.json()
        if (data.success) uploadedImages.value.push(data.url)
        else alert(data.error || "上传失败")
      } catch { alert("粘贴图片上传失败") }
      inlineUploading.value = false
      return
    }
  }
}

function autoGrowTextarea(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = "auto"
  el.style.height = el.scrollHeight + "px"
}

function handleEdit(memo: any) {
  clearDraft()
  const imgRegex = /!\[.*?\]\((.+?)\)/g
  const urls: string[] = []
  const text = memo.content.replace(imgRegex, (_m: string, url: string) => { urls.push(url); return "" })
  inlineContent.value = text.trim()
  uploadedImages.value = urls
  inlineTagsInput.value = memo.tags || []
  editingNoteId.value = memo.id
  showInlineTags.value = true
  nextTick(() => {
    const el = document.querySelector(".inline-textarea") as HTMLTextAreaElement
    if (el) { el.style.height = "auto"; el.style.height = el.scrollHeight + "px" }
  })
  document.querySelector(".inline-textarea")?.scrollIntoView({ behavior: "smooth" })
  ;(document.querySelector(".inline-textarea") as HTMLTextAreaElement)?.focus()
}

// Drag-and-drop reorder for pinned notes
const dragId = ref("")
const dragOverId = ref("")

function onDragStart(note: any) {
  dragId.value = note.id
}

function onDragOver(e: DragEvent, note: any) {
  if (note.id !== dragId.value && note.pinned) {
    e.preventDefault()
    dragOverId.value = note.id
  }
}

function onDragLeave() {
  dragOverId.value = ""
}

async function onDrop(e: DragEvent, targetNote: any) {
  e.preventDefault()
  dragOverId.value = ""
  if (!dragId.value || dragId.value === targetNote.id) return
  const pinnedNotes = filteredNotes.value.filter(n => n.pinned)
  const fromIdx = pinnedNotes.findIndex(n => n.id === dragId.value)
  const toIdx = pinnedNotes.findIndex(n => n.id === targetNote.id)
  if (fromIdx === -1 || toIdx === -1) return
  pinnedNotes.splice(toIdx, 0, pinnedNotes.splice(fromIdx, 1)[0])
  const order = pinnedNotes.map(n => n.id)
  await authFetch("/api/notes/reorder", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ order }),
  })
  await store.fetchNotes()
}
</script>

<template>
  <div class="notes-layout" :class="{ mobile: isMobile }">
    <div class="side-col">
      <div class="side-content">
        <v-text-field v-model="searchQuery" prepend-inner-icon="mdi-magnify"
          label="搜索备忘..." variant="outlined" hide-details density="compact"
          clearable class="mb-3 rounded-search search-border" />
        <Heatmap class="mb-4" style="border-color:#424242 !important" @select-day="selectedDay = $event" />
        <v-card variant="outlined" class="rounded-xl pa-4 side-card">
          <div class="d-flex align-center ga-2 mb-3">
            <span class="text-subtitle-2 font-weight-medium">标签</span>
          </div>
          <div class="d-flex flex-wrap ga-1">
            <v-chip v-for="[tag, count] in allTags" :key="tag" size="x-small" class="tag-chip"
              @click="selectedTag = selectedTag === tag ? '' : tag"
              color="primary"
              :variant="selectedTag === tag ? 'flat' : 'outlined'">
              #{{ tag }}
            </v-chip>
            <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
          </div>
        </v-card>
        <div v-if="versionText" class="d-flex justify-center mt-2">
          <v-chip size="x-small" variant="tonal" color="primary" class="version-chip" @click="openGithub" style="cursor:pointer">{{ versionText }}</v-chip>
        </div>
      </div>
    </div>

    <div class="main-col">
      <v-dialog :model-value="mobileHeatmap" max-width="400" scrollable persistent transition="dialog-bottom-transition"
        @update:model-value="v => !v && emit('close-heatmap')">
        <v-card class="rounded-xl pa-4 heatmap-dialog-card">
          <div class="d-flex align-center mb-3">
            <span class="text-subtitle-2 font-weight-medium">活动日历</span>
            <v-spacer /><v-btn icon="mdi-close" size="small" variant="text" @click="emit('close-heatmap')" />
          </div>
          <v-text-field v-model="searchQuery" prepend-inner-icon="mdi-magnify" label="搜索备忘..." variant="outlined" hide-details density="compact" clearable class="mb-3 rounded-search search-border" />
          <Heatmap class="mb-4" style="border-color:#424242 !important" @select-day="selectedDay = $event; emit('close-heatmap')" />
          <v-card variant="outlined" class="rounded-xl pa-4 side-card">
            <div class="d-flex align-center ga-2 mb-3"><span class="text-subtitle-2 font-weight-medium">标签</span></div>
            <div class="d-flex flex-wrap ga-1">
              <v-chip v-for="[tag, count] in allTags" :key="tag" size="x-small" class="tag-chip"
                @click="selectedTag = selectedTag === tag ? '' : tag; emit('close-heatmap')"
                color="primary" :variant="selectedTag === tag ? 'flat' : 'outlined'">
                #{{ tag }}
              </v-chip>
              <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
            </div>
          </v-card>
          <div v-if="versionText" class="d-flex justify-center mt-2">
            <v-chip size="x-small" variant="tonal" color="primary" class="version-chip" @click="openGithub" style="cursor:pointer">{{ versionText }}</v-chip>
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
          <div v-if="!deletedNotes.length" class="text-caption text-medium-emphasis py-4 text-center">回收站为空</div>
          <div v-else class="d-flex flex-column ga-2">
            <div v-for="note in deletedNotes" :key="note.id" class="d-flex align-center ga-2 pa-2"
              style="border-bottom:1px solid rgba(var(--v-theme-on-surface),0.06)">
              <div class="flex-grow-1 text-caption" style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                {{ note.content?.replace(/!\[.*?\]\(.+?\)/g, "[图片]").substring(0, 60) }}
              </div>
              <v-btn icon="mdi-restore" size="x-small" variant="text" color="primary" @click="restoreNote(note.id)" title="恢复" />
              <v-btn icon="mdi-delete-forever" size="x-small" variant="text" color="error" @click="deleteForever(note.id)" title="永久删除" />
            </div>
          </div>
        </v-card>
      </v-dialog>

      <div v-if="auth.isLoggedIn" class="inline-editor mb-4">
        <div class="editor-box" @drop.prevent="onInlineDrop" @dragover.prevent>
          <div class="md-toolbar">
            <v-btn icon="mdi-format-bold" size="small" variant="text" class="tool-btn" @click="insertBold" title="粗体 (Ctrl+B)" />
            <v-btn icon="mdi-format-italic" size="small" variant="text" class="tool-btn" @click="insertItalic" title="斜体 (Ctrl+I)" />
            <span class="tool-sep" />
            <v-btn icon="mdi-format-header-pound" size="small" variant="text" class="tool-btn" @click="insertHeading" title="标题" />
            <v-btn icon="mdi-code-tags" size="small" variant="text" class="tool-btn" @click="insertCode" title="代码" />
            <v-btn icon="mdi-link-variant" size="small" variant="text" class="tool-btn" @click="insertLink" title="链接" />
            <span class="tool-sep" />
            <v-btn icon="mdi-format-list-bulleted" size="small" variant="text" class="tool-btn" @click="insertList" title="列表" />
            <v-btn icon="mdi-format-list-numbered" size="small" variant="text" class="tool-btn" @click="insertOrderedList" title="有序列表" />
            <span class="tool-sep" />
            <v-btn icon="mdi-format-quote-open" size="small" variant="text" class="tool-btn" @click="insertQuote" title="引用" />
            <v-btn icon="mdi-format-strikethrough-variant" size="small" variant="text" class="tool-btn" @click="insertStrikethrough" title="删除线" />
            <v-btn icon="mdi-format-list-checks" size="small" variant="text" class="tool-btn" @click="insertTodo" title="待办" />
            <span class="tool-sep" />
            <v-btn icon="mdi-code-braces" size="small" variant="text" class="tool-btn" @click="insertCodeBlock" title="代码块" />
            <v-btn icon="mdi-table" size="small" variant="text" class="tool-btn" @click="insertTable" title="表格" />
            <v-btn icon="mdi-minus" size="small" variant="text" class="tool-btn" @click="insertHr" title="分隔线" />
          </div>
          <textarea ref="inlineTextarea" v-model="inlineContent" class="inline-textarea"
            placeholder="写点什么呢.." rows="1" @keydown="onInlineKeydown" @paste="onInlinePaste" @input="autoGrowTextarea"></textarea>
          <div v-if="uploadedImages.length" class="d-flex flex-wrap ga-2 pa-2 pt-0">
            <div v-for="(img, ii) in uploadedImages" :key="ii" style="position:relative;display:inline-block;width:72px;height:72px;border-radius:8px;overflow:hidden;border:1px solid rgba(var(--v-theme-on-surface),0.08);flex-shrink:0">
              <img :src="img" style="width:100%;height:100%;object-fit:cover;cursor:zoom-in" @click.stop="zoomedUpload = img" />
              <v-btn icon="mdi-close-circle" size="x-small" variant="text"
                style="position:absolute;top:-4px;right:-4px;background:rgb(var(--v-theme-surface));border-radius:50%"
                @click="uploadedImages.splice(ii, 1)" />
            </div>
          </div>
          <div class="editor-toolbar">
            <div class="d-flex align-center ga-2">
              <v-btn icon="mdi-image-plus" size="small" variant="text" class="tool-btn" :loading="inlineUploading" @click="triggerInlineUpload" />
              <input ref="inlineFileInput" type="file" accept="image/*" multiple hidden @change="onInlineUpload" />
              <span class="tool-sep-sm" />
              <v-btn :icon="showInlineTags ? 'mdi-tag-off' : 'mdi-tag-outline'" size="small" variant="text" class="tool-btn" @click="showInlineTags = !showInlineTags" />
              <v-btn icon="mdi-delete-outline" size="small" variant="text" class="tool-btn"
                @click="showTrash = !showTrash; if(showTrash) fetchDeletedNotes()" />
            </div>
            <v-btn color="primary" size="small" variant="flat" class="rounded-pill submit-btn" @click="submitInline">
              <v-icon start>mdi-send</v-icon>{{ editingNoteId ? "更新" : "发布" }}
            </v-btn>
          </div>
          <v-expand-transition>
            <div v-if="showInlineTags" class="pa-3">
              <v-combobox v-model="inlineTagsInput" :items="allTags.map(t => t[0])" label="标签" variant="outlined" hide-details density="compact" placeholder="vue, memos, md" multiple chips closable-chips small-chips clearable />
            </div>
          </v-expand-transition>
        </div>
        <div v-if="hasDraft && !editingNoteId" class="draft-indicator">
          <v-icon size="x-small" color="warning">mdi-content-save</v-icon>
          <span>草稿已自动保存</span>
        </div>
      </div>

      <div v-if="!store.loaded" class="d-flex justify-center py-16">
        <v-progress-circular indeterminate color="primary" />
      </div>
      <template v-else>
        <div v-if="selectedDay" class="date-filter-bar">
          <v-icon size="x-small" color="primary">mdi-calendar</v-icon>
          <span>{{ selectedDay }} 的备忘录</span>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="selectedDay = ''" />
        </div>
        <div v-if="filteredNotes.length === 0" class="empty-state">
          <div class="empty-icon-wrap">
            <v-icon size="48" color="rgba(var(--v-theme-on-surface),0.12)">mdi-pencil-box-multiple-outline</v-icon>
          </div>
          <p v-if="searchQuery || selectedTag || selectedDay" class="text-body-1 font-weight-medium mb-1">没有找到匹配的备忘录</p>
          <p v-else class="text-body-1 font-weight-medium mb-1">还没有备忘录</p>
          <p v-if="!searchQuery && !selectedTag && !selectedDay" class="text-caption text-medium-emphasis">点击上方编辑框，写下你的第一段记忆吧 ✨</p>
        </div>
        <div class="d-flex flex-column ga-4">
          <div v-for="note in filteredNotes" :key="note.id"
            :draggable="note.pinned && auth.isLoggedIn"
            @dragstart="onDragStart(note)"
            @dragover="onDragOver($event, note)"
            @dragleave="onDragLeave"
            @drop="onDrop($event, note)"
            :class="['note-drag-wrapper', { 'drag-over': dragOverId === note.id, 'dragging': dragId === note.id }]">
            <NoteCard :memo="note" :search-query="searchQuery" :logged-in="auth.isLoggedIn" @edit="handleEdit" />
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
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <img :src="zoomedUpload" class="zoom-img" @click.stop />
    </div>
  </teleport>
</template>

<style scoped>
.notes-layout { display: flex; gap: 28px; padding: 28px; max-width: 1200px; margin: 0 auto; align-items: flex-start; }
.md-toolbar { overflow-x: auto !important; overflow-y: hidden; white-space: nowrap !important; -webkit-overflow-scrolling: touch; scrollbar-width: thin; }
.md-toolbar::-webkit-scrollbar { height: 3px; }
.md-toolbar::-webkit-scrollbar-thumb { background: rgba(var(--v-theme-on-surface), 0.15); border-radius: 3px; }
.notes-layout.mobile { flex-direction: column; padding: 12px; gap: 12px; }
.side-col { width: 280px; flex-shrink: 0; position: sticky; top: 24px; align-self: flex-start; }
.notes-layout.mobile .side-col { display: none; }
.main-col { flex: 1; min-width: 0; }
.tag-chip { cursor: pointer; }
.tag-chip:hover { opacity: 0.9; }
.rounded-search :deep(.v-field) { border-radius: 12px !important; }
.heatmap-dialog-card { border-color: #424242 !important; }
.icp-link { color: inherit; text-decoration: none; }
.icp-link:hover { text-decoration: underline; }

.inline-editor { width: 100%; }
.editor-box {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 14px; overflow: clip;
  transition: border-color 0.2s, box-shadow 0.2s;
  background: rgb(var(--v-theme-surface));
}
.editor-box:focus-within {
  border-color: rgba(var(--v-theme-primary), 0.3);
  box-shadow: 0 2px 16px rgba(var(--v-theme-primary), 0.08);
}
.inline-textarea {
  width: 100%; border: none; outline: none; resize: none;
  padding: 14px 16px 8px; font-size: 0.95rem; line-height: 1.6;
  font-family: inherit; background: transparent;
  color: rgb(var(--v-theme-on-surface)); min-height: 80px;
}
.inline-textarea::placeholder { color: rgba(var(--v-theme-on-surface), 0.35); }
.editor-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 4px 8px 8px;
}
.editor-toolbar .tool-btn { opacity: 0.5; border-radius: 6px; }
.editor-toolbar .tool-btn:hover { opacity: 1; background: rgba(var(--v-theme-on-surface), 0.05); }
.submit-btn { height: 30px; }
.md-toolbar .tool-btn { width: 34px; height: 34px; opacity: 0.5; border-radius: 6px; flex-shrink: 0; }
.md-toolbar .tool-btn:hover { opacity: 1; background: rgba(var(--v-theme-on-surface), 0.05); }
.tool-sep { width: 1px; height: 22px; background: rgba(var(--v-theme-on-surface), 0.1); flex-shrink: 0; margin: 0 3px; }
.search-border :deep(.v-field) { border-color: #424242 !important; }
.side-card { border-color: #424242 !important; }
.draft-indicator {
  display: flex; align-items: center; gap: 4px;
  padding: 2px 12px 8px 12px;
  font-size: 0.7rem; color: rgba(var(--v-theme-warning), 0.7);
}
.empty-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 48px 16px; gap: 8px;
}
.empty-icon-wrap {
  width: 80px; height: 80px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 50%;
  background: rgba(var(--v-theme-on-surface), 0.03);
  margin-bottom: 8px;
}
.tool-sep {
  width: 1px; height: 20px;
  background: rgba(var(--v-theme-on-surface), 0.1);
  flex-shrink: 0;
}
.tool-sep-sm {
  width: 1px; height: 16px;
  background: rgba(var(--v-theme-on-surface), 0.08);
  flex-shrink: 0;
}
.date-filter-bar {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 12px;
  margin-bottom: 8px;
  background: rgba(var(--v-theme-primary), 0.06);
  border: 1px solid rgba(var(--v-theme-primary), 0.15);
  border-radius: 8px;
  font-size: 0.8rem;
  color: rgb(var(--v-theme-primary));
}
.note-drag-wrapper {
  transition: opacity 0.15s, box-shadow 0.15s;
  border-radius: 12px;
}
.note-drag-wrapper.dragging {
  opacity: 0.4;
}
.note-drag-wrapper.drag-over {
  box-shadow: 0 0 0 2px rgb(var(--v-theme-primary));
}

@media (max-width: 768px) {
  .notes-layout.mobile { flex-direction: column; padding: 12px; gap: 8px; }
  .notes-layout.mobile .main-col { width: 100%; }
  .notes-layout.mobile .inline-textarea { min-height: 60px; padding: 12px 14px 8px; font-size: 0.9rem; }
  .editor-toolbar .tool-btn { width: 28px; height: 28px; }
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






