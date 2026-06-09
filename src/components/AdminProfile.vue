<script setup lang="ts">
import { ref } from "vue"
import { useAuthStore } from "@/stores/auth"
import AvatarPicker from "@/components/AvatarPicker.vue"

const auth = useAuthStore()
const emit = defineEmits<{ back: [] }>()

const snackbar = ref(false)
const snackMsg = ref("")
const nickInput = ref(auth.userNickname)
const nickError = ref("")
const showNickDialog = ref(false)
const showPwdDialog = ref(false)
const pwdOld = ref("")
const pwdNew = ref("")
const pwdConfirm = ref("")
const showAvatarPicker = ref(false)
const themeColorInput = ref(auth.userThemeColor)
const importing = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function onColorChange(e: Event) {
  themeColorInput.value = (e.target as HTMLInputElement).value
}

async function saveThemeColor() {
  await auth.updateThemeColor(themeColorInput.value)
  snackMsg.value = "主题色已保存"; snackbar.value = true
}

function openNickDialog() { nickInput.value = auth.userNickname; nickError.value = ""; showNickDialog.value = true }
function openPwdDialog() { pwdOld.value = ""; pwdNew.value = ""; pwdConfirm.value = ""; showPwdDialog.value = true }

async function saveNickname() {
  nickError.value = ""
  if (!nickInput.value.trim()) return
  const err = await auth.updateNickname(nickInput.value)
  if (err) { nickError.value = err; return }
  showNickDialog.value = false
  snackMsg.value = "昵称已保存"; snackbar.value = true
}
async function savePassword() {
  if (!pwdOld.value || !pwdNew.value || pwdNew.value.length < 4 || pwdNew.value !== pwdConfirm.value) return
  try {
    const res = await fetch("/api/auth/password", {
      method: "PATCH", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: auth.userName, oldPassword: pwdOld.value, newPassword: pwdNew.value })
    })
    const result = await res.json()
    if (result.error) return
    pwdOld.value = ""; pwdNew.value = ""; pwdConfirm.value = ""; showPwdDialog.value = false
    snackMsg.value = "密码已修改"; snackbar.value = true
  } catch {}
}

async function exportNotes() {
  try {
    const res = await fetch(`/api/notes/export?username=${auth.userName}`)
    if (!res.ok) { snackMsg.value = "导出失败"; snackbar.value = true; return }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url; a.download = `suisui-notes-${auth.userName}-${Date.now()}.json`
    a.click(); URL.revokeObjectURL(url)
    snackMsg.value = "导出成功"; snackbar.value = true
  } catch { snackMsg.value = "导出失败"; snackbar.value = true }
}

async function importNotes(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importing.value = true
  try {
    const text = await file.text()
    const notes = JSON.parse(text)
    const res = await fetch("/api/notes/import", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify(notes),
    })
    if (res.ok) {
      const data = await res.json()
      snackMsg.value = `成功导入 ${data.imported} 条备忘录`; snackbar.value = true
    } else { snackMsg.value = "导入失败"; snackbar.value = true }
  } catch { snackMsg.value = "文件格式错误"; snackbar.value = true }
  importing.value = false
  input.value = ""
}
</script>

<template>
  <v-container fluid class="pa-0">
    <v-snackbar v-model="snackbar" :timeout="2000" location="top right" color="success" variant="tonal">
      {{ snackMsg }}
    </v-snackbar>

    <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
      <h3 class="text-subtitle-1 font-weight-medium mb-4">个人资料</h3>
      <div class="d-flex flex-column ga-4">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-account-circle</v-icon>
            <span class="text-body-2">头像</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="showAvatarPicker = true">修改</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-card-account-details</v-icon>
            <span class="text-body-2">昵称</span>
          </div>
          <div class="d-flex align-center ga-2">
            <span class="text-body-2 text-medium-emphasis">{{ auth.userNickname || "未设置" }}</span>
            <v-btn size="small" variant="tonal" color="primary" @click="openNickDialog">修改</v-btn>
          </div>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-lock</v-icon>
            <span class="text-body-2">密码</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="openPwdDialog">修改</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-palette</v-icon>
            <span class="text-body-2">主题色</span>
          </div>
          <div class="d-flex align-center ga-2">
            <input type="color" :value="auth.userThemeColor" @input="onColorChange" class="theme-picker" />
            <v-btn size="small" variant="tonal" color="primary" @click="saveThemeColor">保存</v-btn>
          </div>
        </div>
      </div>
    </v-card>

    <v-card variant="outlined" class="rounded-xl pa-6 mb-4 stat-card">
      <h3 class="text-subtitle-1 font-weight-medium mb-4">数据管理</h3>
      <div class="d-flex flex-column ga-4">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-export</v-icon>
            <span class="text-body-2">导出我的备忘录</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" @click="exportNotes">导出</v-btn>
        </div>
        <v-divider />
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-3">
            <v-icon color="primary">mdi-import</v-icon>
            <span class="text-body-2">导入备忘录</span>
          </div>
          <v-btn size="small" variant="tonal" color="primary" :loading="importing" @click="fileInput?.click()">导入</v-btn>
        </div>
        <input ref="fileInput" type="file" accept=".json" hidden @change="importNotes" />
      </div>
    </v-card>

    <AvatarPicker v-model="showAvatarPicker" />

    <!-- Nickname Dialog -->
    <v-dialog v-model="showNickDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">修改昵称</v-card-title>
        <v-card-text class="px-0">
          <v-text-field v-model="nickInput" variant="outlined" hide-details density="compact" placeholder="设置昵称" @keyup.enter="saveNickname" autofocus />
          <div v-if="nickError" class="text-caption text-error mt-1">{{ nickError }}</div>
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showNickDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="saveNickname">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Password Dialog -->
    <v-dialog v-model="showPwdDialog" max-width="400">
      <v-card class="rounded-xl pa-4">
        <v-card-title class="text-subtitle-1 font-weight-medium px-0">修改密码</v-card-title>
        <v-card-text class="px-0 d-flex flex-column ga-3">
          <v-text-field v-model="pwdOld" type="password" variant="outlined" hide-details density="compact" placeholder="旧密码" autofocus />
          <v-text-field v-model="pwdNew" type="password" variant="outlined" hide-details density="compact" placeholder="新密码（至少4位）" />
          <v-text-field v-model="pwdConfirm" type="password" variant="outlined" hide-details density="compact" placeholder="确认新密码" />
        </v-card-text>
        <v-card-actions class="px-0">
          <v-spacer />
          <v-btn variant="text" @click="showPwdDialog = false">取消</v-btn>
          <v-btn variant="tonal" color="primary" @click="savePassword">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style scoped>
.stat-card { border-color: #424242 !important; }
.theme-picker { width: 36px; height: 36px; border: none; border-radius: 50%; cursor: pointer; padding: 0; background: none; }
.theme-picker::-webkit-color-swatch-wrapper { padding: 0; }
.theme-picker::-webkit-color-swatch { border: 2px solid rgba(var(--v-theme-on-surface), 0.15); border-radius: 50%; }
</style>
