<script setup lang="ts">
import { ref } from "vue"
import { authFetch } from "@/utils/api"
const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: boolean] }>()
const API = "/api"
const uploading = ref(false)

async function onUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  const fd = new FormData()
  fd.append("avatar", file)
  try {
    const res = await authFetch(API + "/auth/avatar/upload", { method: "POST", body: fd })
    const data = await res.json()
    if (data.success) {
      await authFetch(API + "/settings", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: "site_favicon", value: data.url })
      })
      const link = document.querySelector("link[rel=icon]") || document.createElement("link")
      link.setAttribute("rel", "icon")
      link.setAttribute("href", data.url)
      document.head.appendChild(link)
    }
  } catch {}
  uploading.value = false
  emit("update:modelValue", false)
}
</script>
<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)" max-width="400">
    <v-card class="rounded-xl pa-4">
      <v-card-title class="text-subtitle-1 font-weight-medium px-0">Favicon</v-card-title>
      <v-card-text class="px-0 d-flex flex-column align-center ga-3">
        <v-btn variant="outlined" color="primary" :loading="uploading" @click="() => $refs.fileInput?.click()">上传 Favicon</v-btn>
        <input ref="fileInput" type="file" accept="image/*" hidden @change="onUpload" />
      </v-card-text>
      <v-card-actions class="px-0"><v-spacer /><v-btn variant="text" @click="emit('update:modelValue', false)">关闭</v-btn></v-card-actions>
    </v-card>
  </v-dialog>
</template>
