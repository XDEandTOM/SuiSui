import { defineConfig } from 'vite'
import Vue from '@vitejs/plugin-vue'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import { fileURLToPath } from 'url'
import viteCompression from 'vite-plugin-compression'

export default defineConfig({
  plugins: [Vue({ template: { transformAssetUrls } }), Vuetify({ autoImport: true }), viteCompression({ algorithm: 'gzip' }), viteCompression({ algorithm: 'brotliCompress' })],
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: { port: 3000, proxy: { "/api": process.env.VITE_API_PROXY || "http://localhost:3742", "/uploads": process.env.VITE_API_PROXY || "http://localhost:3742" } },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vuetify: ["vuetify"],
          marked: ["marked"],
        },
      },
    },
    chunkSizeWarningLimit: 400,
  },
})
