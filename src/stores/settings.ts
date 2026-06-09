import { defineStore } from "pinia"
import { ref } from "vue"
import { useAuthStore } from "@/stores/auth"

const API = "/api"

function addToken(url: string): string {
  try {
    const auth = useAuthStore()
    const token = auth.getAuthToken()
    if (!token) return url
    const sep = url.includes("?") ? "&" : "?"
    return url + sep + "token=" + encodeURIComponent(token)
  } catch {
    const token = localStorage.getItem("suisui-token")
    if (!token) return url
    const sep = url.includes("?") ? "&" : "?"
    return url + sep + "token=" + encodeURIComponent(token)
  }
}

export const useSettingsStore = defineStore("settings", () => {
  const siteTitle = ref("")
  const siteIcp = ref("")
  const allowRegister = ref(true)
  const siteFavicon = ref("")

  async function load() {
    try {
      const r = await fetch(addToken(API + "/settings"))
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
    const r = await fetch(addToken(API + "/settings"), {
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
