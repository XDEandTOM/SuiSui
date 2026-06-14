<script setup lang="ts">
import { ref, computed, nextTick, watch } from "vue"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
import { authFetch } from "@/utils/api"

const store = useNotesStore()
const auth = useAuthStore()

const emit = defineEmits<{ submitted: [] }>()

const inlineContent = ref("")
const inlineTagsInput = ref<string[]>([])
const showInlineTags = ref(false)
const tagInput = ref("")
const inlineUploading = ref(false)
const inlineTextarea = ref<HTMLTextAreaElement | null>(null)
const inlineFileInput = ref<HTMLInputElement | null>(null)
const uploadedImages = ref<string[]>([])
const editingNoteId = ref("")
const hasDraft = computed(() => !!(inlineContent.value || uploadedImages.value.length))
const DRAFT_KEY = "suisui-draft"

function addTag() {
  const t = tagInput.value.trim()
  if (t && !inlineTagsInput.value.includes(t)) { inlineTagsInput.value.push(t) }
  tagInput.value = ""
}

function saveDraft() {
  if (!auth.isLoggedIn) return
  const draft = { content: inlineContent.value, tags: inlineTagsInput.value, images: uploadedImages.value, editingId: editingNoteId.value }
  try { localStorage.setItem(DRAFT_KEY, JSON.stringify(draft)) } catch { console.warn("saveDraft failed") }
}

function restoreDraft() {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) return
    const draft: { content?: string; tags?: string | string[]; images?: string[]; editingId?: string } = JSON.parse(raw)
    if (draft.content) inlineContent.value = draft.content
    if (draft.tags) { inlineTagsInput.value = typeof draft.tags === "string" ? draft.tags.split(/[,，]/).map(t => t.trim()).filter(Boolean) : draft.tags; showInlineTags.value = true }
    if (draft.images?.length) uploadedImages.value = draft.images
    if (draft.editingId) editingNoteId.value = draft.editingId
  } catch { console.warn("restoreDraft failed") }
}

function clearDraft() { localStorage.removeItem(DRAFT_KEY) }

let draftTimer: ReturnType<typeof setTimeout> | null = null
watch([inlineContent, inlineTagsInput, uploadedImages, editingNoteId], () => {
  if (draftTimer) clearTimeout(draftTimer)
  draftTimer = setTimeout(saveDraft, 500)
}, { deep: true })

function insertMd(b: string, f: string, fb: string) {
  const el = inlineTextarea.value
  if (!el) { inlineContent.value += fb; return }
  const start = el.selectionStart, end = el.selectionEnd
  const t = inlineContent.value, sel = t.substring(start, end)
  inlineContent.value = t.slice(0,start) + b + (sel||fb) + f + t.slice(end)
  nextTick(() => { el.focus(); el.selectionStart = el.selectionEnd = start + b.length + (sel||fb).length })
}
function insertBold() { insertMd("**","**","") }
function insertItalic() { insertMd("*","*","") }
function insertHeading() { insertMd("\n## ","","") }
function insertCode() { insertMd("`","`","") }
function insertLink() { insertMd("[","](url)","") }
function insertList() { insertMd("\n- ","","") }
function insertOrderedList() { insertMd("\n1. ","","") }
function insertQuote() { insertMd("\n> ","","") }
function insertStrikethrough() { insertMd("~~","~~","") }
function insertHr() { insertMd("\n---\n","","") }
function insertTable() { insertMd("\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n| 内容 | 内容 | 内容 |","","") }
function insertTodo() { insertMd("\n- [ ] ","","") }
function insertCodeBlock() { insertMd("\n```\n","\n```\n","") }

function onInlineKeydown(e: KeyboardEvent) {
  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { submitInline(); return }
  if (e.ctrlKey || e.metaKey) {
    if (e.key === "b") { e.preventDefault(); insertBold(); return }
    if (e.key === "i") { e.preventDefault(); insertItalic(); return }
  }
}

