import { useAuthStore } from "@/stores/auth"

const TOKEN_KEY = "suisui-token"

function getToken(): string {
  try {
    const auth = useAuthStore()
    return auth.getAuthToken()
  } catch {
    return localStorage.getItem(TOKEN_KEY) || ""
  }
}

/** Fetch wrapper that automatically attaches Authorization: Bearer header. */
export function authFetch(url: string, options?: RequestInit): Promise<Response> {
  const token = getToken()
  if (token) {
    options = options || {}
    options.headers = { ...options.headers as Record<string, string> || {}, "Authorization": "Bearer " + token }
  }
  return fetch(url, options)
}

/** @deprecated Use authFetch instead. Appends token as URL query parameter. */
export function addToken(url: string): string {
  const token = getToken()
  if (!token) return url
  const sep = url.includes("?") ? "&" : "?"
  return url + sep + "token=" + encodeURIComponent(token)
}
