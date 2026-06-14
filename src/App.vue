<script setup lang="ts">
import { ref, onMounted, watch, computed, defineAsyncComponent } from "vue"
import { useDisplay, useTheme } from "vuetify"
import { useAuthStore } from "@/stores/auth"

const NotesPage = defineAsyncComponent(() => import("@/views/NotesPage.vue"))
const AdminPage = defineAsyncComponent(() => import("@/views/AdminPage.vue"))
import LoginDialog from "@/components/LoginDialog.vue"
import AppLogo from "@/components/AppLogo.vue"
import ShareView from "@/components/ShareView.vue"
import ThemePicker from "@/components/ThemePicker.vue"

const isShareView = computed(() => window.location.pathname.startsWith("/share/"))

const { mobile } = useDisplay()
const isMobile = mobile

const auth = useAuthStore()
const showAdmin = ref(false)
const showLogin = ref(false)
const showProfile = ref(false)
const showThemePicker = ref(false)
const showMobileHeatmap = ref(false)
const nickEdit = ref("")
const savingNick = ref(false)

const THEME_KEY = "suisui-theme"

const vuetifyTheme = useTheme()

const displayName = computed(() => auth.userNickname?.trim() || auth.userName || "")
const avatarLetter = computed(() => displayName.value.charAt(0).toUpperCase() || "?")

onMounted(async () => {
  const saved = localStorage.getItem(THEME_KEY)
  await auth.init()
  await loadSiteTitle()
  applyThemeColor(auth.userThemeColor)
  if (saved === "dark" || saved === "light") vuetifyTheme.change(saved)
})

async function loadSiteTitle() {
  try {
    const r = await fetch("/api/settings")
    if (r.ok) {
      const s = await r.json()
      if (s.site_title) document.title = s.site_title
      if (s.site_favicon) {
        const link = document.querySelector("link[rel=\"icon\"]") || document.createElement("link")
        link.setAttribute("rel", "icon")
        link.setAttribute("href", s.site_favicon)
        document.head.appendChild(link)
      }
    }
  } catch { console.warn("failed silently") }
}

function goLive() { window.open("/live/", "_self") }

function saveNickname() {
  const n = nickEdit.value.trim()
  if (!n || savingNick.value) return
  savingNick.value = true
  auth.updateNickname(n).then((err) => {
    if (err) alert(err)
    savingNick.value = false
    showProfile.value = false
  }).catch(() => { savingNick.value = false })
}

watch(showProfile, (v) => { if (v) nickEdit.value = auth.userNickname || "" })

function applyThemeColor(color: string) {
  if (color && color !== "#1976D2") {
    vuetifyTheme.themes.value.light.colors.primary = color
    vuetifyTheme.themes.value.dark.colors.primary = color
  }
}

watch(() => auth.userThemeColor, (color) => {
  if (color) applyThemeColor(color)
})

watch(() => vuetifyTheme.global.name.value, (val) => {
  if (val === "light" || val === "dark") localStorage.setItem(THEME_KEY, val)
}, { immediate: false })

watch([() => auth.isLoggedIn, () => auth.userRole], () => {
  if (!auth.isLoggedIn || auth.userRole !== "admin") showAdmin.value = false
})

function isImage(val?: string) { return val?.startsWith("/uploads/") || val?.startsWith("http") }
</script>

