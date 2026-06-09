package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed dist/*
var staticFiles embed.FS

func main() {
	initDB()
	initAdmin()
	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/uploads/", handleUploads)
	http.HandleFunc("/", handleStatic)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	if len(os.Args) > 1 && os.Args[1] == "-port" && len(os.Args) > 2 {
		port = os.Args[2]
	}
	log.Println("Server on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api")
	switch {
	case strings.HasPrefix(path, "/auth/"):
		handleAuth(w, r, path)
	case strings.HasPrefix(path, "/notes"):
		// Check for export/import endpoints
		if path == "/notes/export" {
			handleNotesExport(w, r)
			return
		}
		if path == "/notes/import" {
			handleNotesImport(w, r)
			return
		}
		// Check for trash endpoints first, then notes
		if strings.HasSuffix(path, "/restore") || strings.HasSuffix(path, "/hard-delete") || path == "/notes/trash" {
			handleTrash(w, r, path)
			return
		}
		handleNotes(w, r, path)
	case strings.HasPrefix(path, "/settings"):
		handleSettings(w, r)
	case strings.HasPrefix(path, "/admin/"):
		handleAdmin(w, r, path)
	default:
		errResp(w, "not found", 404)
	}
}

func handleUploads(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if strings.Contains(filePath, "..") || strings.Contains(filePath, "/") || strings.Contains(filePath, "\\") {
		errResp(w, "invalid path", 400)
		return
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !allowedUploadExts[ext] {
		http.NotFound(w, r)
		return
	}
	fullPath := filepath.Join("uploads", filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	contentType := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".ico": "image/x-icon", ".bmp": "image/bmp",
	}[ext]
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeFile(w, r, fullPath)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		data, err := staticFiles.ReadFile("dist/index.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	data, err := staticFiles.ReadFile("dist" + r.URL.Path)
	if err != nil {
		data, err = staticFiles.ReadFile("dist/index.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	ext := filepath.Ext(r.URL.Path)
	mime := map[string]string{".js": "application/javascript", ".css": "text/css", ".png": "image/png", ".jpg": "image/jpeg", ".svg": "image/svg+xml", ".woff": "font/woff", ".woff2": "font/woff2", ".ico": "image/x-icon"}
	if m, ok := mime[ext]; ok {
		w.Header().Set("Content-Type", m)
	}
	w.Write(data)
}
