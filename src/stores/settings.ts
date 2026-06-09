import { defineStore } from "pinia"
import { ref } from "vue"
import { useAuthStore } from "@/stores/auth"
import { authFetch } from "@/utils/api"

const API = "/api"

export const useSettingsStore = defineStore("settings", () => {
  const siteTitle = ref("")
  const siteIcp = ref("")
  const allowRegister = ref(true)
  const siteFavicon = ref("")

  async function load() {
    try {
      const r = await authFetch(API + "/settings")
      if (r.ok) {
        const s = await r.json()
        siteTitle.value = s.site_title || ""
        siteIcp.value = s.site_icp || ""
        allowRegister.value = s.allow_register !== "false"
        siteFavicon.value = s.site_favicon || ""
      }
    } catch { /* ignore */ }
  }

  async function save(key: string, value: string) {
    const r = await authFetch(API + "/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value })
    })
    return r.ok
  }

  function applyTitle() {
    document.title = siteTitle.value || "碎碎"
    if (siteFavicon.value) {
      const link = document.querySelector("link[rel=\"icon\"]") || document.createElement("link")
      link.setAttribute("rel", "icon")
      link.setAttribute("href", siteFavicon.value)
      document.head.appendChild(link)
    }
  }

  return { siteTitle, siteIcp, allowRegister, siteFavicon, load, save, applyTitle }
})