<template>
  <ShareView v-if="isShareView" />
  <v-app v-else>
    <!-- Desktop sidebar -->
    <div v-if="!isMobile" class="sidebar">
      <div class="sidebar-top">
        <template v-if="auth.isLoggedIn && auth.userAppIcon">
          <v-img :src="auth.userAppIcon" width="28" height="28" class="sidebar-app-icon" />
        </template>
        <template v-else>
          <AppLogo :size="28" />
        </template>
      </div>

      <!-- User avatar & name in sidebar -->
      <div v-if="auth.ready && auth.isLoggedIn" class="sidebar-user">
        <div v-if="isImage(auth.userAvatar)" class="sidebar-avatar-img" @click="showProfile = true">
          <img :src="auth.userAvatar" alt="" width="36" height="36" style="border-radius:10px;object-fit:cover;cursor:pointer" />
        </div>
        <v-avatar v-else size="36" color="primary" variant="tonal" class="sidebar-avatar" style="cursor:pointer"
          @click="showProfile = true">
          <span class="sidebar-avatar-text">{{ avatarLetter }}</span>
        </v-avatar>
        <span class="sidebar-username">{{ displayName }}</span>
      </div>

      <div class="sidebar-middle" />
      <div class="sidebar-bottom">
        <v-btn icon="mdi-palette-outline" variant="text" size="small" class="sidebar-btn" title="主题配色" @click.stop="showThemePicker = true" />
        <v-btn icon="mdi-video-outline" variant="text" size="small" class="sidebar-btn" title="直播" @click.stop="goLive()" />
        <template v-if="auth.ready && auth.isLoggedIn">
          <v-btn icon="mdi-cog-outline" variant="text" size="small" class="sidebar-btn"
            :color="showAdmin ? 'primary' : undefined"
            @click.stop="showAdmin = !showAdmin" />
          <v-btn icon="mdi-logout" variant="text" size="small" class="sidebar-btn" @click.stop="auth.logout()" />
        </template>
        <v-btn v-else icon="mdi-login" variant="text" size="small" class="sidebar-btn"
          @click.stop="showLogin = true" />
      </div>
    </div>

    <!-- Mobile bottom bar -->
    <div v-if="isMobile" class="mobile-bottom-bar">
      <div class="mobile-bar-inner">
        <template v-if="auth.ready && auth.isLoggedIn">
          <div class="d-flex align-center ga-1" style="cursor:pointer" @click="showProfile = true">
            <div v-if="isImage(auth.userAvatar)" class="mobile-avatar">
              <img :src="auth.userAvatar" alt="" width="26" height="26" style="border-radius:6px;object-fit:cover" />
            </div>
            <div v-else class="mobile-avatar-fallback">{{ avatarLetter }}</div>
          </div>
        </template>
        <template v-else>
          <AppLogo :size="22" />
        </template>
        <v-spacer />
        <v-btn icon="mdi-palette-outline" variant="text" size="small" class="mobile-bar-btn" @click.stop="showThemePicker = true" />
        <v-btn icon="mdi-video-outline" variant="text" size="small" class="mobile-bar-btn" @click.stop="goLive()" />
        <v-btn icon="mdi-fire" variant="text" size="small" class="mobile-bar-btn"
          :color="showMobileHeatmap ? 'primary' : undefined"
          @click.stop="showMobileHeatmap = !showMobileHeatmap" />
        <template v-if="auth.ready && auth.isLoggedIn">
          <v-btn icon="mdi-cog-outline" variant="text" size="small" class="mobile-bar-btn"
            :color="showAdmin ? 'primary' : undefined"
            @click.stop="showAdmin = !showAdmin" />
          <v-btn icon="mdi-logout" variant="text" size="small" class="mobile-bar-btn" @click.stop="auth.logout()" />
        </template>
        <v-btn v-else icon="mdi-login" variant="text" size="small" class="mobile-bar-btn"
          @click.stop="showLogin = true" />
      </div>
    </div>

    <v-main v-if="auth.ready" class="main-bg" :class="{ 'has-sidebar': !isMobile, 'has-bottom-bar': isMobile }">
      <AdminPage v-if="showAdmin" @back="showAdmin = false" />
      <NotesPage v-else :mobile-heatmap="showMobileHeatmap" @close-heatmap="showMobileHeatmap = false" />
    </v-main>
    <v-main v-else class="main-bg d-flex align-center justify-center" :class="{ 'has-sidebar': !isMobile, 'has-bottom-bar': isMobile }">
      <v-progress-circular indeterminate color="primary" />
    </v-main>
    <LoginDialog v-model="showLogin" />
    <ThemePicker v-model="showThemePicker" />
        <v-dialog v-model="showProfile" max-width="420" transition="scale-transition">
      <v-card class="rounded-xl pa-4">
        <div class="d-flex align-center mb-3">
          <span class="text-subtitle-2 font-weight-medium">个人资料</span>
          <v-spacer />
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="showProfile = false" />
        </div>
        <div class="d-flex flex-column ga-3">
          <v-text-field v-model="nickEdit" label="昵称" variant="outlined" hide-details density="compact"
            :disabled="savingNick" @keydown.enter="saveNickname" />
          <v-btn color="primary" variant="flat" size="small" class="rounded-pill align-self-start"
            :loading="savingNick" @click="saveNickname">
