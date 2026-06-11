<script setup lang="ts">
import { ref, onMounted } from "vue"
import type { Note } from "@/stores/notes"
import MarkdownPreview from "./MarkdownPreview.vue"
import AppLogo from "./AppLogo.vue"

const note = ref<Note | null>(null)
const loading = ref(true)
const error = ref("")

async function loadSharedNote() {
  const token = window.location.pathname.replace("/share/", "")
  if (!token) {
    error.value = "无效的分享链接"
    loading.value = false
    return
  }
  try {
    const res = await fetch(`/api/share/${token}`)
    if (res.ok) {
      note.value = await res.json()
    } else {
      const data = await res.json()
      error.value = data.error || "笔记不存在或分享链接已失效"
    }
  } catch {
    error.value = "无法加载笔记"
  }
  loading.value = false
}

onMounted(loadSharedNote)

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

function displayName(memo: Note) {
  return memo.nickname?.trim() || memo.username || "匿名"
}

function isImage(val?: string) {
  return val?.startsWith("/uploads/") || val?.startsWith("http")
}
</script>

<template>
  <div class="share-page">
    <div class="share-header">
      <a href="/" class="share-home-link">
        <AppLogo :size="20" />
        <span>碎碎 SuiSui</span>
      </a>
    </div>

    <main class="share-main">
      <div v-if="loading" class="d-flex justify-center py-16">
        <v-progress-circular indeterminate color="primary" />
      </div>
      <div v-else-if="error" class="share-error">
        <v-icon size="48" color="error" class="mb-3">mdi-link-variant-off</v-icon>
        <h2>分享链接无效</h2>
        <p class="text-medium-emphasis">{{ error }}</p>
        <v-btn variant="flat" color="primary" href="/" class="mt-4 rounded-pill">返回首页</v-btn>
      </div>
      <div v-else-if="note" class="share-note-card">
        <div class="share-note-header">
          <div class="d-flex align-center ga-2">
            <div v-if="isImage(note.avatar)" class="share-avatar">
              <img :src="note.avatar" alt="" width="32" height="32" style="border-radius:8px;object-fit:cover" />
            </div>
            <div v-else class="share-avatar-fallback">{{ displayName(note).charAt(0).toUpperCase() }}</div>
            <div>
              <div class="share-author-name">{{ displayName(note) }}</div>
              <div class="share-time">{{ timeAgo(note.createdAt) }}</div>
            </div>
          </div>
        </div>
        <div class="share-content">
          <MarkdownPreview :content="note.content" />
        </div>
        <div v-if="note.tags && note.tags.length" class="d-flex flex-wrap ga-1 mt-3">
          <v-chip v-for="tag in note.tags" :key="tag" size="x-small" variant="tonal" color="primary">
            #{{ tag }}
          </v-chip>
        </div>
      </div>
    </main>

    <footer class="share-footer">
      <span>来自 <a href="/">碎碎 SuiSui</a> 的分享</span>
    </footer>
  </div>
</template>

<style scoped>
.share-page {
  min-height: 100vh;
  background: rgb(var(--v-theme-background));
  display: flex;
  flex-direction: column;
}
.share-header {
  display: flex;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}
.share-home-link {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: rgb(var(--v-theme-on-surface));
  font-weight: 600;
  font-size: 0.95rem;
  opacity: 0.7;
  transition: opacity 0.2s;
}
.share-home-link:hover { opacity: 1; }
.share-main {
  flex: 1;
  display: flex;
  justify-content: center;
  padding: 40px 16px;
}
.share-error {
  text-align: center;
  padding: 48px 16px;
}
.share-error h2 { font-size: 1.2rem; margin-bottom: 8px; }
.share-note-card {
  width: 100%;
  max-width: 680px;
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 16px;
  padding: 24px;
}
.share-note-header {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}
.share-avatar-fallback {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--v-theme-primary));
  color: #fff;
  font-size: 0.8rem;
  font-weight: 600;
}
.share-author-name { font-weight: 600; font-size: 0.95rem; }
.share-time { font-size: 0.75rem; color: rgba(var(--v-theme-on-surface), 0.45); }
.share-content {
  line-height: 1.7;
  font-size: 0.95rem;
}
.share-footer {
  text-align: center;
  padding: 16px;
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.35);
}
.share-footer a { color: inherit; }
</style>