async function submitInline() {
  if ((!inlineContent.value.trim() && !uploadedImages.value.length) || !auth.isLoggedIn) return
  const tags = Array.isArray(inlineTagsInput.value) ? inlineTagsInput.value.map(t => t.trim()).filter(Boolean) : []
  let content = inlineContent.value
  for (const url of uploadedImages.value) content += "\n\n![](" + url + ")"
  if (editingNoteId.value) { await store.updateNote(editingNoteId.value, content.trim(), tags, auth.userName); editingNoteId.value = "" }
  else { await store.addNote(content.trim(), tags, auth.userName) }
  emit("submitted")
  inlineContent.value = ""; inlineTagsInput.value = []; uploadedImages.value = []; showInlineTags.value = false; clearDraft()
  nextTick(() => { if (inlineTextarea.value) inlineTextarea.value.style.height = '' })
}

function triggerInlineUpload() { inlineFileInput.value?.click() }

async function onInlineUpload(e: Event) {
  const input = e.target as HTMLInputElement; const files = Array.from(input.files || [])
  if (!files.length) return
  if (files.some(f => f.size > 10 * 1024 * 1024)) { alert("图片大小不能超过 10MB"); input.value = ""; return }
  inlineUploading.value = true
  for (const file of files) {
    const fd = new FormData(); fd.append("image", file)
    try { const res = await authFetch("/api/notes/upload", { method: "POST", body: fd }); const data = await res.json(); if (data.success) uploadedImages.value.push(data.url); else alert(data.error || "上传失败") }
    catch { alert("上传失败") }
  }
  inlineUploading.value = false; input.value = ""
}

async function onInlineDrop(e: DragEvent) {
  const files = e.dataTransfer?.files; if (!files?.length) return; e.preventDefault()
  if (Array.from(files).some(f => f.size > 10 * 1024 * 1024)) { alert("图片大小不能超过 10MB"); return }
  inlineUploading.value = true
  for (const file of Array.from(files)) {
    if (!file.type.startsWith("image/")) continue
    const fd = new FormData(); fd.append("image", file)
    try { const res = await authFetch("/api/notes/upload", { method: "POST", body: fd }); const data = await res.json(); if (data.success) uploadedImages.value.push(data.url) }
    catch { alert("上传失败") }
  }
  inlineUploading.value = false
}

async function onInlinePaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items; if (!items) return
  for (let i = 0; i < items.length; i++) {
    if (items[i].type.startsWith("image/")) {
      e.preventDefault(); const file = items[i].getAsFile(); if (!file) continue
      if (file.size > 10 * 1024 * 1024) { alert("图片大小不能超过 10MB"); return }
      inlineUploading.value = true; const fd = new FormData(); fd.append("image", file)
      try { const res = await authFetch("/api/notes/upload", { method: "POST", body: fd }); const data = await res.json(); if (data.success) uploadedImages.value.push(data.url) }
      catch { alert("粘贴图片上传失败") }
      inlineUploading.value = false; return
    }
  }
}

function autoGrowTextarea(e: Event) {
  const el = e.target as HTMLTextAreaElement; el.style.height = "auto"; el.style.height = el.scrollHeight + "px"
}

function handleEdit(memo: { id: string; content: string; tags?: string[] }) {
  clearDraft()
  const imgRegex = /!\[.*?\]\((.+?)\)/g; const urls: string[] = []
  const text = memo.content.replace(imgRegex, (_m: string, url: string) => { urls.push(url); return "" })
  inlineContent.value = text.trim(); uploadedImages.value = urls; inlineTagsInput.value = memo.tags || []; editingNoteId.value = memo.id; showInlineTags.value = true
  nextTick(() => { const el = inlineTextarea.value; if (el) { el.style.height = "auto"; el.style.height = el.scrollHeight + "px" } })
  inlineTextarea.value?.scrollIntoView({ behavior: "smooth" }); inlineTextarea.value?.focus()
}

const zoomedUpload = ref("")

restoreDraft()

defineExpose({ handleEdit })
</script>

