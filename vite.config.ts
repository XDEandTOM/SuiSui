import { defineConfig } from 'vite'
import Vue from '@vitejs/plugin-vue'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import { fileURLToPath } from 'url'

export default defineConfig({
  plugins: [Vue({ template: { transformAssetUrls } }), Vuetify({ autoImport: true })],
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: { port: 3000, proxy: { "/api": process.env.VITE_API_PROXY || "http://localhost:3001", "/uploads": process.env.VITE_API_PROXY || "http://localhost:3001" } }
})
