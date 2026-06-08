<script setup lang="ts">
import { ref, computed } from "vue"
import { useNotesStore } from "@/stores/notes"
import Heatmap from "./Heatmap.vue"

const emit = defineEmits<{ "tag-click": [] }>()
const store = useNotesStore()
const searchQuery = defineModel<string>("search", { default: "" })
const selectedTag = defineModel<string>("tag", { default: "" })
const props = defineProps<{ emitOnTag?: boolean }>()

const allTags = computed(() => {
  const tagCount = new Map<string, number>()
  for (const n of store.notes) {
    for (const t of n.tags) tagCount.set(t, (tagCount.get(t) || 0) + 1)
  }
  return [...tagCount.entries()].sort((a, b) => b[1] - a[1])
})

function onTagClick(tag: string) {
  selectedTag.value = selectedTag.value === tag ? "" : tag
  if (props.emitOnTag) emit("tag-click")
}
</script>

<template>
  <v-text-field v-model="searchQuery" prepend-inner-icon="mdi-magnify"
    label="��������..." variant="outlined" hide-details density="compact"
    clearable class="mb-3 rounded-search" />
  <Heatmap class="mb-4" />
  <v-card variant="outlined" class="rounded-xl pa-4 side-card">
    <div class="d-flex align-center ga-2 mb-3">
      <span class="text-subtitle-2 font-weight-medium">��ǩ</span>
    </div>
    <div class="d-flex flex-wrap ga-1">
      <v-chip v-for="[tag, count] in allTags" :key="tag" size="x-small" class="tag-chip"
        @click="onTagClick(tag)"
        :color="selectedTag === tag ? 'primary' : undefined"
        :variant="selectedTag === tag ? 'flat' : 'outlined'">
        {{ tag }}
        <template #append><span class="text-caption opacity-75">{{ count }}</span></template>
      </v-chip>
      <div v-if="!allTags.length" class="text-caption text-medium-emphasis py-2">���ޱ�ǩ</div>
    </div>
  </v-card>
</template>