<template>
  <div v-if="auth.isLoggedIn" class="inline-editor mb-4">
    <div class="editor-box" @drop.prevent="onInlineDrop" @dragover.prevent>
      <div class="md-toolbar">
        <v-btn icon="mdi-format-bold" size="small" variant="text" class="tool-btn" title="粗体 (Ctrl+B)" @click="insertBold" />
        <v-btn icon="mdi-format-italic" size="small" variant="text" class="tool-btn" title="斜体 (Ctrl+I)" @click="insertItalic" />
        <span class="tool-sep" />
        <v-btn icon="mdi-format-header-pound" size="small" variant="text" class="tool-btn" title="标题" @click="insertHeading" />
        <v-btn icon="mdi-code-tags" size="small" variant="text" class="tool-btn" title="代码" @click="insertCode" />
        <v-btn icon="mdi-link-variant" size="small" variant="text" class="tool-btn" title="链接" @click="insertLink" />
        <span class="tool-sep" />
        <v-btn icon="mdi-format-list-bulleted" size="small" variant="text" class="tool-btn" title="列表" @click="insertList" />
        <v-btn icon="mdi-format-list-numbered" size="small" variant="text" class="tool-btn" title="有序列表" @click="insertOrderedList" />
        <span class="tool-sep" />
        <v-btn icon="mdi-format-quote-open" size="small" variant="text" class="tool-btn" title="引用" @click="insertQuote" />
        <v-btn icon="mdi-format-strikethrough-variant" size="small" variant="text" class="tool-btn" title="删除线" @click="insertStrikethrough" />
        <v-btn icon="mdi-format-list-checks" size="small" variant="text" class="tool-btn" title="待办" @click="insertTodo" />
        <span class="tool-sep" />
        <v-btn icon="mdi-code-braces" size="small" variant="text" class="tool-btn" title="代码块" @click="insertCodeBlock" />
        <v-btn icon="mdi-table" size="small" variant="text" class="tool-btn" title="表格" @click="insertTable" />
        <v-btn icon="mdi-minus" size="small" variant="text" class="tool-btn" title="分隔线" @click="insertHr" />
      </div>
      <textarea ref="inlineTextarea" v-model="inlineContent" class="inline-textarea"
        placeholder="写点什么呢.." rows="1" @keydown="onInlineKeydown" @paste="onInlinePaste" @input="autoGrowTextarea"></textarea>
      <div v-if="uploadedImages.length" class="d-flex flex-wrap ga-2 pa-2 pt-0">
        <div v-for="(img, ii) in uploadedImages" :key="ii" class="uploaded-img-wrap">
          <img :src="img" style="width:100%;height:100%;object-fit:cover;cursor:zoom-in" @click.stop="zoomedUpload = img" />
          <v-btn icon="mdi-close-circle" size="x-small" variant="text" class="img-close-btn" @click="uploadedImages.splice(ii, 1)" />
        </div>
      </div>
      <div class="editor-toolbar">
        <div class="d-flex align-center ga-2">
          <v-btn icon="mdi-image-plus" size="small" variant="text" class="tool-btn" :loading="inlineUploading" @click="triggerInlineUpload" />
          <input ref="inlineFileInput" type="file" accept="image/*" multiple hidden @change="onInlineUpload" />
          <span class="tool-sep-sm" />
        </div>
        <v-btn color="primary" size="small" variant="flat" class="rounded-pill submit-btn" @click="submitInline">
          <v-icon start>mdi-send</v-icon>{{ editingNoteId ? "更新" : "发布" }}
        </v-btn>
      </div>
      <div class="inline-tag-bar">
        <template v-for="(tag, i) in inlineTagsInput" :key="i">
          <v-chip size="x-small" closable @click:close="inlineTagsInput.splice(i, 1)">{{ tag }}</v-chip>
        </template>
        <v-text-field v-model="tagInput" variant="plain" hide-details density="compact" placeholder="+ 添加标签" single-line class="tag-input" @keydown.enter.prevent="addTag" />
      </div>
    </div>
    <div v-if="hasDraft && !editingNoteId" class="draft-indicator">
      <v-icon size="x-small" color="warning">mdi-content-save</v-icon><span>草稿已自动保存</span>
    </div>
  </div>
  <teleport to="body">
    <div v-if="zoomedUpload" class="zoom-overlay" @click="zoomedUpload = ''">
      <button class="zoom-close-btn" @click.stop="zoomedUpload = ''"><svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg></button>
      <img :src="zoomedUpload" class="zoom-img" @click.stop />
    </div>
  </teleport>
