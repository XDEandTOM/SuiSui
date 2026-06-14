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

const serverConfig = ref({ version: "", port: "", tls: false, dataDir: "" })
const siteDomain = ref("")
const domainInput = ref("")
const showDomainDialog = ref(false)
const liveStreamUrl = ref("")
const liveInput = ref("")
const showLiveDialog = ref(false)
const certText = ref("")
const keyText = ref("")
const savingSSL = ref(false)
const brotliEnabled = ref(true)
const githubTokenConfigured = ref(false)
const githubTokenInput = ref("")
const showGitHubTokenDialog = ref(false)

async function loadBrotliConfig() {
  try {
    const r = await authFetch("/api/admin/config/brotli")
    if (r.ok) { const d = await r.json(); brotliEnabled.value = d.enabled }
  } catch { console.warn("loadBrotli failed") }
}

async function loadGitHubTokenStatus() {
  try {
    const r = await authFetch("/api/admin/config/github-token")
    if (r.ok) { const d = await r.json(); githubTokenConfigured.value = d.configured }
  } catch { console.warn("loadGitHubToken failed") }
}

async function saveGitHubToken() {
  try {
    await authFetch("/api/admin/config/github-token", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token: githubTokenInput.value })
    })
    githubTokenConfigured.value = !!githubTokenInput.value
    showGitHubTokenDialog.value = false
  } catch { alert("保存失败") }
}

async function toggleBrotli() {
  try {
    await authFetch("/api/admin/config/brotli", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: brotliEnabled.value })
    })
  } catch { console.warn("toggleBrotli failed") }
}

async function saveSSL() {
  if (!certText.value || !keyText.value) { alert("请填写证书和私钥内容"); return }
  savingSSL.value = true
  try {
    // Save cert/key as files via backend
    const fd = new FormData()
    fd.append("cert", new Blob([certText.value]), "cert.pem")
    fd.append("key", new Blob([keyText.value]), "key.pem")
    const res = await authFetch("/api/admin/config/ssl", { method: "POST", body: fd })
    if (res.ok) {
      await authFetch("/api/admin/restart", { method: "POST" })
      alert("配置已保存，服务器正在重启… 请稍等后刷新页面")
      setTimeout(() => location.reload(), 3000)
    } else {
      alert("SSL 配置保存失败")
    }
  } catch { alert("SSL 配置失败") }
  savingSSL.value = false
}

async function clearSSL() {
  await authFetch("/api/admin/config/ssl", { method: "DELETE" })
  await authFetch("/api/admin/restart", { method: "POST" })
  alert("TLS 已关闭，服务器正在重启…")
  setTimeout(() => location.reload(), 3000)
}

async function loadServerConfig() {
  try {
    const r = await authFetch("/api/admin/config")
    if (r.ok) serverConfig.value = await r.json()
  } catch { console.warn("loadServerConfig failed") }
}

async function loadSettings() {
  try {
    const r = await fetch(API + "/settings")
    if (r.ok) {
      const s = await r.json()
      siteTitle.value = s.site_title || ""
      siteIcp.value = s.site_icp || ""
      siteDomain.value = s.site_domain || ""
      liveStreamUrl.value = s.live_stream_url || ""
      document.title = s.site_title || "碎碎"
      allowRegister.value = s.allow_register !== "false"
    }
  } catch { console.warn("loadSettings failed") }
}

