<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from "vue"
import { useDisplay } from "vuetify"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
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

const siteIcp = ref("")
onMounted(async () => { await store.fetchNotes(); await loadSiteIcp() })

async function loadSiteIcp() {
  try {
    const r = await fetch("/api/settings")
    if (r.ok) {
      const s = await r.json()
      siteIcp.value = s.site_icp || ""
    }
  } catch { /* ignore */ }
}

const allTags = computed(() => {
  const tagCount = new Map()
  for (const n of store.notes) {
    for (const t of n.tags) tagCount.set(t, (tagCount.get(t) || 0) + 1)
  }
  return [...tagCount.entries()].sort((a, b) => b[1] - a[1])
})

const filteredNotes = computed(() => {
  let list = store.notes
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(n => n.content.toLowerCase().includes(q) || n.tags.some(t => t.toLowerCase().includes(q)))
  }
  if (selectedTag.value) list = list.filter(n => n.tags.includes(selectedTag.value))
  return list
})

const inlineContent = ref("")
const inlineTagsInput = ref("")
const showInlineTags = ref(false)
const inlineUploading = ref(false)
const inlineTextarea = ref<HTMLTextAreaElement | null>(null)
const inlineFileInput = ref<HTMLInputElement | null>(null)
const uploadedImages = ref<string[]>([])
const editingNoteId = ref("")
const zoomedUpload = ref("")

function onInlineKeydown(e: KeyboardEvent) {
  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) submitInline()
}

async function submitInline() {
  if ((!inlineContent.value.trim() && !uploadedImages.value.length) || !auth.isLoggedIn) return
  const tags = inlineTagsInput.value.split(/[,，]/).map(t => t.trim()).filter(Boolean)
  let content = inlineContent.value
  for (const url of uploadedImages.value) content += "\n\n![](" + url + ")"
  if (editingNoteId.value) {
    await store.updateNote(editingNoteId.value, content.trim(), tags)
    editingNoteId.value = ""
  } else {
    await store.addNote(content.trim(), tags, auth.userName)
  }
  inlineContent.value = ""
  inlineTagsInput.value = ""
  uploadedImages.value = []
  showInlineTags.value = false
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
      const res = await fetch("/api/notes/upload", { method: "POST", body: fd })
      const data = await res.json()
      if (data.success) uploadedImages.value.push(data.url)
      else alert(data.error || "上传失败")
    } catch { alert("上传失败") }
  }
  inlineUploading.value = false
  input.value = ""
}

function autoGrowTextarea(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = "auto"
  el.style.height = el.scrollHeight + "px"
}

function handleEdit(memo: any) {
  const imgRegex = /!\[.*?\]\((.+?)\)/g
  const urls: string[] = []
  const text = memo.content.replace(imgRegex, (_m: string, url: string) => { urls.push(url); return "" })
  inlineContent.value = text.trim()
  uploadedImages.value = urls
  editingNoteId.value = memo.id
  showInlineTags.value = false
  nextTick(() => {
    const el = document.querySelector(".inline-textarea") as HTMLTextAreaElement
    if (el) { el.style.height = "auto"; el.style.height = el.scrollHeight + "px" }
  })
  document.querySelector(".inline-textarea")?.scrollIntoView({ behavior: "smooth" })
  ;(document.querySelector(".inline-textarea") as HTMLTextAreaElement)?.focus()
}
</script>

