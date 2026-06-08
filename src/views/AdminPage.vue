<script setup lang="ts">
import { ref, onMounted, watch } from "vue"
import { useAuthStore } from "@/stores/auth"
import AvatarPicker from "@/components/AvatarPicker.vue"
import AppIconPicker from "@/components/AppIconPicker.vue"
import FaviconPicker from "@/components/FaviconPicker.vue"

const API = "/api"
const auth = useAuthStore()
const emit = defineEmits<{ back: [] }>()

const tab = ref("overview")
const stats = ref<null | any>(null)
const users = ref<any[]>([])
const loading = ref(false)
const deleting = ref<null | number>(null)
const nickError = ref("")
const oldPwd = ref("")
const newPwd = ref("")
const snackbar = ref(false)
const snackMsg = ref("")
const nickInput = ref(auth.userNickname)
const showAvatarPicker = ref(false)
const showAppIconPicker = ref(false)
const showFaviconPicker = ref(false)
const allowRegister = ref(true)
const siteTitle = ref("")
const siteIcp = ref("")

onMounted(() => {
  if (auth.userRole !== "admin") tab.value = "profile"
  else { loadData(); loadSettings() }
  nickInput.value = auth.userNickname
})

watch(() => auth.userRole, (val) => { if (val !== "admin") tab.value = "profile" })

async function loadData() {
  loading.value = true
  await Promise.all([loadStats(), loadUsers()])
  loading.value = false
}
async function loadStats() {
  try { const r = await fetch(API + "/admin/stats"); if (r.ok) stats.value = await r.json() } catch {}
}
async function loadUsers() {
  try { const r = await fetch(API + "/admin/users"); if (r.ok) users.value = await r.json() } catch {}
}
async function loadSettings() {
  try {
    const r = await fetch(API + "/settings")
    if (r.ok) {
      const s = await r.json()
      siteTitle.value = s.site_title || ""
      siteIcp.value = s.site_icp || ""
      document.title = s.site_title || "Mengji"
      allowRegister.value = s.allow_register !== "false"
    }
  } catch {}
}
async function saveSiteTitle() {
  try {
    await fetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "site_title", value: siteTitle.value.trim() })
    })
    document.title = siteTitle.value.trim() || "Mengji"
    snackMsg.value = "网站标题已保存"; snackbar.value = true
  } catch {}
}
async function saveSiteIcp() {
  try {
    await fetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "site_icp", value: siteIcp.value.trim() })
    })
    snackMsg.value = "备案号已保存"; snackbar.value = true
  } catch {}
}
async function toggleRegister(val: boolean) {
  try {
    await fetch(API + "/settings", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "allow_register", value: val ? "true" : "false" })
    })
    snackMsg.value = val ? "已允许注册" : "已关闭注册"; snackbar.value = true
  } catch {}
}
async function deleteUser(id: number) {
  if (!confirm("确定删除？")) return
  deleting.value = id
  try { await fetch(API + "/admin/users/" + id, { method: "DELETE" }); await loadData() } catch {}
  deleting.value = null
}
async function saveNickname() {
  nickError.value = ""
  if (!nickInput.value.trim()) return
  const err = await auth.updateNickname(nickInput.value)
  if (err) { nickError.value = err; return }
  snackMsg.value = "昵称已保存"; snackbar.value = true
}
async function savePassword() {
  if (!oldPwd.value || !newPwd.value || newPwd.value.length < 4) return
  try {
    const res = await fetch(API + "/auth/password", {
      method: "PATCH", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: auth.userName, oldPassword: oldPwd.value, newPassword: newPwd.value })
    })
    const result = await res.json()
    if (result.error) return
    oldPwd.value = ""; newPwd.value = ""
    snackMsg.value = "密码已修改"; snackbar.value = true
  } catch {}
}
function formatDate(ts: number) { return new Date(ts).toLocaleString("zh-CN") }
</script>

