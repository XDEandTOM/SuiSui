<script setup lang="ts">
import { computed } from "vue"
import Heatmap from "./Heatmap.vue"
import type { Note } from "@/stores/notes"

const props = defineProps<{
  search: string
  selectedTag: string
  selectedDay: string
  allTags: [string, number][]
  versionText: string
  outline: { level: number; text: string; noteId: string }[]
}>()
const emit = defineEmits<{
  "update:search": [v: string]
  "update:selectedTag": [v: string]
  "update:selectedDay": [v: string]
  "scroll-to-note": [id: string]
}>()

const TAG_COLORS = ["primary", "teal", "orange", "pink", "indigo", "cyan", "deep-purple", "amber"]
function tagColor(tag: string) {
  let h = 0; for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) | 0
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}
</script>

<template>
  <div class="side-content">
    <v-text-field :model-value="search" @update:model-value="v => emit('update:search', v)"
      prepend-inner-icon="mdi-magnify" label="搜索备忘..." variant="outlined" hide-details density="compact"
      clearable class="mb-3 rounded-search search-border" data-search-input />
    <Heatmap class="mb-4" style="border-color:#424242 !important" @select-day="emit('update:selectedDay', $event)" />
    <v-card variant="outlined" class="rounded-xl pa-4 side-card">
      <div class="d-flex align-center ga-2 mb-3">
        <span class="text-subtitle-2 font-weight-medium">标签</span>
      </div>
      <div class="d-flex flex-wrap ga-1">
        <v-chip v-for="[tag] in allTags" :key="tag" size="x-small" class="tag-chip"
          :color="tagColor(tag)"
          :variant="selectedTag === tag ? 'flat' : 'outlined'"
          @click="emit('update:selectedTag', selectedTag === tag ? '' : tag)">
          #{{ tag }}
        </v-chip>
        <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">暂无标签</div>
      </div>
    </v-card>
    <v-card v-if="outline.length" variant="outlined" class="rounded-xl pa-4 side-card mt-3">
      <div class="d-flex align-center ga-2 mb-2">
        <span class="text-subtitle-2 font-weight-medium">大纲</span>
        <span class="outline-count">{{ outline.length }}</span>
      </div>
      <div class="outline-list">
        <div v-for="(h, i) in outline.slice(0, 20)" :key="i" class="outline-item"
          :style="{ paddingLeft: (h.level - 1) * 10 + 'px' }" :title="h.text"
          @click="emit('scroll-to-note', h.noteId)">
          <span class="outline-marker" :style="{ width: 3 + (5 - h.level) + 'px' }" />
          <span class="outline-text">{{ h.text }}</span>
        </div>
        <div v-if="outline.length > 20" class="outline-more">+{{ outline.length - 20 }} 个标题</div>
      </div>
    </v-card>
    <div v-if="versionText" class="d-flex justify-center mt-2">
      <v-chip size="x-small" variant="tonal" color="primary" class="version-chip" style="cursor:pointer" prepend-icon="mdi-github">
        {{ versionText }}
      </v-chip>
    </div>
  </div>
</template>

<style scoped>
.side-card { border-color: #424242 !important; }
.tag-chip { cursor: pointer; }
.tag-chip:hover { opacity: 0.9; }
.rounded-search :deep(.v-field) { border-radius: 12px !important; }
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
</style>
