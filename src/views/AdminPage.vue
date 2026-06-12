<script setup lang="ts">
import { ref, onMounted, watch } from "vue"
import { useAuthStore } from "@/stores/auth"
import { authFetch } from "@/utils/api"
import AdminProfile from "@/components/AdminProfile.vue"
import AdminSystem from "@/components/AdminSystem.vue"

const API = "/api"
const auth = useAuthStore()
const emit = defineEmits<{ back: [] }>()

interface AdminStats {
  totalUsers: number
  totalNotes: number
}
interface AdminUser {
  id: number
  username: string
  nickname: string
  role: string
  memoCount: number
  createdAt: number
}

const tab = ref("overview")
const stats = ref<AdminStats | null>(null)
const users = ref<AdminUser[]>([])
const loading = ref(false)
const deleting = ref<null | number>(null)

watch(tab, () => window.scrollTo(0, 0))

onMounted(() => {
  if (auth.userRole !== "admin") tab.value = "profile"
  else { loadData() }
})

watch(() => auth.userRole, (val) => { if (val !== "admin") tab.value = "profile" })

async function loadData() {
  loading.value = true
  await Promise.all([loadStats(), loadUsers()])
  loading.value = false
}
async function loadStats() {
  try { const r = await authFetch(API + "/admin/stats"); if (r.ok) stats.value = await r.json() } catch { console.warn("loadStats failed") }
}
const userPage = ref(1)
const userTotal = ref(0)
const userPerPage = ref(10)

async function loadUsers() {
  try { const r = await authFetch(API + "/admin/users?page=" + userPage.value + "&per_page=" + userPerPage.value); if (r.ok) { const d = await r.json(); users.value = d.users || []; userTotal.value = d.total || 0 } } catch { console.warn("loadUsers failed") }
}
function prevPage() { if (userPage.value > 1) { userPage.value--; loadUsers() } }
function nextPage() { if (userPage.value * userPerPage.value < userTotal.value) { userPage.value++; loadUsers() } }
async function deleteUser(id: number) {
  if (!confirm("确定删除？")) return
  deleting.value = id
  try { await authFetch(API + "/admin/users/" + id, { method: "DELETE" }); await loadData() } catch { console.warn("deleteUser failed") }
  deleting.value = null
}
function formatDate(ts: number) { return new Date(ts).toLocaleString("zh-CN") }
</script>

<template>
  <v-container fluid class="pa-6 admin-container" style="max-width:900px">
    <div class="d-flex align-center mb-4">
      <v-btn icon="mdi-arrow-left" variant="text" size="small" class="mr-2" @click="emit('back')" />
      <div>
        <h1 class="text-h4 font-weight-bold mb-1">后台管理</h1>
        <p class="text-body-2 text-medium-emphasis">管理用户与碎片笔记</p>
      </div>
      <v-spacer />
      <v-btn prepend-icon="mdi-refresh" variant="text" size="small" :loading="loading" @click="loadData">刷新</v-btn>
    </div>

    <v-tabs v-model="tab" color="primary" class="mb-4">
      <v-tab v-if="auth.isAdmin" value="overview"><v-icon start size="small">mdi-view-dashboard</v-icon>概览</v-tab>
      <v-tab v-if="auth.isAdmin" value="system"><v-icon start size="small">mdi-cog</v-icon>系统设置</v-tab>
      <v-tab v-if="auth.isAdmin" value="users"><v-icon start size="small">mdi-account-group</v-icon>用户管理</v-tab>
      <v-tab value="profile"><v-icon start size="small">mdi-account</v-icon>个人资料</v-tab>
    </v-tabs>

    <Transition name="fade" mode="out-in">
      <div :key="tab">
        <template v-if="tab === &quot;overview&quot; && auth.isAdmin">
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
            <span class="text-body-2">碎片笔记总数</span>
          </div>
          <span class="text-h5 font-weight-bold">{{ stats?.totalNotes || 0 }}</span>
        </div>
      </v-card>
    </template>

    <template v-if="tab === &quot;system&quot; && auth.isAdmin">
      <AdminSystem />
    </template>

    <template v-if="tab === &quot;users&quot; && auth.isAdmin">
      <v-card variant="outlined" class="rounded-xl pa-4 mb-4 stat-card">
        <h3 class="text-subtitle-1 font-weight-medium mb-4 px-2">用户管理</h3>
        <div v-if="loading" class="d-flex justify-center py-12">
          <v-progress-circular indeterminate color="primary" size="40" />
        </div>
        <div v-else-if="!users.length" class="text-center py-8 text-medium-emphasis text-body-2">暂无用户</div>
        <div v-else class="d-flex flex-column">
          <div v-for="u in users" :key="u.id" class="d-flex align-center pa-3 user-row">
            <div class="d-flex align-center ga-3 flex-grow-1" style="min-width:0">
              <v-avatar size="36" color="primary" variant="tonal">{{ u.nickname?.charAt(0)?.toUpperCase() || u.username.charAt(0).toUpperCase() }}</v-avatar>
              <div style="min-width:0">
                <div class="text-body-2 font-weight-medium text-truncate">{{ u.nickname || u.username }}</div>
                <div class="text-caption text-medium-emphasis">
                  @{{ u.username }} · {{ u.role === "admin" ? "管理员" : "用户" }} · {{ u.memoCount }} 条备忘
                </div>
              </div>
            </div>
            <div class="text-caption text-medium-emphasis mr-3">{{ formatDate(u.createdAt) }}</div>
            <v-btn v-if="u.role !== 'admin'" icon="mdi-delete" size="x-small" variant="text" color="error"
              :loading="deleting === u.id" @click="deleteUser(u.id)" />
          </div>
          <div v-if="users.length" class="d-flex align-center justify-center ga-3 pt-3 px-2">
            <v-btn size="small" variant="tonal" :disabled="userPage <= 1" @click="prevPage">
              <v-icon>mdi-chevron-left</v-icon> 上一页
            </v-btn>
            <span class="text-caption text-medium-emphasis">{{ userPage }} / {{ Math.ceil(userTotal / userPerPage) || 1 }}</span>
            <v-btn size="small" variant="tonal" :disabled="userPage * userPerPage >= userTotal" @click="nextPage">
              下一页 <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
          </div>
        </div>
      </v-card>
    </template>

    <template v-if="tab === &quot;profile&quot;">
      <AdminProfile />
    </template>
      </div>
    </Transition>
</v-container>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.stat-card { border-color: #424242 !important; }
.user-row { border-bottom: 1px solid rgba(var(--v-theme-on-surface),0.06); }
.user-row:last-child { border-bottom: none; }
@media (max-width: 768px) {
  .admin-container { padding: 12px !important; }
  .admin-container :deep(.v-tabs) { flex-wrap: nowrap; overflow-x: auto; }
  .admin-container :deep(.v-tab) { min-width: auto; padding: 0 12px; font-size: 0.8rem; }
}
</style>