<template>
  <v-container fluid class="pa-6 admin-container" style="max-width:900px">
    <v-snackbar v-model="snackbar" :timeout="2000" location="top right" color="success" variant="tonal">
      {{ snackMsg }}
    </v-snackbar>
    <div class="d-flex align-center mb-4">
      <v-btn icon="mdi-arrow-left" variant="text" size="small" class="mr-2" @click="emit('back')" />
      <div>
        <h1 class="text-h4 font-weight-bold mb-1">后台管理</h1>
        <p class="text-body-2 text-medium-emphasis">管理用户与备忘录</p>
      </div>
      <v-spacer />
      <v-btn prepend-icon="mdi-refresh" variant="text" size="small" :loading="loading" @click="loadData">刷新</v-btn>
    </div>

    <v-tabs v-model="tab" color="primary" class="mb-4">
      <v-tab value="overview" v-if="auth.isAdmin"><v-icon start size="small">mdi-view-dashboard</v-icon>概览</v-tab>
      <v-tab value="system" v-if="auth.isAdmin"><v-icon start size="small">mdi-cog</v-icon>系统设置</v-tab>
      <v-tab value="users" v-if="auth.isAdmin"><v-icon start size="small">mdi-account-group</v-icon>用户管理</v-tab>
      <v-tab value="profile"><v-icon start size="small">mdi-account</v-icon>个人资料</v-tab>
    </v-tabs>

    <template v-if='tab === "overview" && auth.isAdmin'>
      <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
        <h3 class="text-subtitle-1 font-weight-medium mb-4">网站概览</h3>
        <div class="d-flex align-center justify-space-between py-3">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-account</v-icon>
            <span class="text-body-2">用户总数</span>
          </div>
          <span class="text-h5 font-weight-bold">{{ stats?.totalUsers || 0 }}</span>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between py-3">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-pencil-box-multiple</v-icon>
            <span class="text-body-2">备忘录总数</span>
          </div>
          <span class="text-h5 font-weight-bold">{{ stats?.totalNotes || 0 }}</span>
        </div>
      </v-card>
    </template>

    <template v-if='tab === "system" && auth.isAdmin'>
      <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
        <h3 class="text-subtitle-1 font-weight-medium mb-4">系统设置</h3>
        <div class="d-flex flex-column ga-4">
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-3">
              <v-icon color="primary">mdi-web</v-icon>
              <span class="text-body-2">网站标题</span>
            </div>
            <div class="d-flex align-center ga-2" style="flex:1;max-width:400px">
              <v-text-field v-model="siteTitle" variant="outlined" hide-details density="compact" placeholder="网站标题" style="width:100%" @keyup.enter="saveSiteTitle" />
              <v-btn size="small" variant="tonal" color="primary" @click="saveSiteTitle">保存</v-btn>
            </div>
          </div>
          <v-divider />
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-3">
              <v-icon color="primary">mdi-account-plus</v-icon>
              <span class="text-body-2">允许新用户注册</span>
            </div>
            <v-switch v-model="allowRegister" hide-details density="compact" @update:model-value="toggleRegister" color="primary" />
          </div>
          <v-divider />
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-3">
              <v-icon color="primary">mdi-certificate-outline</v-icon>
              <span class="text-body-2">备案号</span>
            </div>
            <div class="d-flex align-center ga-2" style="flex:1;max-width:400px">
              <v-text-field v-model="siteIcp" variant="outlined" hide-details density="compact" placeholder="沪ICP备xxxxxxxx号" style="width:100%" @keyup.enter="saveSiteIcp" />
              <v-btn size="small" variant="tonal" color="primary" @click="saveSiteIcp">保存</v-btn>
            </div>
          </div>
          <v-divider />
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-3">
              <v-icon color="primary">mdi-apps</v-icon>
              <span class="text-body-2">工具栏图标</span>
            </div>
            <div class="d-flex align-center ga-2">
              <v-btn size="small" variant="tonal" color="primary" @click="showAppIconPicker = true">修改</v-btn>
            </div>
          </div>
          <v-divider />
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-3">
              <v-icon color="primary">mdi-image-multiple</v-icon>
              <span class="text-body-2">网站图标 (Favicon)</span>
            </div>
            <div class="d-flex align-center ga-2">
              <v-btn size="small" variant="tonal" color="primary" @click="showFaviconPicker = true">设置</v-btn>
            </div>
          </div>
        </div>
      </v-card>
      <AppIconPicker v-model="showAppIconPicker" />
      <FaviconPicker v-model="showFaviconPicker" />
    </template>

    <template v-if='tab === "users" && auth.isAdmin'>
      <v-card variant="outlined" class="rounded-xl stat-card">
        <v-list lines="two" bg-color="transparent">
          <v-list-item v-for="user in users" :key="user.id">
            <template #prepend>
              <v-avatar color="primary" variant="tonal">
                <v-img v-if="user.avatar && (user.avatar.startsWith('/uploads/') || user.avatar.startsWith('http'))" :src="user.avatar" alt="" cover />
                <span v-else class="font-weight-medium">{{ (user.nickname || user.username).charAt(0).toUpperCase() }}</span>
              </v-avatar>
            </template>
            <v-list-item-title>{{ user.nickname || user.username }}</v-list-item-title>
            <v-list-item-subtitle>@{{ user.username }} - {{ formatDate(user.createdAt) }} - {{ user.memoCount }}条 -
              <v-chip size="x-small" :color="user.role === 'admin' ? 'primary' : 'default'" variant="tonal">
                {{ user.role === 'admin' ? '管理员' : '普通用户' }}
              </v-chip>
            </v-list-item-subtitle>
            <template #append>
              <v-btn v-if="user.username !== auth.userName" icon="mdi-delete" size="small" variant="text" color="error" :loading="deleting === user.id" @click="deleteUser(user.id)" />
            </template>
          </v-list-item>
        </v-list>
      </v-card>
    </template>

    <div class="text-center text-caption text-medium-emphasis pt-4">v1.0.0</div>
  </v-container>
</template>

<style scoped>
.stat-card { border-color: #424242 !important; }
@media (max-width: 768px) {
  .admin-container { padding: 12px !important; }
  .admin-container :deep(.v-tabs) { flex-wrap: nowrap; overflow-x: auto; }
  .admin-container :deep(.v-tab) { min-width: auto; padding: 0 12px; font-size: 0.8rem; }
}
</style>

