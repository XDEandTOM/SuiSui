<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue"
import type { Note } from "@/stores/notes"
import { useNotesStore } from "@/stores/notes"
import MarkdownPreview from "./MarkdownPreview.vue"

const props = defineProps<{ memo: Note; loggedIn: boolean }>()
const emit = defineEmits<{ edit: [memo: Note] }>()
const store = useNotesStore()

const expanded = ref(false)
const contentRef = ref<HTMLElement | null>(null)
const isOverflow = ref(false)

onMounted(() => { nextTick(checkOverflow) })

function checkOverflow() {
  const el = contentRef.value
  if (el) { isOverflow.value = el.scrollHeight > 300 }
}

function isImage(val?: string) {
  return val?.startsWith("/uploads/") || val?.startsWith("http")
}

function displayName(memo: Note) {
  return memo.nickname?.trim() || memo.username || "匿名"
}

function timeAgo(ts: number) {
  const diff = Date.now() - ts
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return "刚刚"
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} 个月前`
  return `${Math.floor(months / 12)} 年前`
}
</script>

<template>
  <v-card variant="flat" class="memo-card" :class="{ pinned: memo.pinned }">
    <div class="card-inner">
      <div class="d-flex align-start ga-3 mb-2">
        <div class="avatar-wrap">
          <v-img v-if="isImage(memo.avatar)" :src="memo.avatar" alt="" cover width="40" height="40" class="avatar-img" />
          <div v-else class="avatar-fallback">{{ displayName(memo).charAt(0).toUpperCase() }}</div>
        </div>
        <div class="flex-grow-1" style="min-width:0">
          <div class="d-flex align-center ga-1">
            <span class="nickname">{{ displayName(memo) }}</span>
            <v-icon v-if="memo.pinned" size="x-small" color="primary">mdi-pin</v-icon>
          </div>
          <div class="time">{{ timeAgo(memo.createdAt) }}</div>
        </div>
        <div v-if="loggedIn" class="d-flex ga-1 flex-shrink-0" style="margin-top:2px">
          <v-btn icon="mdi-pencil" size="x-small" variant="text" class="action-btn" @click="emit('edit', memo)" />
          <v-btn icon="mdi-pin-outline" size="x-small" variant="text"
            :color="memo.pinned ? 'primary' : undefined" class="action-btn"
            @click="store.togglePin(memo.id)" />
          <v-btn icon="mdi-delete" size="x-small" variant="text" color="error" class="action-btn"
            @click="store.deleteNote(memo.id)" />
        </div>
      </div>
      <div ref="contentRef" class="memo-content" :class="{ collapsed: !expanded && isOverflow }">
        <MarkdownPreview :content="memo.content" />
      </div>
      <div v-if="isOverflow" class="expand-bar">
        <button class="expand-btn" @click="expanded = !expanded">
          {{ expanded ? "收起" : "展开全文" }}
          <v-icon size="x-small">{{ expanded ? "mdi-chevron-up" : "mdi-chevron-down" }}</v-icon>
        </button>
      </div>
      <div v-if="memo.tags && memo.tags.length" class="tags-row">
        <v-chip v-for="tag in memo.tags" :key="tag" size="x-small" variant="tonal"
          color="primary" class="tag-chip-card">#{{ tag }}</v-chip>
      </div>
    </div>
  </v-card>
</template>

<style scoped>
.memo-card {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 12px !important;
  transition: all 0.25s ease;
  overflow: hidden;
}
.memo-card:hover {
  border-color: rgba(var(--v-theme-primary), 0.25);
  box-shadow: 0 4px 16px rgba(var(--v-theme-primary), 0.08) !important;
  transform: translateY(-1px);
}
.memo-card.pinned {
  border-left: 3px solid rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.02);
}
.card-inner { padding: 12px; }
.avatar-wrap {
  width: 40px; height: 40px; flex-shrink: 0; overflow: hidden;
  border-radius: 8px; background: rgb(var(--v-theme-primary));
}
.avatar-img { border-radius: 8px; }
.avatar-fallback {
  width: 100%; height: 100%; display: flex; align-items: center;
  justify-content: center; color: #fff; font-size: 0.85rem; font-weight: 600;
}
.nickname { font-size: 1.25rem; font-weight: 500; line-height: 1.25; }
.time { font-size: 0.7rem; color: rgba(var(--v-theme-on-surface), 0.5); line-height: 1; margin-top: 1px; }
.action-btn { opacity: 0.5; transition: opacity 0.2s; }
.memo-card:hover .action-btn { opacity: 1; }
.tags-row { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.tag-chip-card { font-size: 0.7rem; height: 22px !important; }

.memo-content.collapsed {
  max-height: 300px;
  overflow: hidden;
  position: relative;
}
.memo-content.collapsed::after {
  content: "";
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: linear-gradient(transparent, rgb(var(--v-theme-surface)));
  pointer-events: none;
}
.expand-bar {
  text-align: center;
  padding: 4px 0 0;
}
.expand-btn {
  background: none;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 20px;
  padding: 4px 16px;
  font-size: 0.8rem;
  color: rgb(var(--v-theme-primary));
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.2s;
}
.expand-btn:hover {
  background: rgba(var(--v-theme-primary), 0.06);
  border-color: rgba(var(--v-theme-primary), 0.3);
}

@media (max-width: 768px) {
  .card-inner { padding: 10px; }
  .nickname { font-size: 1rem !important; }
  .avatar-wrap { width: 32px; height: 32px; }
  .memo-card { border-radius: 10px !important; }
}
</style>


