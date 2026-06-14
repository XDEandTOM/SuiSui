package main

import (
	"bufio"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/andybalholm/brotli"
	_ "modernc.org/sqlite"
)

//go:embed dist/*
var staticFiles embed.FS

//go:embed live.html
var livePage string

// Version is set at build time via -ldflags, fallback to "dev" in local builds.
var Version = "dev"
var serverPort = "3742"
var serverCertFile = ""
var serverKeyFile = ""
var brotliEnabled = true
var githubToken = ""

// SSE event hub
var sseClients = make(map[chan string]bool)
var sseMu sync.Mutex

// Live chat
var liveChat []string
var liveChatMu sync.Mutex
var liveSSEClients = make(map[chan string]bool)
var liveSSEMu sync.Mutex

// Live push status
var liveOnline bool
var liveOnlineMu sync.Mutex

// mediamtx process
var mediamtxCmd *exec.Cmd

func startMediaMTX() {
	path := os.Getenv("MEDIAMTX_PATH")
	if path == "" { return }
	// Create config
	cfg := `rtmp: yes
hls: yes
hlsDirectory: /data/hls
paths:
  stream:
    source: publisher
`
	cfgPath := filepath.Join(dataDir, "mediamtx.yml")
	os.WriteFile(cfgPath, []byte(cfg), 0644)

	mediamtxCmd = exec.Command(path, cfgPath)
	mediamtxCmd.Stdout = os.Stdout
	mediamtxCmd.Stderr = os.Stderr
	if err := mediamtxCmd.Start(); err != nil {
		log.Printf("mediamtx start failed: %v", err)
		return
	}
	log.Printf("MediaMTX started (pid %d)", mediamtxCmd.Process.Pid)
}

func liveChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var msg struct { Text string `json:"text"`; Author string `json:"author"` }
		if json.NewDecoder(r.Body).Decode(&msg) != nil || msg.Text == "" { errResp(w, "bad request", 400); return }
		payload := msg.Author + ": " + msg.Text
		liveChatMu.Lock()
		liveChat = append(liveChat, payload)
		if len(liveChat) > 100 { liveChat = liveChat[len(liveChat)-100:] }
		liveChatMu.Unlock()
		// Broadcast via SSE
		data, _ := json.Marshal(map[string]string{"text": msg.Text, "author": msg.Author})
		liveSSEMu.Lock()
		for ch := range liveSSEClients {
			select { case ch <- string(data): default: close(ch); delete(liveSSEClients, ch) }
		}
		liveSSEMu.Unlock()
		jsonResp(w, map[string]bool{"ok": true})
		return
	}
	// SSE stream
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
	// Send chat history
	liveChatMu.Lock()
	for _, m := range liveChat { fmt.Fprintf(w, "data: %s\n\n", m) }
	liveChatMu.Unlock()
	if flusher != nil { flusher.Flush() }

	ch := make(chan string, 10)
	liveSSEMu.Lock()
	liveSSEClients[ch] = true
	liveSSEMu.Unlock()

	notify := r.Context().Done()
	go func() {
		<-notify
		liveSSEMu.Lock()
		delete(liveSSEClients, ch); close(ch)
		liveSSEMu.Unlock()
	}()
	for msg := range ch { fmt.Fprintf(w, "data: %s\n\n", msg); flusher.Flush() }
}

func liveStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Check if HLS playlist exists
	online := false
	if _, err := os.Stat(filepath.Join(dataDir, "hls", "stream.m3u8")); err == nil {
		online = true
	}
	jsonResp(w, map[string]bool{"online": online})
}

