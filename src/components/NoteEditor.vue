<script setup lang="ts">
import { ref, nextTick } from "vue"
import { useAuthStore } from "@/stores/auth"

const auth = useAuthStore()

const props = defineProps<{
  modelValue: string
  tagsInput: string
  showTags: boolean
  uploading: boolean
  uploadedImages: string[]
  editingNoteId: string
}>()

const emit = defineEmits<{
  "update:modelValue": [val: string]
  "update:tagsInput": [val: string]
  "update:showTags": [val: boolean]
  "update:uploadedImages": [val: string[]]
  submit: []
  "trigger-upload": []
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

function insertMd(b: string, f: string, fb: string) {
  const el = document.querySelector(".inline-textarea") as HTMLTextAreaElement
  if (!el) { emit("update:modelValue", props.modelValue + fb); return }
  const start = el.selectionStart, end = el.selectionEnd
  const t = props.modelValue
  const sel = t.substring(start, end)
  emit("update:modelValue", t.slice(0, start) + b + (sel || fb) + f + t.slice(end))
  nextTick(() => { el.focus(); el.selectionStart = el.selectionEnd = start + b.length + (sel || fb).length })
}
function insertBold() { insertMd("**", "**", "粗体") }
function insertItalic() { insertMd("*", "*", "斜体") }
function insertHeading() { insertMd("\n## ", "", "标题") }
function insertCode() { insertMd("`", "`", "code") }
function insertLink() { insertMd("[", "](url)", "链接文字") }
function insertList() { insertMd("\n- ", "", "列表项") }
function insertQuote() { insertMd("\n> ", "", "引用") }

function onKeydown(e: KeyboardEvent) {
  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) emit("submit")
}

function autoGrow(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = "auto"
  el.style.height = el.scrollHeight + "px"
}

async function onUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  if (files.some(f => f.size > 10 * 1024 * 1024)) { alert("图片大小不能超过 10MB"); input.value = ""; return }
  const uploading = ref(false)
  uploading.value = true
  for (const file of files) {
    const fd = new FormData()
    fd.append("image", file)
    try {
      const res = await fetch("/api/notes/upload", { method: "POST", body: fd })
      const data = await res.json()
      if (data.success) {
        const imgs = [...props.uploadedImages, data.url]
        emit("update:uploadedImages", imgs)
      } else alert(data.error || "上传失败")
    } catch { alert("上传失败") }
  }
  uploading.value = false
  input.value = ""
}

function removeUploadedImg(idx: number) {
  const imgs = props.uploadedImages.filter((_, i) => i !== idx)
  emit("update:uploadedImages", imgs)
}
</script>

<template>
  <div class="inline-editor">
    <div class="editor-box" :class="{ 'has-edit': !!editingNoteId }">
      <div class="md-toolbar">
        <v-btn icon="mdi-format-bold" size="x-small" variant="text" class="tool-btn" @click="insertBold" />
        <v-btn icon="mdi-format-italic" size="x-small" variant="text" class="tool-btn" @click="insertItalic" />
        <v-btn icon="mdi-format-header-2" size="x-small" variant="text" class="tool-btn" @click="insertHeading" />
        <v-btn icon="mdi-code-tags" size="x-small" variant="text" class="tool-btn" @click="insertCode" />
        <v-btn icon="mdi-link-variant" size="x-small" variant="text" class="tool-btn" @click="insertLink" />
        <v-btn icon="mdi-format-list-bulleted" size="x-small" variant="text" class="tool-btn" @click="insertList" />
        <v-btn icon="mdi-format-quote-close" size="x-small" variant="text" class="tool-btn" @click="insertQuote" />
      </div>
      <textarea
        ref="textareaRef"
        v-bind="$attrs"
        :value="modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
        class="inline-textarea"
        :placeholder="editingNoteId ? '编辑备忘录...' : '写点什么...'"
        @keydown="onKeydown"
        @input.capture="autoGrow"
      />
      <div v-if="uploadedImages.length" class="d-flex flex-wrap ga-2 pa-2 pt-0">
        <div v-for="(url, idx) in uploadedImages" :key="idx" class="upload-preview" @click="() => {}">
          <img :src="url" alt="" class="upload-thumb" />
          <button class="upload-remove" @click="removeUploadedImg(idx)">&times;</button>
        </div>
      </div>
      <div class="editor-toolbar">
        <div class="d-flex align-center ga-1">
          <v-btn icon="mdi-image-plus" size="x-small" variant="text" class="tool-btn" :loading="uploading" @click="$emit('trigger-upload')" />
          <v-btn icon="mdi-tag-plus" size="x-small" variant="text" class="tool-btn" @click="emit('update:showTags', !showTags)" />
          <input ref="fileInputRef" type="file" accept="image/*" multiple hidden @change="onUpload" />
          <v-btn v-if="editingNoteId" icon="mdi-close" size="x-small" variant="text" color="error" class="tool-btn"
            @click="emit('update:modelValue', ''); emit('update:uploadedImages', [])" />
        </div>
        <v-btn size="small" color="primary" variant="flat" class="rounded-pill px-4 submit-btn" @click="$emit('submit')">
          <v-icon start size="x-small">mdi-send</v-icon>{{ editingNoteId ? "更新" : "发布" }}
        </v-btn>
      </div>
      <v-expand-transition>
        <div v-if="showTags" class="pa-3">
          <v-text-field :model-value="tagsInput" @update:model-value="emit('update:tagsInput', $event)" label="标签（逗号分隔）" variant="outlined" hide-details density="compact" placeholder="vue, memos, md" />
        </div>
      </v-expand-transition>
    </div>
  </div>
</template>

<style scoped>
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
.submit-btn { }
.md-toolbar {
  display: flex; align-items: center; gap: 0;
  padding: 4px 8px 0;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}
.md-toolbar .tool-btn { width: 28px; height: 28px; opacity: 0.5; }
.md-toolbar .tool-btn:hover { opacity: 1; }
.upload-preview { position: relative; width: 64px; height: 64px; border-radius: 8px; overflow: hidden; }
.upload-thumb { width: 100%; height: 100%; object-fit: cover; }
.upload-remove { position: absolute; top: 2px; right: 2px; width: 18px; height: 18px; border-radius: 50%; border: none; background: rgba(0,0,0,0.5); color: #fff; font-size: 12px; line-height: 18px; text-align: center; cursor: pointer; display: none; }
.upload-preview:hover .upload-remove { display: block; }

@media (max-width: 768px) {
  .inline-textarea { min-height: 60px; padding: 12px 14px 8px; font-size: 0.9rem; }
}
</style>
