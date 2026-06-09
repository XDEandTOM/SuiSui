package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	loginAttempts = make(map[string]struct {
		count    int
		lastTime time.Time
	})
	loginMu sync.Mutex
)

func checkLoginRateLimit(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()
	entry, exists := loginAttempts[ip]
	now := time.Now()
	if !exists || now.Sub(entry.lastTime) > 1*time.Minute {
		loginAttempts[ip] = struct {
			count    int
			lastTime time.Time
		}{count: 1, lastTime: now}
		return true
	}
	if entry.count >= 5 {
		return false
	}
	loginAttempts[ip] = struct {
		count    int
		lastTime time.Time
	}{count: entry.count + 1, lastTime: now}
	return true
}

func resetLoginRateLimit(ip string) {
	loginMu.Lock()
	delete(loginAttempts, ip)
	loginMu.Unlock()
}

var db *sql.DB
var dataDir = "."

// execSQL executes a statement and logs any error without returning it.
// Use for best-effort operations (cache, cleanup, startup).
func execSQL(query string, args ...interface{}) {
	if _, err := db.Exec(query, args...); err != nil {
		log.Printf("sql exec error: %s — %v", query, err)
	}
}

var allowedUploadExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".ico": true, ".bmp": true,
}

func uploadsDir() string {
	return filepath.Join(dataDir, "uploads")
}

func initDB() {
	dbPath := filepath.Join(dataDir, "suisui.db")
	_, err := os.Stat(dbPath)
	os.MkdirAll(filepath.Join(dataDir, "uploads"), 0755)
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	execSQL("PRAGMA journal_mode=WAL")
	execSQL("PRAGMA foreign_keys=ON")
	tables := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at INTEGER)`,
		`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, nickname TEXT DEFAULT '', avatar TEXT DEFAULT '', role TEXT DEFAULT 'user', created_at INTEGER DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS notes (id TEXT PRIMARY KEY, content TEXT, created_at INTEGER, updated_at INTEGER, pinned INTEGER DEFAULT 0, tags TEXT DEFAULT '[]', username TEXT, avatar TEXT, nickname TEXT)`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE IF NOT EXISTS reactions (id TEXT, emoji TEXT, username TEXT, PRIMARY KEY (id, emoji, username))`,
		`CREATE TABLE IF NOT EXISTS trash (id TEXT PRIMARY KEY, content TEXT, created_at INTEGER, updated_at INTEGER, pinned INTEGER DEFAULT 0, tags TEXT DEFAULT '[]', username TEXT, avatar TEXT, nickname TEXT, deleted_at INTEGER)`,
		`CREATE TABLE IF NOT EXISTS auth_tokens (token TEXT PRIMARY KEY, username TEXT, created_at INTEGER)`,
	}
	for _, t := range tables {
		if _, err := db.Exec(t); err != nil {
			log.Fatal(err)
		}
	}
	migrate()
}

const schemaVersion = 3

func migrate() {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		log.Printf("failed to read schema version: %v", err)
	}
	if version >= schemaVersion {
		return
	}
	if version < 1 {
		execSQL("INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (1, ?)", time.Now().UnixMilli())
		version = 1
	}
	if version < 2 {
		execSQL("ALTER TABLE users ADD COLUMN theme_color TEXT DEFAULT '#1976D2'")
		execSQL("ALTER TABLE users ADD COLUMN app_icon TEXT DEFAULT ''")
		execSQL("ALTER TABLE users ADD COLUMN salt TEXT DEFAULT ''")
		execSQL("CREATE INDEX IF NOT EXISTS idx_notes_username ON notes(username)")
		execSQL("CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at)")
		execSQL("CREATE INDEX IF NOT EXISTS idx_trash_username ON trash(username)")
		execSQL("INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (2, ?)", time.Now().UnixMilli())
		version = 2
	}
	if version < 3 {
		execSQL("ALTER TABLE notes ADD COLUMN pin_order INTEGER DEFAULT 0")
		execSQL("ALTER TABLE trash ADD COLUMN pin_order INTEGER DEFAULT 0")
		execSQL("INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (3, ?)", time.Now().UnixMilli())
		version = 3
	}
	log.Printf("Schema migrated to v%d", version)
}

func initAdmin() {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		log.Printf("failed to check admin existence: %v", err)
	}
	if count == 0 {
		salt := generateSalt()
		execSQL("INSERT INTO users (username, password, nickname, role, created_at, salt) VALUES (?, ?, ?, ?, ?, ?)",
			"admin", hashPassword("admin", salt), "Admin", "admin", time.Now().UnixMilli(), salt)
		log.Println("Admin user created: admin / admin")
	}
}

func generateSalt() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(pwd, salt string) string {
	h := sha256.Sum256([]byte(salt + pwd))
	return hex.EncodeToString(h[:])
}

func checkPassword(input, stored, salt string) bool {
	return hashPassword(input, salt) == stored
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func verifyToken(r *http.Request) (string, bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:]
		}
	}
	if token == "" {
		return "", false
	}
	var username string
	err := db.QueryRow("SELECT username FROM auth_tokens WHERE token=?", token).Scan(&username)
	if err != nil {
		return "", false
	}
	return username, true
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func errResp(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
