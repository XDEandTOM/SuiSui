<script setup lang="ts">
import { ref } from "vue"
import { useTheme } from "vuetify"

const THEME_PRESET_KEY = "suisui-theme-preset"
const THEME_KEY = "suisui-theme"

interface Preset { name: string; emoji: string; light: Record<string, string>; dark: Record<string, string> }
const presets: Record<string, Preset> = {
  default: { name: "默认", emoji: "🎨", light: { primary: "#1976D2" }, dark: { primary: "#1976D2" } },
  sunset:  { name: "暖阳", emoji: "🌅", light: { primary: "#E65100" }, dark: { primary: "#FF8A65" } },
  forest:  { name: "森林", emoji: "🌲", light: { primary: "#2E7D32" }, dark: { primary: "#66BB6A" } },
  ocean:   { name: "海洋", emoji: "🌊", light: { primary: "#00695C" }, dark: { primary: "#4DB6AC" } },
  lavender:{ name: "薰衣草", emoji: "🌸", light: { primary: "#7B1FA2" }, dark: { primary: "#CE93D8" } },
  starry:  { name: "星空", emoji: "🌌", light: { primary: "#4A148C" }, dark: { primary: "#B39DDB" } },
  aurora:  { name: "极光", emoji: "💚", light: { primary: "#1B5E20" }, dark: { primary: "#81C784" } },
  sakura:  { name: "樱花", emoji: "🌸", light: { primary: "#AD1457" }, dark: { primary: "#F48FB1" } },
  graphite:{ name: "石墨", emoji: "🪨", light: { primary: "#37474F" }, dark: { primary: "#78909C" } },
}

const visible = defineModel<boolean>("modelValue", { required: true })
const vuetify = useTheme()
const current = ref(localStorage.getItem(THEME_PRESET_KEY) || "default")
const isDark = ref(vuetify.global.name.value === "dark")

function pick(id: string) {
  const p = presets[id]
  if (!p) return
  current.value = id
  localStorage.setItem(THEME_PRESET_KEY, id)
  const theme = vuetify.global.name.value as "light" | "dark"
  Object.assign(vuetify.themes.value[theme].colors, p[theme])
  visible.value = false
}

function toggleDark() {
  const theme = isDark.value ? "dark" : "light"
  vuetify.global.name.value = theme
  localStorage.setItem(THEME_KEY, theme)
  const p = presets[current.value]
  if (p) Object.assign(vuetify.themes.value[theme].colors, p[theme])
}
</script>

<template>
  <v-dialog :model-value="visible" max-width="370" @update:model-value="v => visible = v">
    <v-card class="rounded-xl pa-4">
      <div class="d-flex align-center mb-3">
        <span class="text-subtitle-2 font-weight-medium">主题配色</span>
        <v-spacer />
        <v-btn icon="mdi-close" size="x-small" variant="text" @click="visible = false" />
      </div>
      <div class="theme-grid">
        <div v-for="(p, id) in presets" :key="id" class="theme-card"
          :class="{ active: current === id }"
          @click="pick(id)">
          <div class="theme-preview" :style="{ background: p.light.primary }">
            <span class="theme-emoji">{{ p.emoji }}</span>
          </div>
          <span class="theme-name">{{ p.name }}</span>
        </div>
      </div>
      <v-divider class="my-3" />
      <div class="d-flex align-center justify-space-between px-1">
        <span class="text-body-2">外观</span>
        <div class="mode-toggle">
          <button class="mode-btn" :class="{ active: !isDark }" @click="isDark = false; toggleDark()">☀️ 浅色</button>
          <button class="mode-btn" :class="{ active: isDark }" @click="isDark = true; toggleDark()">🌙 深色</button>
        </div>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.theme-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
.theme-card {
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  padding: 10px 6px; border-radius: 10px; cursor: pointer;
  border: 2px solid transparent; transition: all 0.15s;
}
.theme-card:hover { background: rgba(128,128,128,0.04); }
.theme-card.active { border-color: rgb(var(--v-theme-primary)); background: rgba(var(--v-theme-primary), 0.06); }
.theme-preview {
  width: 36px; height: 36px; border-radius: 8px;
  display: flex; align-items: center; justify-content: center;
}
.theme-emoji { font-size: 1.1rem; }
.theme-name { font-size: 0.72rem; font-weight: 500; }
.mode-toggle { display: flex; background: rgba(128,128,128,0.08); border-radius: 8px; padding: 2px; }
.mode-btn { border: none; background: transparent; padding: 4px 12px; border-radius: 6px; font-size: 0.78rem; cursor: pointer; transition: all 0.15s; color: rgba(var(--v-theme-on-surface), 0.5); }
.mode-btn.active { background: rgb(var(--v-theme-surface)); color: rgb(var(--v-theme-on-surface)); box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
</style>
