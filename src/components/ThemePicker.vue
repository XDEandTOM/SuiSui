<script setup lang="ts">
import { ref, watch } from "vue"
import { useTheme } from "vuetify"

const THEME_PRESET_KEY = "suisui-theme-preset"
const FONT_KEY = "suisui-font"

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
const currentFont = ref(localStorage.getItem(FONT_KEY) || "sans")
const fonts = [
  { id: "sans",   name: "默认无衬线",   css: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif" },
  { id: "serif",  name: "衬线体",      css: "'Georgia', 'Noto Serif SC', 'Source Han Serif SC', serif" },
  { id: "mono",   name: "等宽体",      css: "'JetBrains Mono', 'Fira Code', 'Consolas', 'Courier New', monospace" },
  { id: "round",  name: "圆体",        css: "'Rounded Mplus 1c', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif" },
  { id: "kai",    name: "楷体",        css: "'STKaiti', 'KaiTi', 'Noto Serif SC', serif" },
]

function applyFont(id: string) {
  const f = fonts.find(f => f.id === id)
  if (!f) return
  currentFont.value = id
  localStorage.setItem(FONT_KEY, id)
  document.documentElement.style.fontFamily = f.css
  // Set code font too
  document.documentElement.style.setProperty("--code-font", id === "mono" ? f.css : "'JetBrains Mono', 'Fira Code', 'Consolas', monospace")
}

function pick(id: string) {
  const p = presets[id]
  if (!p) return
  current.value = id
  localStorage.setItem(THEME_PRESET_KEY, id)
  const theme = vuetify.global.name.value as "light" | "dark"
  Object.assign(vuetify.themes.value[theme].colors, p[theme])
}

// Restore font on mount
const font = fonts.find(f => f.id === currentFont.value)
if (font) {
  document.documentElement.style.fontFamily = font.css
}
</script>

<template>
  <v-dialog :model-value="visible" @update:model-value="v => visible = v" max-width="370">
    <v-card class="rounded-xl pa-4">
      <div class="d-flex align-center mb-3">
        <span class="text-subtitle-2 font-weight-medium">主题与字体</span>
        <v-spacer />
        <v-btn icon="mdi-close" size="x-small" variant="text" @click="visible = false" />
      </div>
      <div class="section-label">配色方案</div>
      <div class="theme-grid mb-4">
        <div v-for="(p, id) in presets" :key="id" class="theme-card"
          :class="{ active: current === id }"
          @click="pick(id); visible = false">
          <div class="theme-preview" :style="{ background: p.light.primary }">
            <span class="theme-emoji">{{ p.emoji }}</span>
          </div>
          <span class="theme-name">{{ p.name }}</span>
        </div>
      </div>
      <div class="section-label">字体</div>
      <div class="d-flex flex-column ga-1">
        <div v-for="f in fonts" :key="f.id" class="font-option"
          :class="{ active: currentFont === f.id }"
          @click="applyFont(f.id); visible = false">
          <span class="font-sample" :style="{ fontFamily: f.css }">{{ f.name }}</span>
        </div>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.section-label { font-size: 0.78rem; font-weight: 600; opacity: 0.6; margin-bottom: 6px; }
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
.font-option {
  padding: 8px 12px; border-radius: 8px; cursor: pointer;
  border: 1px solid transparent; transition: all 0.15s;
}
.font-option:hover { background: rgba(128,128,128,0.04); }
.font-option.active { border-color: rgba(var(--v-theme-primary), 0.3); background: rgba(var(--v-theme-primary), 0.04); }
.font-sample { font-size: 0.88rem; }
</style>
