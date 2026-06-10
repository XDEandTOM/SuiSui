<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue"
import type { Note } from "@/stores/notes"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
import MarkdownPreview from "./MarkdownPreview.vue"
import emojiRaw from "emojibase-data/zh/compact.json"

const props = defineProps<{ memo: Note; loggedIn: boolean; searchQuery?: string }>()
const emit = defineEmits<{ edit: [memo: Note] }>()
const store = useNotesStore()
const auth = useAuthStore()

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

function handleTodoToggle(idx: number) {
  // Only allow toggling your own notes
  if (!auth.isLoggedIn || props.memo.username !== auth.userName) return
  const lines = props.memo.content.split("\n")
  let found = -1
  for (let i = 0; i < lines.length; i++) {
    if (/^\s*[-*+]\s+\[[ x]\]/.test(lines[i])) {
      found++
      if (found === idx) {
        lines[i] = lines[i].includes("[x]")
          ? lines[i].replace("[x]", "[ ]")
          : lines[i].replace("[ ]", "[x]")
        store.updateNote(props.memo.id, lines.join("\n"), undefined, auth.userName)
        break
      }
    }
  }
}

const showEmojiPicker = ref(false)

function getReactionUserId() {
  if (auth.isLoggedIn && auth.userName) return auth.userName
  let gid = localStorage.getItem("suisui-guest")
  if (!gid) { gid = "guest_" + Date.now().toString(36) + Math.random().toString(36).slice(2,6); localStorage.setItem("suisui-guest", gid) }
  return gid
}
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

function hasReacted(emoji: string) {
  return props.memo.reactions?.[emoji]?.includes(getReactionUserId())
}

function toggleReaction(emoji: string) {
  if (hasReacted(emoji)) store.removeReaction(props.memo.id, emoji, getReactionUserId())
  else store.reactToNote(props.memo.id, emoji, getReactionUserId())
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
        <v-icon v-if="memo.pinned" size="small" color="rgba(var(--v-theme-on-surface),0.15)" class="drag-handle">mdi-drag</v-icon>
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
        <div v-if="loggedIn && (auth.isAdmin || memo.username === auth.userName)" class="d-flex ga-1 flex-shrink-0" style="margin-top:2px">
          <v-btn icon="mdi-pencil" size="x-small" variant="text" class="action-btn" @click="emit('edit', memo)" />
          <v-btn icon="mdi-pin-outline" size="x-small" variant="text"
            :color="memo.pinned ? 'primary' : undefined" class="action-btn"
            @click="store.togglePin(memo.id)" />
          <v-btn icon="mdi-delete" size="x-small" variant="text" color="error" class="action-btn"
            @click="store.deleteNote(memo.id, auth.userName)" />
        </div>
      </div>
      <div ref="contentRef" class="memo-content" :class="{ collapsed: !expanded && isOverflow, 'not-owned': (auth.isLoggedIn && memo.username !== auth.userName) || !auth.isLoggedIn }">
        <MarkdownPreview :content="memo.content" :search-query="props.searchQuery" @todo-toggle="handleTodoToggle" />
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
    
      <div class="reactions-row">
        <template v-for="(users, emoji, ri) in memo.reactions" :key="ri">
          <v-chip v-if="users && users.length" size="x-small" variant="tonal"
            :class="['reaction-chip', { active: hasReacted(emoji) }]"
            @click="toggleReaction(emoji)">
            {{ emoji }} {{ users.length }}
          </v-chip>
        </template>
        <v-menu v-model="showEmojiPicker" :close-on-content-click="false" location="top">
          <template v-slot:activator="{ props }">
            <v-btn icon="mdi-plus-circle-outline" size="x-small" variant="text"
              class="reaction-add-btn" v-bind="props" />
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
      </div></div>
  </v-card>
</template>

<style scoped>
.memo-card {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.06);
  border-radius: 14px !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02);
}
.memo-card:hover {
  border-color: rgba(var(--v-theme-primary), 0.2);
  box-shadow: 0 6px 24px rgba(var(--v-theme-primary), 0.06), 0 2px 6px rgba(0,0,0,0.04) !important;
  transform: translateY(-2px);
}
.memo-card.pinned {
  border-left: 3px solid rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.015);
}
.memo-content.not-owned :deep(input[type="checkbox"]) {
  pointer-events: none;
  opacity: 0.5;
}
.card-inner { padding: 14px; }
.avatar-wrap {
  width: 38px; height: 38px; flex-shrink: 0; overflow: hidden;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(var(--v-theme-primary), 0.15);
}
.avatar-img { border-radius: 10px; }
.avatar-fallback {
  width: 100%; height: 100%; display: flex; align-items: center;
  justify-content: center; color: #fff; font-size: 0.85rem; font-weight: 600;
  background: rgb(var(--v-theme-primary));
}
.nickname { font-size: 1.1rem; font-weight: 600; line-height: 1.3; letter-spacing: -0.01em; }
.time { font-size: 0.68rem; color: rgba(var(--v-theme-on-surface), 0.45); line-height: 1; margin-top: 2px; }
.action-btn {
  opacity: 0;
  transition: opacity 0.2s, transform 0.15s;
  transform: scale(0.9);
}
.memo-card:hover .action-btn {
  opacity: 0.6;
  transform: scale(1);
}
.action-btn:hover { opacity: 1 !important; }
.tags-row { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.tag-chip-card { font-size: 0.7rem; height: 22px !important; }
.reactions-row { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; margin-top: 8px; }
.reaction-chip { font-size: 0.75rem; height: 26px !important; cursor: pointer; }
.reaction-chip.active { outline: 1px solid rgb(var(--v-theme-primary)); }
.reaction-add-btn { opacity: 0.4; }
.reaction-add-btn:hover { opacity: 1; }
.drag-handle { cursor: grab; margin-top: 12px; }
.emoji-picker { background: rgb(var(--v-theme-surface)); border: 1px solid rgba(var(--v-theme-on-surface),0.12); border-radius: 12px; overflow: hidden; }
.emoji-btn { font-size: 1.1rem; width: 32px; height: 32px; min-width: 0 !important; padding: 0 !important; }
.cat-btn { font-size: 1rem; width: 28px; height: 28px; min-width: 0 !important; border-radius: 8px; opacity:0.5; transition:all 0.15s; }
.cat-btn:hover { opacity:1; }
.cat-btn.active { opacity:1; background: rgba(var(--v-theme-primary),0.1); }
.emoji-grid { display: grid; grid-template-columns: repeat(7, 32px); gap: 4px; max-height: 280px; overflow-y: auto; }

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