func sseBroadcast(event string, data string) {
	sseMu.Lock()
	defer sseMu.Unlock()
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	for ch := range sseClients {
		select {
		case ch <- msg:
		default:
			close(ch)
			delete(sseClients, ch)
		}
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Remove server-level WriteTimeout for SSE (long-lived connection)
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	flusher, ok := w.(http.Flusher)
	if !ok { errResp(w, "streaming not supported", 500); return }
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Send initial keepalive to confirm connection
	fmt.Fprintf(w, ":ok\n\n")
	flusher.Flush()

	ch := make(chan string, 3)
	sseMu.Lock()
	sseClients[ch] = true
	sseMu.Unlock()

	notify := r.Context().Done()
	go func() {
		<-notify
		sseMu.Lock()
		delete(sseClients, ch)
		close(ch)
		sseMu.Unlock()
	}()

	for msg := range ch {
		fmt.Fprint(w, msg)
		flusher.Flush()
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3742"
	}
	certFile := ""
	keyFile := ""
	dataDir = "."

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-port":
			if i+1 < len(os.Args) { port = os.Args[i+1]; i++ }
		case "-data":
			if i+1 < len(os.Args) { dataDir = os.Args[i+1]; i++ }
		case "-cert":
			if i+1 < len(os.Args) { certFile = os.Args[i+1]; i++ }
		case "-key":
			if i+1 < len(os.Args) { keyFile = os.Args[i+1]; i++ }
		}
	}

	initDB()
	initAdmin()

	// Read server config from data directory (if exists)
	if certFile == "" && keyFile == "" {
		cfgPath := filepath.Join(dataDir, "server.json")
		if cfgData, err := os.ReadFile(cfgPath); err == nil {
			var cfg struct {
				Cert string `json:"cert"`
				Key  string `json:"key"`
				GitHubToken string `json:"github_token,omitempty"`
			}
			if json.Unmarshal(cfgData, &cfg) == nil {
				githubToken = cfg.GitHubToken
				if cfg.Cert != "" && cfg.Key != "" {
					certPath := filepath.Join(dataDir, cfg.Cert)
					keyPath := filepath.Join(dataDir, cfg.Key)
					if _, err := os.Stat(certPath); err == nil {
						certFile = certPath
						keyFile = keyPath
					}
				}
			}
		}
	}

	serverPort = port
	serverCertFile = certFile
	serverKeyFile = keyFile

	// Start mediamtx for live streaming (if binary exists)
	startMediaMTX()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", handleAPI)
	mux.HandleFunc("/uploads/", handleUploads)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/live/", handleLive)
	mux.HandleFunc("/hls/", handleHLS)
	mux.HandleFunc("/", handleStatic)

	handler := loggingMiddleware(compressMiddleware(mux))

	// TLS mode — cert and key provided
	if certFile != "" && keyFile != "" {
		// HTTP→HTTPS redirect server on :80
		go func() {
			redirMux := http.NewServeMux()
			redirMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
			})
			redirSrv := &http.Server{
				Addr:    ":80",
				Handler: loggingMiddleware(redirMux),
				ReadTimeout: 10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout: 30 * time.Second,
			}
			log.Printf("Redirect HTTP :80 → HTTPS")
			if err := redirSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("redirect server: %v", err)
			}
		}()

		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Printf("Server %s on :%s (TLS, data: %s)", Version, port, dataDir)
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen TLS: %v", err)
			}
		}()
	} else {
		// Plain HTTP mode (no TLS)
		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Printf("Server %s on :%s (data: %s)", Version, port, dataDir)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %v", err)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}

// compressMiddleware wraps responses with Brotli (preferred) or Gzip compression.
func compressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't compress already-compressed content (e.g. uploaded images, woff2)
		// or Server-Sent Events
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}

		ae := r.Header.Get("Accept-Encoding")
		if brotliEnabled && strings.Contains(ae, "br") {
			w.Header().Set("Content-Encoding", "br")
			w.Header().Set("Vary", "Accept-Encoding")
			bw := brotli.NewWriterLevel(w, brotli.DefaultCompression)
			defer bw.Close()
			next.ServeHTTP(&compressWriter{ResponseWriter: w, writer: bw}, r)
		} else if strings.Contains(ae, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			gw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
			defer gw.Close()
			next.ServeHTTP(&compressWriter{ResponseWriter: w, writer: gw}, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type compressWriter struct {
	http.ResponseWriter
	writer   io.Writer
	status   int
	headerWritten bool
}

func (c *compressWriter) Write(b []byte) (int, error) {
	if !c.headerWritten {
		if c.status != 0 {
			c.ResponseWriter.WriteHeader(c.status)
		}
		c.headerWritten = true
	}
	return c.writer.Write(b)
}

func (c *compressWriter) WriteHeader(status int) {
	if c.headerWritten { return }
	c.status = status
}

func (c *compressWriter) Flush() {
	if f, ok := c.writer.(interface{ Flush() }); ok {
		f.Flush()
	}
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
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.written {
		w.status = status
		w.written = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
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
	log.Printf("handleAPI path: %s", path)
	switch {
	case strings.HasPrefix(path, "/gh/"):
		handleGitHubProxy(w, r)
	case strings.HasPrefix(path, "/auth/"):
		handleAuth(w, r, path)
	case strings.HasPrefix(path, "/notes"):
		if path == "/notes/export" { handleNotesExport(w, r); return }
		if path == "/notes/import" { handleNotesImport(w, r); return }
		if strings.HasSuffix(path, "/restore") || strings.HasSuffix(path, "/hard-delete") || path == "/notes/trash" {
			handleTrash(w, r, path); return
		}
		handleNotes(w, r, path)
	case strings.HasPrefix(path, "/share/"):
		handleShareView(w, r)
	case strings.HasPrefix(path, "/settings"):
		handleSettings(w, r)
	case strings.HasPrefix(path, "/events"):
		sseHandler(w, r)
	case strings.HasPrefix(path, "/live/chat"):
		liveChatHandler(w, r)
	case strings.HasPrefix(path, "/live/status"):
		liveStatusHandler(w, r)
	case path == "/admin/config":
		jsonResp(w, map[string]interface{}{
			"version": Version,
			"port":    serverPort,
			"tls":     serverCertFile != "" && serverKeyFile != "",
			"dataDir": dataDir,
		})
	case strings.HasPrefix(path, "/admin/"):
		handleAdmin(w, r, path)
	default:
		errResp(w, "not found", 404)
	}
}

func handleUploads(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if strings.Contains(filePath, "..") || strings.Contains(filePath, "/") || strings.Contains(filePath, "\\") {
		errResp(w, "invalid path", 400); return
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !allowedUploadExts[ext] { http.NotFound(w, r); return }
	fullPath := filepath.Join(uploadsDir(), filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) { http.NotFound(w, r); return }
	w.Header().Set("X-Content-Type-Options", "nosniff")
	contentType := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".ico": "image/x-icon", ".bmp": "image/bmp"}[ext]
	if contentType != "" { w.Header().Set("Content-Type", contentType) }
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, fullPath)
}

func handleGitHubProxy(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/gh/")
	if path == "" { errResp(w, "missing path", 400); return }
	// Optional: use GITHUB_TOKEN env var for higher rate limits
	token := githubToken
	ghURL := "https://api.github.com/" + path
	req, _ := http.NewRequest("GET", ghURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil { errResp(w, "proxy error", 502); return }
	defer resp.Body.Close()
	for k, v := range resp.Header {
		if k == "Content-Type" || k == "Content-Length" {
			w.Header()[k] = v
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(livePage))
}

func handleHLS(w http.ResponseWriter, r *http.Request) {
	// Serve HLS segments (m3u8, ts) from /data/hls
	path := filepath.Join(dataDir, "hls", strings.TrimPrefix(r.URL.Path, "/hls/"))
	// Security: ensure path is within hls directory
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(filepath.Join(dataDir, "hls"))) {
		errResp(w, "forbidden", 403); return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, path)
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