function openTitleDialog() { titleInput.value = siteTitle.value; showTitleDialog.value = true }
function openIcpDialog() { icpInput.value = siteIcp.value; showIcpDialog.value = true }
function openDomainDialog() { domainInput.value = siteDomain.value; showDomainDialog.value = true }

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
async function saveDomain() {
  try {
    await authFetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "site_domain", value: domainInput.value.trim() })
    })
    siteDomain.value = domainInput.value.trim()
    snackMsg.value = "域名已保存"; snackbar.value = true; showDomainDialog.value = false
  } catch { console.warn("saveDomain failed") }
}
async function saveLiveUrl() {
  try {
    await authFetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "live_stream_url", value: liveInput.value.trim() })
    })
    liveStreamUrl.value = liveInput.value.trim()
    snackMsg.value = "直播流地址已保存"; snackbar.value = true; showLiveDialog.value = false
  } catch { console.warn("saveLiveUrl failed") }
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
loadServerConfig()
loadBrotliConfig()
loadGitHubTokenStatus()
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
            <v-icon color="primary">mdi-domain</v-icon>
            <span class="text-body-2">域名</span>
          </div>
          <div class="d-flex align-center ga-2">
            <span class="text-body-2 text-medium-emphasis text-caption">{{ siteDomain || "未设置" }}</span>
            <v-btn size="small" variant="tonal" color="primary" @click="openDomainDialog">修改</v-btn>
          </div>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-video</v-icon>
            <span class="text-body-2">直播流地址</span>
          </div>
          <div class="d-flex align-center ga-2">
            <span class="text-body-2 text-medium-emphasis text-caption" style="max-width:240px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ liveStreamUrl || "未设置" }}</span>
            <v-btn size="small" variant="tonal" color="primary" @click="liveInput = liveStreamUrl; showLiveDialog = true">修改</v-btn>
          </div>
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

    <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
      <h3 class="text-subtitle-1 font-weight-medium mb-4">服务器</h3>
      <div class="d-flex flex-column ga-3">
        <div class="d-flex align-center ga-3">
          <v-chip size="x-small" variant="tonal" color="default" class="server-badge">{{ serverConfig.version || "—" }}</v-chip>
          <v-chip size="x-small" variant="tonal" :color="serverConfig.tls ? 'success' : 'default'" class="server-badge">
            {{ serverConfig.tls ? 'TLS' : '直连 HTTP' }}
          </v-chip>
        </div>
        <div class="d-flex align-center ga-2">
          <span class="text-body-2" style="min-width:100px">Brotli 压缩</span>
          <v-switch v-model="brotliEnabled" hide-details density="compact" color="primary" @update:model-value="toggleBrotli" />
        </div>
        <v-divider />
        <span class="text-subtitle-2 font-weight-medium">SSL 证书</span>
        <v-textarea v-model="certText" variant="outlined" hide-details density="compact" rows="3"
          placeholder="-----BEGIN CERTIFICATE-----" class="ssl-textarea" />
        <v-textarea v-model="keyText" variant="outlined" hide-details density="compact" rows="3"
          placeholder="-----BEGIN PRIVATE KEY-----" class="ssl-textarea" />
        <div class="d-flex align-center ga-2">
          <v-btn size="small" variant="flat" color="primary" :loading="savingSSL" @click="saveSSL">保存并重启</v-btn>
          <v-btn v-if="serverConfig.tls" size="small" variant="tonal" color="error" @click="clearSSL">关闭 TLS</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <span class="text-body-2">GitHub Token</span>
          <div class="d-flex align-center ga-2">
            <v-chip size="x-small" :color="githubTokenConfigured ? 'success' : 'default'" variant="tonal">
              {{ githubTokenConfigured ? '已配置' : '未配置' }}
            </v-chip>
            <v-btn size="small" variant="tonal" color="primary" @click="showGitHubTokenDialog = true">修改</v-btn>
          </div>
        </div>
      </div>
    </v-card>

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
    <!-- Domain Dialog -->
    <v-dialog v-model="showDomainDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">配置域名</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="domainInput" variant="outlined" hide-details density="compact" placeholder="https://suisui.example.com" autofocus @keyup.enter="saveDomain" />
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showDomainDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="saveDomain">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <!-- Live Stream URL Dialog -->
    <v-dialog v-model="showLiveDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">配置直播流地址</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="liveInput" variant="outlined" hide-details density="compact" placeholder="https://example.com/live/stream.m3u8" autofocus @keyup.enter="saveLiveUrl" />
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showLiveDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="saveLiveUrl">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <!-- GitHub Token Dialog -->
    <v-dialog v-model="showGitHubTokenDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">GitHub Token</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="githubTokenInput" variant="outlined" hide-details density="compact"
            placeholder="ghp_xxxxxxxxxxxx" autofocus @keyup.enter="saveGitHubToken" />
          <div class="text-caption text-medium-emphasis mt-2">用于提升 GitHub API 请求频率（60→5000 次/小时）。不会泄露到前端。</div>
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showGitHubTokenDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="saveGitHubToken">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style scoped>
.stat-card { border-color: #424242 !important; background: rgba(var(--v-theme-surface), 0.6); backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); }
.server-badge { height: 24px !important; font-size: 0.7rem !important; }
.ssl-textarea :deep(textarea) { font-size: 0.75rem !important; font-family: var(--code-font) !important; }
</style>
