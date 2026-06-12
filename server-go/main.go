package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed dist/*
var staticFiles embed.FS

// Version is set at build time via -ldflags, fallback to "dev" in local builds.
var Version = "dev"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-port":
			if i+1 < len(os.Args) {
				port = os.Args[i+1]
				i++
			}
		case "-data":
			if i+1 < len(os.Args) {
				dataDir = os.Args[i+1]
				i++
			}
		}
	}
	initDB()
	initAdmin()

	// Build the handler chain
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", handleAPI)
	mux.HandleFunc("/uploads/", handleUploads)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/", handleStatic)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start gracefully
	go func() {
		log.Printf("Server %s on :%s (data: %s)", Version, port, dataDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Wait for signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

// loggingMiddleware logs each request with method, path, status and duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		duration := time.Since(start)
		if r.URL.Path != "/favicon.ico" {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, duration)
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Verify DB connectivity
	var dbVer int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&dbVer)
	if err != nil {
		jsonResp(w, healthResponse{Status: "error", Message: err.Error()})
		return
	}
	jsonResp(w, healthResponse{Status: "ok", DBSchemaVersion: dbVer, Version: Version})
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	cors(w)
	securityHeaders(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api")
	switch {
	case strings.HasPrefix(path, "/auth/"):
		handleAuth(w, r, path)
	case strings.HasPrefix(path, "/notes"):
		if path == "/notes/export" {
			handleNotesExport(w, r)
			return
		}
		if path == "/notes/import" {
			handleNotesImport(w, r)
			return
		}
		if strings.HasSuffix(path, "/restore") || strings.HasSuffix(path, "/hard-delete") || path == "/notes/trash" {
			handleTrash(w, r, path)
			return
		}
		handleNotes(w, r, path)
	case strings.HasPrefix(path, "/share/"):
		handleShareView(w, r)
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
	fullPath := filepath.Join(uploadsDir(), filePath)
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
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, fullPath)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	if r.URL.Path == "/" {
		data, err := staticFiles.ReadFile("dist/index.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
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
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		w.Write(data)
		return
	}
	ext := filepath.Ext(r.URL.Path)
	mime := map[string]string{".js": "application/javascript", ".css": "text/css", ".png": "image/png", ".jpg": "image/jpeg", ".svg": "image/svg+xml", ".woff": "font/woff", ".woff2": "font/woff2", ".ico": "image/x-icon"}
	if m, ok := mime[ext]; ok {
		w.Header().Set("Content-Type", m)
	}
	// Hash-based assets (.js/.css/.woff2 etc.) — cache forever
	if ext == ".js" || ext == ".css" || ext == ".woff2" || ext == ".woff" || ext == ".ttf" || ext == ".eot" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".ico" || ext == ".svg" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	w.Write(data)
}
