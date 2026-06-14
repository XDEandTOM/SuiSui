<script setup lang="ts">
import { ref, onMounted, nextTick, defineAsyncComponent, watch } from "vue"
import type { Note } from "@/stores/notes"
import { useNotesStore } from "@/stores/notes"
import { useAuthStore } from "@/stores/auth"
const MarkdownPreview = defineAsyncComponent(() => import("./MarkdownPreview.vue"))


const props = defineProps<{ memo: Note; loggedIn: boolean; searchQuery?: string }>()
const emit = defineEmits<{ edit: [memo: Note]; movePin: [note: Note, dir: "up" | "down"] }>()
const store = useNotesStore()
const auth = useAuthStore()

const expanded = ref(false)
const contentRef = ref<HTMLElement | null>(null)
const isOverflow = ref(false)
const showShareDialog = ref(false)
const shareLink = ref("")
const shareCopied = ref(false)
const showCopiedToast = ref(false)

const TAG_COLORS = ["primary", "teal", "orange", "pink", "indigo", "cyan", "deep-purple", "amber"]
function tagColor(tag: string) {
  let h = 0; for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) | 0
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}

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
const emojiCategories = ref<{ id: number; icon: string; list: string[] }[]>([])
const groupLabels: Record<number, string> = { 0: "😊", 1: "🤝", 3: "🐻", 4: "🍔", 5: "🏠", 6: "⚽", 7: "💡", 8: "❤️", 9: "🚩" }
const activeEmojiCat = ref(0)

async function loadEmojiData() {
  if (emojiCategories.value.length) return
  const raw = (await import("emojibase-data/zh/compact.json")).default
  const cats = [0,1,3,4,5,6,7,8,9].map(g => ({ id: g, icon: groupLabels[g] || "?", list: [] as string[] }))
  for (const e of raw) {
    if (e.group === undefined || e.group === 2) continue
    const cat = cats.find(c => c.id === e.group)
    if (cat && e.unicode) cat.list.push(e.unicode)
  }
  emojiCategories.value = cats
}

watch(showEmojiPicker, (v) => { if (v) loadEmojiData() })

function getReactionUserId() {
  if (auth.isLoggedIn && auth.userName) return auth.userName
  let gid = localStorage.getItem("suisui-guest")
  if (!gid) { gid = "guest_" + Date.now().toString(36) + Math.random().toString(36).slice(2,6); localStorage.setItem("suisui-guest", gid) }
  return gid
}


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
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, "0")
  const dateStr = `${d.getMonth() + 1}月${pad(d.getDate())}日`
  const timeStr = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const year = d.getFullYear()
  const nowYear = new Date().getFullYear()
  if (year !== nowYear) return `${year}年${dateStr} ${timeStr}`
  return `${dateStr} ${timeStr}`
}

function copyContent() {
  navigator.clipboard.writeText(props.memo.content).then(() => {
    showCopiedToast.value = true
    setTimeout(() => showCopiedToast.value = false, 1500)
  }).catch(() => {
    const ta = document.createElement("textarea")
    ta.value = props.memo.content
    document.body.appendChild(ta)
    ta.select()
    document.execCommand("copy")
    document.body.removeChild(ta)
  })
}

function exportMarkdown() {
  const content = props.memo.content
  const tags = props.memo.tags?.length ? `\n\n标签：${props.memo.tags.join(", ")}` : ""
  const meta = `---\n日期：${new Date(props.memo.createdAt).toLocaleDateString("zh-CN")}\n作者：${displayName(props.memo)}${tags}\n---\n\n`
  const full = meta + content
  const blob = new Blob([full], { type: "text/markdown" })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `note-${props.memo.id}.md`
  a.click()
  URL.revokeObjectURL(url)
}

function shareNote() {
  shareCopied.value = false
  shareLink.value = window.location.origin + "/share/" + props.memo.id
  showShareDialog.value = true
}