保存昵称
</v-btn>
        </div>
      </v-card>
    </v-dialog>
  </v-app>
</template>

<style>
/* Font variable for code blocks */
:root { --code-font: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace; }
html, body { font-family: "Maple Mono NF CN", -apple-system, BlinkMacSystemFont, sans-serif; }
::selection { background: rgba(var(--v-theme-primary), 0.25); }
.main-bg {
  min-height: 100vh;
  transition: background 0.3s ease;
  background:
    radial-gradient(ellipse at 20% 50%, rgba(var(--v-theme-primary), 0.06) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 20%, rgba(var(--v-theme-primary), 0.04) 0%, transparent 50%),
    rgb(var(--v-theme-background));
}
.main-bg.has-sidebar { margin-left: 64px; }
.main-bg.has-bottom-bar { margin-left: 0; padding-bottom: 56px; }
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-thumb { background: rgba(var(--v-theme-on-surface), 0.15); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(var(--v-theme-on-surface), 0.25); }
.v-main { transition: margin-left 0.3s ease, padding-bottom 0.3s ease; }
/* Dialog glass background */
.v-dialog > .v-card:not(.v-card--flat) { background: rgba(var(--v-theme-surface), 0.88) !important; backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px); }


</style>

<style scoped>
.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  width: 64px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 0;
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background: rgba(var(--v-theme-surface), 0.3);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  z-index: 100;
  gap: 2px;
}
.sidebar-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4px 0 10px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
  width: 40px;
}
.sidebar-app-icon { border-radius: 8px; }
.sidebar-user {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 0;
}
.sidebar-avatar {
  cursor: pointer;
  transition: transform 0.2s;
}
.sidebar-avatar-img { width: 36px; height: 36px; border-radius: 10px; overflow: hidden; box-shadow: 0 2px 8px rgba(var(--v-theme-primary), 0.15); cursor: pointer; transition: transform 0.2s; }
.sidebar-avatar-img:hover { transform: scale(1.1); }
.sidebar-avatar:hover { transform: scale(1.1); }
.sidebar-avatar-text { font-size: 0.85rem; font-weight: 600; color: rgb(var(--v-theme-on-surface)); }
.sidebar-username {
  font-size: 0.6rem;
  color: rgba(var(--v-theme-on-surface), 0.55);
  max-width: 56px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: center;
}
.sidebar-middle { flex: 1; }
.sidebar-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-bottom: 12px;
}
.sidebar-btn {
  opacity: 0.55;
  transition: opacity 0.2s, transform 0.15s, background 0.15s;
  position: relative; border-radius: 10px;
}
.sidebar-btn:hover { opacity: 1; transform: scale(1.1); background: rgba(var(--v-theme-primary), 0.06); }

.mobile-bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 56px;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background: rgba(var(--v-theme-surface), 0.6);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  z-index: 100;
  padding: 0 12px;
}
.mobile-bar-inner {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 4px;
}
.mobile-bar-icon { border-radius: 6px; }
.mobile-bar-btn { opacity: 0.55; transition: opacity 0.2s; }
.mobile-bar-btn:active { opacity: 1; }
.mobile-avatar { display: flex; align-items: center; }
.mobile-avatar-fallback {
  width: 26px; height: 26px; border-radius: 6px; display: flex;
  align-items: center; justify-content: center;
  background: rgb(var(--v-theme-primary)); color: #fff;
  font-size: 0.75rem; font-weight: 600; flex-shrink: 0;
}
</style>