</template>

<style scoped>
.inline-editor { width: 100%; }
.editor-box {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 14px; overflow: clip;
  transition: border-color 0.2s, box-shadow 0.2s;
  background: rgba(var(--v-theme-surface), 0.55);
  backdrop-filter: blur(8px);
}
.editor-box:focus-within { border-color: rgba(var(--v-theme-primary), 0.3); box-shadow: 0 2px 16px rgba(var(--v-theme-primary), 0.08); }
.inline-textarea {
  width: 100%; border: none; outline: none; resize: none;
  padding: 14px 16px 8px; font-size: 0.95rem; line-height: 1.6;
  font-family: inherit; background: transparent;
  color: rgb(var(--v-theme-on-surface)); min-height: 80px;
}
.inline-textarea::placeholder { color: rgba(var(--v-theme-on-surface), 0.35); }
.md-toolbar { overflow-x: auto !important; white-space: nowrap !important; }
.md-toolbar .tool-btn { width: 34px; height: 34px; opacity: 0.5; border-radius: 6px; flex-shrink: 0; }
.md-toolbar .tool-btn:hover { opacity: 1; background: rgba(var(--v-theme-on-surface), 0.05); }
.editor-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 4px 8px 8px; }
.editor-toolbar .tool-btn { opacity: 0.5; border-radius: 6px; }
.editor-toolbar .tool-btn:hover { opacity: 1; background: rgba(var(--v-theme-on-surface), 0.05); }
.submit-btn { height: 30px; }
.inline-tag-bar { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; padding: 0 12px 8px; }
.inline-tag-bar .tag-input { min-width: 100px; max-width: 160px; }
.draft-indicator { display: flex; align-items: center; gap: 4px; padding: 2px 12px 8px; font-size: 0.7rem; color: rgba(var(--v-theme-warning), 0.7); }
.uploaded-img-wrap { position: relative; display: inline-block; width: 72px; height: 72px; border-radius: 8px; overflow: hidden; border: 1px solid rgba(var(--v-theme-on-surface), 0.08); flex-shrink: 0; }
.img-close-btn { position: absolute; top: -4px; right: -4px; background: rgb(var(--v-theme-surface)); border-radius: 50%; }
.tool-sep { width: 1px; height: 20px; background: rgba(var(--v-theme-on-surface), 0.1); flex-shrink: 0; display: inline-block; vertical-align: middle; }
.tool-sep-sm { width: 1px; height: 16px; background: rgba(var(--v-theme-on-surface), 0.08); flex-shrink: 0; }
@media (max-width: 768px) { .inline-textarea { min-height: 60px; padding: 12px 14px 8px; font-size: 0.9rem; } .editor-toolbar .tool-btn { width: 28px; height: 28px; } }
</style>
<style>
.zoom-overlay { position: fixed; inset: 0; z-index: 9999; background: rgba(0,0,0,0.8); display: flex; align-items: center; justify-content: center; cursor: zoom-out; animation: zoomFadeIn 0.25s ease; }
.zoom-img { max-width: 90vw; max-height: 90vh; border-radius: 8px; object-fit: contain; cursor: default; }
.zoom-close-btn { position: fixed; top: 16px; right: 16px; width: 36px; height: 36px; border-radius: 50%; border: none; background: rgba(255,255,255,0.15); color: #fff; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: background 0.2s; z-index: 10000; }
.zoom-close-btn:hover { background: rgba(255,255,255,0.3); }
.zoom-img-wrap { display: flex; align-items: center; justify-content: center; }
@keyframes zoomFadeIn { from { opacity: 0 } to { opacity: 1 } }
</style>
