<script setup lang="ts">
import { ref, onMounted } from "vue"
import type { Note } from "@/stores/notes"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
import MarkdownPreview from "./MarkdownPreview.vue"
import AppLogo from "./AppLogo.vue"
import emojiRaw from "emojibase-data/zh/compact.json"

const note = ref<Note | null>(null)
const loading = ref(true)
const error = ref("")
const store = useNotesStore()
const auth = useAuthStore()

const showEmojiPicker = ref(false)
const groupLabels: Record<number, string> = { 0: "😊", 1: "🤝", 3: "🐻", 4: "🍔", 5: "🏠", 6: "⚽", 7: "💡", 8: "❤️", 9: "🚩" }
const EMOJI_CATEGORIES = (() => {
  const cats = [0,1,3,4,5,6,7,8,9].map(g => ({ id: g, icon: groupLabels[g] || "?", list: [] as string[] }))
  for (const e of emojiRaw) {
    if (e.group === undefined || e.group === 2) continue
    const cat = cats.find(c => c.id === e.group)
    if (cat && e.unicode) cat.list.push(e.unicode)
  }
  return cats
})()
const activeEmojiCat = ref(0)

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

function getReactionUserId() {
  if (auth.isLoggedIn && auth.userName) return auth.userName
  let gid = localStorage.getItem("suisui-guest")
  if (!gid) { gid = "guest_" + Date.now().toString(36) + Math.random().toString(36).slice(2,6); localStorage.setItem("suisui-guest", gid) }
  return gid
}

function hasReacted(emoji: string) {
  return note.value?.reactions?.[emoji]?.includes(getReactionUserId())
}

function toggleReaction(emoji: string) {
  const n = note.value
  if (!n) return
  if (hasReacted(emoji)) {
    store.removeReaction(n.id, emoji, getReactionUserId())
    const users = n.reactions?.[emoji]
    if (users) {
      const r = n.reactions!
      r[emoji] = users.filter(u => u !== getReactionUserId())
      if (r[emoji].length === 0) delete r[emoji]
    }
  } else {
    store.reactToNote(n.id, emoji, getReactionUserId())
    if (!n.reactions) n.reactions = {}
    if (!n.reactions[emoji]) n.reactions[emoji] = []
    n.reactions[emoji].push(getReactionUserId())
  }
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
              <img :src="note.avatar" alt="" width="28" height="28" style="border-radius:6px;object-fit:cover" />
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
        <div v-if="note.tags && note.tags.length" class="d-flex flex-wrap ga-1 mt-1">
          <v-chip v-for="tag in note.tags" :key="tag" size="x-small" variant="tonal" color="primary">
            #{{ tag }}
          </v-chip>
        </div>
        <div class="reactions-row mt-1">
          <template v-for="(users, emoji, ri) in note.reactions || {}" :key="ri">
            <v-chip v-if="users && users.length" size="x-small" variant="tonal"
              :class="['reaction-chip', { active: hasReacted(emoji) }]"
              @click="toggleReaction(emoji)">
              {{ emoji }} {{ users.length }}
            </v-chip>
          </template>
          <v-menu v-model="showEmojiPicker" :close-on-content-click="false" location="top">
            <template #activator="{ props: menuProps }">
              <v-btn icon="mdi-plus-circle-outline" size="x-small" variant="text"
                class="reaction-add-btn" v-bind="menuProps" />
            </template>
            <div class="emoji-picker" style="width:280px">
              <div class="d-flex ga-1 pa-2" style="border-bottom:1px solid rgba(var(--v-theme-on-surface),0.08);overflow-x:auto">
                <v-btn v-for="cat in EMOJI_CATEGORIES" :key="cat.id" size="x-small" variant="text"
                  :class="['cat-btn', { active: activeEmojiCat === cat.id }]"
                  @click="activeEmojiCat = cat.id">{{ cat.icon }}</v-btn>
              </div>
              <div class="emoji-grid pa-2">
                <v-btn v-for="(e, ei) in EMOJI_CATEGORIES.find(c => c.id === activeEmojiCat)?.list || []" :key="activeEmojiCat + '-' + ei"
                  size="x-small" variant="text" class="emoji-btn"
                  @click="toggleReaction(e); showEmojiPicker = false">{{ e }}</v-btn>
              </div>
            </div>
          </v-menu>
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
  background: rgb(var(--v-theme-background));
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
  display: flex;
  justify-content: center;
  padding: 32px 16px;
}
.share-error {
  text-align: center;
  padding: 48px 16px;
}
.share-error h2 { font-size: 1.2rem; margin-bottom: 8px; }
.share-note-card {
  width: 100%;
  max-width: 560px;
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 12px;
  padding: 14px 18px 10px;
}
.share-note-header {
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}
.share-avatar-fallback {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--v-theme-primary));
  color: #fff;
  font-size: 0.75rem;
  font-weight: 600;
}
.share-author-name { font-weight: 600; font-size: 0.85rem; }
.share-time { font-size: 0.7rem; color: rgba(var(--v-theme-on-surface), 0.45); }
.share-content {
  line-height: 1.55;
  font-size: 0.88rem;
}
.share-footer {
  text-align: center;
  padding: 16px;
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.35);
}
.share-footer a { color: inherit; }
.reactions-row { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; }
.reaction-chip { font-size: 0.75rem; height: 26px !important; cursor: pointer; }
.reaction-chip.active { outline: 1px solid rgb(var(--v-theme-primary)); }
.reaction-add-btn { opacity: 0.4; }
.reaction-add-btn:hover { opacity: 1; }
.emoji-picker { background: rgb(var(--v-theme-surface)); border: 1px solid rgba(var(--v-theme-on-surface),0.12); border-radius: 12px; overflow: hidden; }
.emoji-btn { font-size: 1.1rem; width: 32px; height: 32px; min-width: 0 !important; padding: 0 !important; }
.cat-btn { font-size: 1rem; width: 28px; height: 28px; min-width: 0 !important; border-radius: 8px; opacity:0.5; transition:all 0.15s; }
.cat-btn:hover { opacity:1; }
.cat-btn.active { opacity:1; background: rgba(var(--v-theme-primary),0.1); }
.emoji-grid { display: grid; grid-template-columns: repeat(7, 32px); gap: 4px; max-height: 280px; overflow-y: auto; }
</style>
