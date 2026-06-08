import { ref, computed } from "vue"
import { defineStore } from "pinia"

const API = "/api"
const AUTH_KEY = "mengji-auth"
const USER_KEY = "mengji-user"
const AVATAR_KEY = "mengji-avatar"
const NICK_KEY = "mengji-nick"

function clearStorage() {
  ;[AUTH_KEY, USER_KEY, AVATAR_KEY, NICK_KEY].forEach(k => localStorage.removeItem(k))
}

export const useAuthStore = defineStore("auth", () => {
  const isLoggedIn = ref(false)
  const userName = ref("")
  const userAvatar = ref("")
  const userNickname = ref("")
  const userAppIcon = ref("")
  const userThemeColor = ref("#1976D2")
const userRole = ref("user")
  const isAdmin = computed(() => userRole.value === "admin")
  const ready = ref(false)

  async function init() {
    const storedUser = localStorage.getItem(USER_KEY)
    const storedAuth = localStorage.getItem(AUTH_KEY)
    if (storedAuth === "true" && storedUser) {
      try {
        const res = await fetch(`${API}/auth/verify?username=${encodeURIComponent(storedUser)}`)
        const data = await res.json()
        if (data.valid) {
          isLoggedIn.value = true
          userName.value = storedUser
          userAvatar.value = data.avatar || ""
          userNickname.value = data.nickname || ""
          userRole.value = data.role || "user"
      userThemeColor.value = data.theme_color || "#1976D2"
          localStorage.setItem(AVATAR_KEY, data.avatar || "")
          localStorage.setItem(NICK_KEY, data.nickname || "")
        } else {
          clearStorage()
        }
      } catch {
        isLoggedIn.value = true
        userName.value = storedUser
        userAvatar.value = localStorage.getItem(AVATAR_KEY) || ""
        userNickname.value = localStorage.getItem(NICK_KEY) || ""
        userRole.value = "user"
        userThemeColor.value = "#1976D2"
      }
    }
    ready.value = true
  }

  async function updateAvatar(avatar: string) {
    if (!isLoggedIn.value) return
    try {
      await fetch(`${API}/auth/avatar`, {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: userName.value, avatar }),
      })
      userAvatar.value = avatar
      localStorage.setItem(AVATAR_KEY, avatar)
    } catch { /* ignore */ }
  }

  async function updateNickname(nickname: string) {
    if (!isLoggedIn.value) return
    try {
      const res = await fetch(`${API}/auth/nickname`, {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: userName.value, nickname }),
      })
      const data = await res.json()
      if (data.success) {
        userNickname.value = data.nickname
        localStorage.setItem(NICK_KEY, data.nickname)
        return null
      }
      return data.error || "保存失败"
    } catch { return "无法连接服务器" }
  }

  async function updateThemeColor(color: string) {
    if (!isLoggedIn.value) return
    try {
      await fetch(`${API}/auth/theme`, {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: userName.value, theme: color }),
      })
      userThemeColor.value = color
    } catch { /* ignore */ }
  }

  async function updateAppIcon(appIcon: string) {
    if (!isLoggedIn.value) return
    try {
      await fetch(`${API}/auth/app-icon`, {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: userName.value, appIcon }),
      })
      userAppIcon.value = appIcon
    } catch { /* ignore */ }
  }

  async function register(username: string, password: string) {
    try {
      const res = await fetch(`${API}/auth/register`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })
      const data = await res.json()
      if (!res.ok) return data.error
      localStorage.setItem(AUTH_KEY, "true")
      localStorage.setItem(USER_KEY, username.trim())
      localStorage.setItem(AVATAR_KEY, ""); localStorage.setItem(NICK_KEY, "")
      isLoggedIn.value = true
      userName.value = username.trim()
      userAvatar.value = ""; userNickname.value = ""; userAppIcon.value = ""; userRole.value = data.role || "user"
      userThemeColor.value = "#1976D2"
      return null
    } catch { return "无法连接服务器" }
  }

  async function login(username: string, password: string) {
    try {
      const res = await fetch(`${API}/auth/login`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })
      const data = await res.json()
      if (!res.ok) return data.error
      localStorage.setItem(AUTH_KEY, "true")
      localStorage.setItem(USER_KEY, username.trim())
      localStorage.setItem(AVATAR_KEY, data.avatar || "")
      localStorage.setItem(NICK_KEY, data.nickname || "")
      isLoggedIn.value = true
      userName.value = username.trim()
      userAvatar.value = data.avatar || ""
      userNickname.value = data.nickname || ""
      userRole.value = data.role || "user"
      userThemeColor.value = data.theme_color || "#1976D2"
      return null
    } catch { return "无法连接服务器" }
  }

  function logout() {
    clearStorage()
    isLoggedIn.value = false
    userName.value = ""; userAvatar.value = ""; userNickname.value = ""; userAppIcon.value = ""; userThemeColor.value = "#1976D2"; userRole.value = "user"
  }

  return { isLoggedIn, userName, userAvatar, userNickname, userAppIcon, userThemeColor, userRole, isAdmin, ready,
    init, updateAvatar, updateNickname, updateThemeColor, updateAppIcon, register, login, logout }
})