<template>
  <div class="notes-layout" :class="{ mobile: isMobile }">
    <div class="side-col">
      <div class="side-content">
        <v-text-field v-model="searchQuery" prepend-inner-icon="mdi-magnify"
          label="搜索备忘..." variant="outlined" hide-details density="compact"
          clearable class="mb-3 rounded-search search-border" />
        <Heatmap class="mb-4" style="border-color:#424242 !important" />
        <v-card variant="outlined" class="rounded-xl pa-4 side-card">
          <div class="d-flex align-center ga-2 mb-3">
            <span class="text-subtitle-2 font-weight-medium">标签</span>
          </div>
          <div class="d-flex flex-wrap ga-1">
            <v-chip v-for="[tag, count] in allTags" :key="tag" size="x-small" class="tag-chip"
              @click="selectedTag = selectedTag === tag ? '' : tag"
              :color="selectedTag === tag ? 'primary' : undefined"
              :variant="selectedTag === tag ? 'flat' : 'outlined'">
              {{ tag }}
              <template #append><span class="text-caption opacity-75">{{ count }}</span></template>
            </v-chip>
            <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
          </div>
        </v-card>
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
          <Heatmap class="mb-4" style="border-color:#424242 !important" />
          <v-card variant="outlined" class="rounded-xl pa-4 side-card">
            <div class="d-flex align-center ga-2 mb-3"><span class="text-subtitle-2 font-weight-medium">标签</span></div>
            <div class="d-flex flex-wrap ga-1">
              <v-chip v-for="[tag, count] in allTags" :key="tag" size="x-small" class="tag-chip"
                @click="selectedTag = selectedTag === tag ? '' : tag; emit('close-heatmap')"
                :color="selectedTag === tag ? 'primary' : undefined" :variant="selectedTag === tag ? 'flat' : 'outlined'">
                {{ tag }}<template #append><span class="text-caption opacity-75">{{ count }}</span></template>
              </v-chip>
              <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
            </div>
          </v-card>
        </v-card>
      </v-dialog>

      <div v-if="auth.isLoggedIn" class="inline-editor mb-4">
        <div class="editor-box">
          <textarea ref="inlineTextarea" v-model="inlineContent" class="inline-textarea"
            placeholder="写点什么呢.." rows="1" @keydown="onInlineKeydown" @input="autoGrowTextarea"></textarea>
          <div v-if="uploadedImages.length" class="d-flex flex-wrap ga-2 pa-2 pt-0">
            <div v-for="(img, ii) in uploadedImages" :key="ii" style="position:relative;display:inline-block;width:72px;height:72px;border-radius:8px;overflow:hidden;border:1px solid rgba(var(--v-theme-on-surface),0.08);flex-shrink:0">
              <img :src="img" style="width:100%;height:100%;object-fit:cover;cursor:zoom-in" @click.stop="zoomedUpload = img" />
              <v-btn icon="mdi-close-circle" size="x-small" variant="text"
                style="position:absolute;top:-4px;right:-4px;background:rgb(var(--v-theme-surface));border-radius:50%"
                @click="uploadedImages.splice(ii, 1)" />
            </div>
          </div>
          <div class="editor-toolbar">
            <div class="d-flex align-center ga-1">
              <v-btn icon="mdi-image-plus" size="x-small" variant="text" class="tool-btn" :loading="inlineUploading" @click="triggerInlineUpload" />
              <input ref="inlineFileInput" type="file" accept="image/*" multiple hidden @change="onInlineUpload" />
              <v-btn :icon="showInlineTags ? 'mdi-tag-off' : 'mdi-tag-outline'" size="x-small" variant="text" class="tool-btn" @click="showInlineTags = !showInlineTags" />
            </div>
            <v-btn color="#1976D2" size="small" variant="flat" class="rounded-pill px-4 submit-btn" @click="submitInline">
              <v-icon start size="x-small">mdi-send</v-icon>{{ editingNoteId ? "更新" : "发布" }}
            </v-btn>
          </div>
          <v-expand-transition>
            <div v-if="showInlineTags" class="pa-3">
              <v-text-field v-model="inlineTagsInput" label="标签（逗号分隔）" variant="outlined" hide-details density="compact" placeholder="vue, memos, md" />
            </div>
          </v-expand-transition>
        </div>
      </div>

      <div v-if="!store.loaded" class="d-flex justify-center py-16">
        <v-progress-circular indeterminate color="primary" />
      </div>
      <template v-else>
        <div v-if="filteredNotes.length === 0" class="d-flex flex-column align-center justify-center py-16 text-medium-emphasis">
          <p v-if="searchQuery || selectedTag" class="text-body-1 mb-1 font-weight-medium">没有找到匹配的备忘录</p>
          <p v-else class="text-body-1 mb-1 font-weight-medium">还没有备忘录</p>
        </div>
        <div class="d-flex flex-column ga-3">
          <NoteCard v-for="note in filteredNotes" :key="note.id" :memo="note" :logged-in="auth.isLoggedIn" @edit="handleEdit" />
        </div>
            </template>
      <div v-if="siteIcp" class="text-center text-caption py-4 icp-text" style="opacity:0.6">
        {{ siteIcp }}
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
.notes-layout { display: flex; gap: 24px; padding: 24px; max-width: 1200px; margin: 0 auto; align-items: flex-start; }
.notes-layout.mobile { flex-direction: column; padding: 12px; gap: 12px; }
.side-col { width: 280px; flex-shrink: 0; position: sticky; top: 24px; align-self: flex-start; }
.notes-layout.mobile .side-col { display: none; }
.main-col { flex: 1; min-width: 0; }
.tag-chip { cursor: pointer; }
.tag-chip:hover { opacity: 0.9; }
.rounded-search :deep(.v-field) { border-radius: 12px !important; }
.heatmap-dialog-card { border-color: #424242 !important; }

.inline-editor { width: 100%; }
.editor-box {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 12px; overflow: hidden;
  transition: border-color 0.2s, box-shadow 0.2s;
  background: rgb(var(--v-theme-surface));
}
.editor-box:focus-within {
  border-color: rgba(var(--v-theme-primary), 0.3);
  box-shadow: 0 2px 12px rgba(var(--v-theme-primary), 0.06);
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
.tool-btn { opacity: 0.5; transition: opacity 0.2s; }
.tool-btn:hover { opacity: 1; }
.submit-btn { height: 30px; }
.search-border :deep(.v-field) { border-color: #424242 !important; }
.side-card { border-color: #424242 !important; }

@media (max-width: 768px) {
  .notes-layout.mobile { flex-direction: column; padding: 12px; gap: 8px; }
  .notes-layout.mobile .main-col { width: 100%; }
  .notes-layout.mobile .inline-textarea { min-height: 60px; padding: 12px 14px 8px; font-size: 0.9rem; }
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






