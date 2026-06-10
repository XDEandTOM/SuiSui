<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue"
import { useDisplay, useTheme } from "vuetify"
import { useAuthStore } from "@/stores/auth"
import NotesPage from "@/views/NotesPage.vue"

import AdminPage from "@/views/AdminPage.vue"
import LoginDialog from "@/components/LoginDialog.vue"
import AppLogo from "@/components/AppLogo.vue"

const { mobile } = useDisplay()
const isMobile = mobile

const auth = useAuthStore()
const showAdmin = ref(false)
const showLogin = ref(false)
const showMobileHeatmap = ref(false)

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
  } catch { /* ignore */ }
}

function toggleTheme() {
  const next = vuetifyTheme.global.name.value === "dark" ? "light" : "dark"
  vuetifyTheme.change(next)
}

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
</script>

<template>
  <v-app>
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
        <v-avatar size="36" color="primary" variant="tonal" class="sidebar-avatar">
          <span class="sidebar-avatar-text">{{ avatarLetter }}</span>
        </v-avatar>
        <span class="sidebar-username">{{ displayName }}</span>
      </div>

      <div class="sidebar-middle" />
      <div class="sidebar-bottom">
        <v-btn icon="mdi-theme-light-dark" variant="text" size="small" class="sidebar-btn" @click.stop="toggleTheme" />
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
        <template v-if="auth.isLoggedIn && auth.userAppIcon">
          <v-img :src="auth.userAppIcon" width="22" height="22" class="mobile-bar-icon" />
        </template>
        <template v-else>
          <AppLogo :size="22" />
        </template>
        <v-spacer />
        <v-btn icon="mdi-theme-light-dark" variant="text" size="small" class="mobile-bar-btn" @click.stop="toggleTheme" />
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
  </v-app>
</template>

<style>
.main-bg { min-height: 100vh; background: rgb(var(--v-theme-background)); }
.main-bg.has-sidebar { margin-left: 64px; }
.main-bg.has-bottom-bar { margin-left: 0; padding-bottom: 56px; }
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-thumb { background: rgba(var(--v-theme-on-surface), 0.15); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(var(--v-theme-on-surface), 0.25); }
.v-main { transition: margin-left 0.3s ease, padding-bottom 0.3s ease; }
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
  background: rgb(var(--v-theme-surface));
  z-index: 100;
  gap: 4px;
}
.sidebar-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4px 0 8px;
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
  transition: opacity 0.2s, transform 0.15s;
}
.sidebar-btn:hover { opacity: 1; transform: scale(1.1); }

.mobile-bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 56px;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background: rgb(var(--v-theme-surface));
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
</style>
