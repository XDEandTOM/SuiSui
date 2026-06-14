package main

import (
	"embed"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed dist/*
var staticFiles embed.FS

var livePage string

func init() {
	if data, err := os.ReadFile("live.html"); err == nil {
		livePage = string(data)
	}
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	if r.URL.Path == "/" {
		data, err := staticFiles.ReadFile("dist/index.html")
		if err != nil { w.WriteHeader(500); return }
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		w.Write(data); return
	}
	data, err := staticFiles.ReadFile("dist" + r.URL.Path)
	if err != nil {
		data, err = staticFiles.ReadFile("dist/index.html")
		if err != nil { w.WriteHeader(500); return }
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		w.Write(data); return
	}
	ext := filepath.Ext(r.URL.Path)
	mime := map[string]string{".js": "application/javascript", ".css": "text/css",
		".png": "image/png", ".jpg": "image/jpeg", ".svg": "image/svg+xml",
		".woff": "font/woff", ".woff2": "font/woff2", ".ico": "image/x-icon"}
	if m, ok := mime[ext]; ok { w.Header().Set("Content-Type", m) }
	if ext == ".js" || ext == ".css" || ext == ".woff2" || ext == ".woff" || ext == ".ttf" ||
		ext == ".eot" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" ||
		ext == ".webp" || ext == ".ico" || ext == ".svg" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	w.Write(data)
}
