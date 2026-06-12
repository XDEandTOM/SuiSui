<script setup lang="ts">
import { ref } from "vue"
import { authFetch } from "@/utils/api"
import AppIconPicker from "@/components/AppIconPicker.vue"
import FaviconPicker from "@/components/FaviconPicker.vue"

const API = "/api"
const allowRegister = ref(true)
const siteTitle = ref("")
const siteIcp = ref("")
const snackbar = ref(false)
const snackMsg = ref("")
const showTitleDialog = ref(false)
const showIcpDialog = ref(false)
const titleInput = ref("")
const icpInput = ref("")
const showAppIconPicker = ref(false)
const showFaviconPicker = ref(false)

async function loadSettings() {
  try {
    const r = await fetch(API + "/settings")
    if (r.ok) {
      const s = await r.json()
      siteTitle.value = s.site_title || ""
      siteIcp.value = s.site_icp || ""
      document.title = s.site_title || "碎碎"
      allowRegister.value = s.allow_register !== "false"
    }
  } catch { console.warn("loadSettings failed") }
}

function openTitleDialog() { titleInput.value = siteTitle.value; showTitleDialog.value = true }
function openIcpDialog() { icpInput.value = siteIcp.value; showIcpDialog.value = true }

async function saveSiteTitle() {
  try {
    await authFetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "site_title", value: siteTitle.value.trim() })
    })
    document.title = siteTitle.value.trim() || "碎碎"
    snackMsg.value = "网站标题已保存"; snackbar.value = true; showTitleDialog.value = false
  } catch { console.warn("saveSiteTitle failed") }
}
async function saveSiteIcp() {
  try {
    await authFetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "site_icp", value: siteIcp.value.trim() })
    })
    snackMsg.value = "备案号已保存"; snackbar.value = true; showIcpDialog.value = false
  } catch { console.warn("saveSiteIcp failed") }
}
async function toggleRegister(val: boolean) {
  try {
    await authFetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "allow_register", value: val ? "true" : "false" })
    })
    snackMsg.value = val ? "已允许注册" : "已关闭注册"; snackbar.value = true
  } catch { console.warn("toggleRegister failed") }
}

// Load on mount
loadSettings()
</script>

<template>
  <v-container fluid class="pa-0">
    <v-snackbar v-model="snackbar" :timeout="2000" location="top right" color="success" variant="tonal">
      {{ snackMsg }}
    </v-snackbar>

    <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
      <h3 class="text-subtitle-1 font-weight-medium mb-4">系统设置</h3>
      <div class="d-flex flex-column ga-4">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-web</v-icon>
            <span class="text-body-2">网站标题</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="openTitleDialog">修改</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-account-plus</v-icon>
            <span class="text-body-2">允许新用户注册</span>
          </div>
          <v-switch v-model="allowRegister" hide-details density="compact" color="primary" @update:model-value="(val: boolean | null) => toggleRegister(val ?? false)" />
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-information</v-icon>
            <span class="text-body-2">备案号</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="openIcpDialog">修改</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-application</v-icon>
            <span class="text-body-2">图标</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="showAppIconPicker = true">修改</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-star</v-icon>
            <span class="text-body-2">Favicon</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="showFaviconPicker = true">修改</v-btn>
        </div>
      </div>
    </v-card>

    <AppIconPicker v-model="showAppIconPicker" />
    <FaviconPicker v-model="showFaviconPicker" />

    <!-- Title Dialog -->
    <v-dialog v-model="showTitleDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">修改网站标题</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="titleInput" variant="outlined" hide-details density="compact" placeholder="网站标题" autofocus @keyup.enter="saveSiteTitle" />
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showTitleDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="siteTitle = titleInput; saveSiteTitle()">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- ICP Dialog -->
    <v-dialog v-model="showIcpDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">修改备案号</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="icpInput" variant="outlined" hide-details density="compact" placeholder="沪ICP备xxxxxxxx号" autofocus @keyup.enter="saveSiteIcp" />
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showIcpDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="siteIcp = icpInput; saveSiteIcp()">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style scoped>
.stat-card { border-color: #424242 !important; background: rgba(var(--v-theme-surface), 0.6); backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); }
</style>
