<script setup lang="ts">
import { ref, onMounted, watch } from "vue"
import { useDisplay } from "vuetify"
import { useAuthStore } from "@/stores/auth"
import NotesPage from "@/views/NotesPage.vue"

const AdminPage = defineAsyncComponent(() => import("@/views/AdminPage.vue"))
const LoginDialog = defineAsyncComponent(() => import("@/components/LoginDialog.vue"))

const { mobile } = useDisplay()
const isMobile = mobile

const auth = useAuthStore()
const showAdmin = ref(false)
const showLogin = ref(false)
const showMobileHeatmap = ref(false)

onMounted(() => { auth.init(); loadSiteTitle() })

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
          <v-img :src="auth.userAppIcon" width="28" height="28" class="sidebar-icon-img" />
        </template>
        <template v-else>
          <v-icon size="28" color="primary" class="mb-4">mdi-pencil-box-multiple</v-icon>
        </template>
      </div>
      <div class="sidebar-middle" />
      <div class="sidebar-bottom">
        <v-btn icon="mdi-theme-light-dark" variant="text" size="small" class="sidebar-btn" @click.stop="$vuetify.theme.cycle()" />
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
          <v-icon size="22" color="primary">mdi-pencil-box-multiple</v-icon>
        </template>
        <v-spacer />
        <v-btn icon="mdi-theme-light-dark" variant="text" size="small" class="mobile-bar-btn" @click.stop="$vuetify.theme.cycle()" />
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

    <v-main class="main-bg" :class="{ 'has-sidebar': !isMobile, 'has-bottom-bar': isMobile }">
      <AdminPage v-if="showAdmin" @back="showAdmin = false" />
      <NotesPage v-else :mobile-heatmap="showMobileHeatmap" @close-heatmap="showMobileHeatmap = false" />
    </v-main>
    <LoginDialog v-model="showLogin" />
  </v-app>
</template>

<style>
.main-bg { min-height: 100vh; background: rgb(var(--v-theme-background)); }
.main-bg.has-sidebar { margin-left: 56px; }
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
  width: 56px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background: rgb(var(--v-theme-surface));
  z-index: 100;
}
.sidebar-top {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.sidebar-middle { flex: 1; }
.sidebar-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  padding-bottom: 16px;
}
.sidebar-btn { opacity: 0.6; transition: opacity 0.2s; }
.sidebar-btn:hover { opacity: 1; }

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
.mobile-bar-btn { opacity: 0.6; transition: opacity 0.2s; }
.mobile-bar-btn:active { opacity: 1; }
</style>
