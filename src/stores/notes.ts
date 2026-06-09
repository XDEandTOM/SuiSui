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

export interface NoteReaction { [emoji: string]: string[] }

export interface Note {
  id: string
  content: string
  createdAt: number
  updatedAt: number
  pinned: boolean
  tags: string[]
  username: string
  avatar?: string
  nickname?: string
  reactions?: NoteReaction
}

function generateId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

export const useNotesStore = defineStore("notes", () => {
  const notes = ref<Note[]>([])
  const loaded = ref(false)

  async function fetchNotes() {
    try {
      const res = await fetch(addToken(`${API}/notes`))
      if (res.ok) { notes.value = await res.json(); notes.value = [...notes.value].sort((a, b) => { if (a.pinned !== b.pinned) return (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0); return b.createdAt - a.createdAt; }); loaded.value = true }
    } catch { console.warn("Failed to fetch notes from server") }
  }

  async function addNote(content: string, tags: string[] = [], username: string = "") {
    const auth = useAuthStore()
    const note: Note = {
      id: generateId(), content, createdAt: Date.now(), updatedAt: Date.now(),
      pinned: false, tags, username,
      avatar: auth.userAvatar || undefined,
      nickname: auth.userNickname || undefined,
    }
    try {
      const res = await fetch(addToken(`${API}/notes`), {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify(note),
      })
      if (res.ok) { notes.value = [note, ...notes.value].sort((a, b) => { if (a.pinned !== b.pinned) return (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0); return b.updatedAt - a.updatedAt; }); }
    } catch { console.warn("Failed to create note") }
  }

  async function updateNote(id: string, content: string, tags?: string[], username?: string) {
    const note = notes.value.find(m => m.id === id)
    if (!note) return
    const updatedAt = Date.now()
    try {
      const res = await fetch(addToken(`${API}/notes/${id}`), {
        method: "PUT", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content, tags: tags ?? note.tags, updatedAt, username: username || useAuthStore().userName || "" }),
      })
      if (res.ok) { const auth = useAuthStore(); note.content = content; note.updatedAt = updatedAt; if (tags !== undefined) note.tags = tags; note.avatar = auth.userAvatar || undefined; note.nickname = auth.userNickname || undefined }
    } catch { console.warn("Failed to update note") }
  }

  async function deleteNote(id: string, username?: string) {
    try {
      const auth = useAuthStore()
      const res = await fetch(addToken(`${API}/notes/${id}?username=${encodeURIComponent(username || auth.userName || "")}`), { method: "DELETE" })
      if (res.ok) notes.value = notes.value.filter(n => n.id !== id)
    } catch { console.warn("Failed to delete note") }
  }

  async function togglePin(id: string) {
    const note = notes.value.find(m => m.id === id)
    if (!note) return
    try {
      const res = await fetch(addToken(`${API}/notes/${id}/pin`), { method: "PATCH" })
      if (res.ok) { note.pinned = !note.pinned; notes.value = [...notes.value].sort((a, b) => { if (a.pinned !== b.pinned) return (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0); return b.updatedAt - a.updatedAt; }); }
    } catch { console.warn("Failed to toggle pin") }
  }

async function reactToNote(id: string, emoji: string, uid?: string) {
  if (!uid) uid = useAuthStore().userName || ""
  try {
    const res = await fetch(addToken(`${API}/notes/${id}/react`), {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ emoji, username: uid }),
    })
    if (res.ok) {
      const note = notes.value.find(n => n.id === id)
      if (note) {
        if (!note.reactions) note.reactions = {}
        if (!note.reactions[emoji]) note.reactions[emoji] = []
        if (!note.reactions[emoji].includes(uid)) note.reactions[emoji].push(uid)
      }
    }
  } catch { }
}

async function removeReaction(id: string, emoji: string, uid?: string) {
  if (!uid) uid = useAuthStore().userName || ""
  try {
    const res = await fetch(addToken(`${API}/notes/${id}/react`), {
      method: "DELETE", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ emoji, username: uid }),
    })
    if (res.ok) {
      const note = notes.value.find(n => n.id === id)
      if (note && note.reactions && note.reactions[emoji]) {
        note.reactions[emoji] = note.reactions[emoji].filter(u => u !== uid)
        if (note.reactions[emoji].length === 0) delete note.reactions[emoji]
      }
    }
  } catch { }
}

return { notes, loaded, fetchNotes, addNote, updateNote, deleteNote, togglePin, reactToNote, removeReaction }
})