function copyShareLink() {
  navigator.clipboard.writeText(shareLink.value).then(() => {
    shareCopied.value = true
    setTimeout(() => { shareCopied.value = false }, 2000)
  })
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
        <div class="flex-grow-1" style="min-width:0;overflow:hidden">
          <div class="d-flex align-center ga-1">
            <span class="nickname text-truncate">{{ displayName(memo) }}</span>
            <v-icon v-if="memo.pinned" size="x-small" color="primary">mdi-pin</v-icon>
          </div>
          <div class="time">{{ timeAgo(memo.createdAt) }}</div>
        </div>
        <div class="d-flex ga-0 flex-shrink-0" style="margin-top:2px;align-items:center">
          <template v-if="loggedIn && (auth.isAdmin || memo.username === auth.userName)">
            <template v-if="memo.pinned">
              <v-btn icon="mdi-chevron-up" size="x-small" variant="text" class="pin-move-btn" @click="emit('movePin', memo, 'up')" />
              <v-btn icon="mdi-chevron-down" size="x-small" variant="text" class="pin-move-btn" @click="emit('movePin', memo, 'down')" />
            </template>
            <v-btn icon="mdi-pencil" size="x-small" variant="text" class="action-btn" @click="emit('edit', memo)" />
            <v-btn icon="mdi-pin-outline" size="x-small" variant="text"
              :color="memo.pinned ? 'primary' : undefined" class="action-btn"
              @click="store.togglePin(memo.id)" />
            <v-btn icon="mdi-delete" size="x-small" variant="text" color="error" class="action-btn"
              @click="store.deleteNote(memo.id, auth.userName)" />
          </template>
          <v-btn icon="mdi-share-variant" size="x-small" variant="text" class="action-btn"
            @click="shareNote" />
          <v-menu location="bottom">
            <template #activator="{ props: menuProps }">
              <v-btn icon="mdi-dots-horizontal" size="x-small" variant="text" class="action-btn" v-bind="menuProps" />
            </template>
            <v-list density="compact" class="pa-1">
              <v-list-item density="compact" @click="copyContent">
                <template #prepend><v-icon size="small">mdi-content-copy</v-icon></template>
                <v-list-item-title>复制内容</v-list-item-title>
              </v-list-item>
              <v-list-item density="compact" @click="exportMarkdown">
                <template #prepend><v-icon size="small">mdi-file-download</v-icon></template>
                <v-list-item-title>导出 Markdown</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
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
          :color="tagColor(tag)" class="tag-chip-card">
#{{ tag }}
</v-chip>
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
          <template #activator="{ props: menuProps }">
            <v-btn icon="mdi-plus-circle-outline" size="x-small" variant="text"
              class="reaction-add-btn" v-bind="menuProps" />
          </template>
          <div class="emoji-picker" style="width:280px">
            <div class="d-flex ga-1 pa-2" style="border-bottom:1px solid rgba(var(--v-theme-on-surface),0.08);overflow-x:auto">
              <v-btn v-for="cat in emojiCategories" :key="cat.id" size="x-small" variant="text"
                :class="['cat-btn', { active: activeEmojiCat === cat.id }]"
                @click="activeEmojiCat = cat.id">
{{ cat.icon }}
</v-btn>
            </div>
            <div class="emoji-grid pa-2">
              <v-btn v-for="(e, ei) in emojiCategories.find(c => c.id === activeEmojiCat)?.list || []" :key="activeEmojiCat + '-' + ei"
                size="x-small" variant="text" class="emoji-btn"
                @click="toggleReaction(e); showEmojiPicker = false">
{{ e }}
</v-btn>
            </div>
          </div>
        </v-menu>
      </div>
</div>
  </v-card>

  <!-- Share dialog -->
  <v-dialog v-model="showShareDialog" max-width="400" persistent>
    <v-card class="rounded-xl pa-4">
      <div class="d-flex align-center mb-3">
        <span class="text-subtitle-2 font-weight-medium">分享笔记</span>
        <v-spacer />
        <v-btn icon="mdi-close" size="x-small" variant="text" @click="showShareDialog = false" />
      </div>
      <div class="d-flex align-center ga-2 mb-3">
        <v-text-field v-model="shareLink" variant="outlined" hide-details density="compact" readonly
          class="share-link-input" />
        <v-btn :color="shareCopied ? 'success' : 'primary'" variant="flat" size="small" class="rounded-pill flex-shrink-0"
          @click="copyShareLink">
          <v-icon start>{{ shareCopied ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
          {{ shareCopied ? '已复制' : '复制' }}
        </v-btn>
      </div>
      <div class="text-caption text-medium-emphasis mb-2">任何拥有此链接的人都可以查看这条笔记</div>
    </v-card>
  </v-dialog>
  <Transition name="toast-fade">
    <div v-if="showCopiedToast" class="copy-toast">
      <v-icon size="small" class="mr-1">mdi-check</v-icon>已复制
    </div>
  </Transition>
</template>

<style scoped>
.memo-card {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.06);
  border-radius: 14px !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(0,0,0,0.02), 0 1px 4px rgba(0,0,0,0.02);
  background: rgba(var(--v-theme-surface), 0.7);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
}
.memo-card:hover {
  border-color: rgba(var(--v-theme-primary), 0.2);
  box-shadow: 0 2px 8px rgba(0,0,0,0.04), 0 8px 24px rgba(var(--v-theme-primary), 0.06) !important;
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
  width: 26px !important;
  min-width: 26px !important;
  height: 26px !important;
  opacity: 0;
  transition: opacity 0.2s, transform 0.15s;
}
.pin-move-btn {
  width: 26px !important;
  min-width: 26px !important;
  height: 26px !important;
  opacity: 0;
  transition: opacity 0.2s;
}
.memo-card:hover .action-btn,
.memo-card:hover .pin-move-btn {
  opacity: 0.8;
  transform: scale(1);
}
.action-btn:hover,
.pin-move-btn:hover { opacity: 1 !important; }
.tags-row { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.tag-chip-card { font-size: 0.7rem; height: 22px !important; }
</style>

<style scoped>
.nickname { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.reactions-row { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; margin-top: 8px; }
.reaction-chip { font-size: 0.75rem; height: 26px !important; cursor: pointer; }
.reaction-chip.active { outline: 1px solid rgb(var(--v-theme-primary)); }
.reaction-add-btn { opacity: 0.4; transition: opacity 0.2s; }
.reaction-add-btn:hover { opacity: 1; }
.emoji-picker { background: rgba(var(--v-theme-surface), 0.92); backdrop-filter: blur(16px); -webkit-backdrop-filter: blur(16px); border: 1px solid rgba(var(--v-theme-on-surface),0.08); border-radius: 14px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.08); }
.emoji-btn { font-size: 1.1rem; width: 32px; height: 32px; min-width: 0 !important; padding: 0 !important; }
.cat-btn { font-size: 1rem; width: 28px; height: 28px; min-width: 0 !important; border-radius: 8px; opacity:0.5; transition:all 0.15s; }
.cat-btn:hover { opacity:1; }
.cat-btn.active { opacity:1; background: rgba(var(--v-theme-primary),0.1); }
.emoji-grid { display: grid; grid-template-columns: repeat(7, 32px); gap: 4px; max-height: 280px; overflow-y: auto; }
.share-link-input :deep(input) { font-size: 0.8rem !important; color: rgb(var(--v-theme-primary)); }

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
  .pin-move-btn, .action-btn { width: 24px !important; min-width: 24px !important; height: 24px !important; }
}
</style>
<style>
.copy-toast {
  position: fixed; bottom: 24px; left: 50%; transform: translateX(-50%);
  z-index: 9999; padding: 8px 18px; border-radius: 10px;
  background: rgb(var(--v-theme-primary)); color: #fff;
  font-size: 0.82rem; display: flex; align-items: center;
  box-shadow: 0 4px 16px rgba(0,0,0,0.15);
}
.toast-fade-enter-active { transition: all 0.2s ease; }
.toast-fade-leave-active { transition: all 0.2s ease; }
.toast-fade-enter-from, .toast-fade-leave-to { opacity: 0; transform: translateX(-50%) translateY(8px); }
</style>


