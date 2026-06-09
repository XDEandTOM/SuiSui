package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed dist/*
var staticFiles embed.FS


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

var db *sql.DB

func main() {
	initDB()
	initAdmin()
	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/uploads/", handleUploads)
	http.HandleFunc("/", handleStatic)
	port := "3001"
	if len(os.Args) > 1 && os.Args[1] == "-port" && len(os.Args) > 2 {
		port = os.Args[2]
	}
	log.Println("Server on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	dbPath := "suisui.db"
	_, err := os.Stat(dbPath)
	os.MkdirAll("uploads", 0755)
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")
	tables := []string{
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
	db.Exec("ALTER TABLE users ADD COLUMN theme_color TEXT DEFAULT '#1976D2'")
	db.Exec("ALTER TABLE users ADD COLUMN app_icon TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN salt TEXT DEFAULT ''")
}

func initAdmin() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		salt := generateSalt()
		db.Exec("INSERT INTO users (username, password, nickname, role, created_at, salt) VALUES (?, ?, ?, ?, ?, ?)",
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

func handleAuth(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/auth/login" && r.Method == "POST":
		var body struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&body)
		// Rate limit: extract client IP
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" { ip = r.RemoteAddr }
		if !checkLoginRateLimit(ip) {
			errResp(w, "登录尝试过于频繁，请稍后再试", 429)
			return
		}
		var storedPwd, role, salt string
		err := db.QueryRow("SELECT password, role, salt FROM users WHERE username=?", body.Username).Scan(&storedPwd, &role, &salt)
		if err != nil || !checkPassword(body.Password, storedPwd, salt) {
			errResp(w, "用户名或密码错误", 401)
			return
		}
		var avatar, nickname string
		var themeColor string
		db.QueryRow("SELECT avatar, nickname, theme_color FROM users WHERE username=?", body.Username).Scan(&avatar, &nickname, &themeColor)
		token := generateToken()
		db.Exec("INSERT INTO auth_tokens (token, username, created_at) VALUES (?, ?, ?)", token, body.Username, time.Now().UnixMilli())
		jsonResp(w, map[string]interface{}{"username": body.Username, "avatar": avatar, "nickname": nickname, "role": role, "theme_color": themeColor, "token": token})

	case path == "/auth/register" && r.Method == "POST":
		var body struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Username) < 2 || len(body.Password) < 4 {
			errResp(w, "用户名至少2个字符，密码至少4个字符", 400)
			return
		}
		var allowReg string
		db.QueryRow("SELECT value FROM settings WHERE key='allow_register'").Scan(&allowReg)
		if allowReg == "false" {
			errResp(w, "注册已关闭", 403)
			return
		}
		regSalt := generateSalt()
		_, err := db.Exec("INSERT INTO users (username, password, role, created_at, salt) VALUES (?, ?, ?, ?, ?)",
			body.Username, hashPassword(body.Password, regSalt), "user", time.Now().UnixMilli(), regSalt)
		if err != nil {
			errResp(w, "用户已存在", 409)
			return
		}
		regToken := generateToken()
		db.Exec("INSERT INTO auth_tokens (token, username, created_at) VALUES (?, ?, ?)", regToken, body.Username, time.Now().UnixMilli())
		jsonResp(w, map[string]interface{}{"username": body.Username, "role": "user", "token": regToken})

	case path == "/auth/verify" && r.Method == "GET":
		username := r.URL.Query().Get("username")
		token := r.URL.Query().Get("token")
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != username {
			jsonResp(w, map[string]interface{}{"valid": false})
			return
		}
		var avatar, nickname, role, themeColor string
		err := db.QueryRow("SELECT avatar, nickname, role, theme_color FROM users WHERE username=?", username).Scan(&avatar, &nickname, &role, &themeColor)
		if err != nil {
			jsonResp(w, map[string]interface{}{"valid": false})
			return
		}
		var dbToken string
		_ = db.QueryRow("SELECT token FROM auth_tokens WHERE username=? ORDER BY created_at DESC LIMIT 1", username).Scan(&dbToken)
		if dbToken == "" {
			dbToken = token
		}
		jsonResp(w, map[string]interface{}{"valid": true, "avatar": avatar, "nickname": nickname, "role": role, "theme_color": themeColor, "token": dbToken})

	case path == "/auth/avatar" && r.Method == "PATCH":
		var body struct{ Username, Avatar string }
		json.NewDecoder(r.Body).Decode(&body)
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != body.Username {
			errResp(w, "unauthorized", 401)
			return
		}
		db.Exec("UPDATE users SET avatar=? WHERE username=?", body.Avatar, body.Username)
		jsonResp(w, map[string]string{"success": "ok"})

	case path == "/auth/nickname" && r.Method == "PATCH":
		var body struct{ Username, Nickname string }
		json.NewDecoder(r.Body).Decode(&body)
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != body.Username {
			errResp(w, "unauthorized", 401)
			return
		}
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE nickname=? AND username!=?", body.Nickname, body.Username).Scan(&count)
		if count > 0 {
			errResp(w, "昵称已存在", 409)
			return
		}
		db.Exec("UPDATE users SET nickname=? WHERE username=?", body.Nickname, body.Username)
		jsonResp(w, map[string]interface{}{"success": true, "nickname": body.Nickname})

	case path == "/auth/app-icon" && r.Method == "PATCH":
		var body struct{ Username, AppIcon string }
		json.NewDecoder(r.Body).Decode(&body)
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != body.Username {
			errResp(w, "unauthorized", 401)
			return
		}
		db.Exec("UPDATE users SET app_icon=? WHERE username=?", body.AppIcon, body.Username)
		jsonResp(w, map[string]string{"success": "ok"})

	case path == "/auth/theme" && r.Method == "PATCH":
		var body struct{ Username, Theme string }
		json.NewDecoder(r.Body).Decode(&body)
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != body.Username {
			errResp(w, "unauthorized", 401)
			return
		}
		db.Exec("UPDATE users SET theme_color=? WHERE username=?", body.Theme, body.Username)
		jsonResp(w, map[string]string{"success": "ok"})

	case path == "/auth/password" && r.Method == "PATCH":
		var body struct{ Username, OldPassword, NewPassword string }
		json.NewDecoder(r.Body).Decode(&body)
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != body.Username {
			errResp(w, "unauthorized", 401)
			return
		}
		var storedPwd, salt string
		db.QueryRow("SELECT password, salt FROM users WHERE username=?", body.Username).Scan(&storedPwd, &salt)
		if !checkPassword(body.OldPassword, storedPwd, salt) {
			errResp(w, "密码验证失败", 401)
			return
		}
		db.Exec("UPDATE users SET password=?, salt=? WHERE username=?", hashPassword(body.NewPassword, generateSalt()), generateSalt(), body.Username)
		jsonResp(w, map[string]string{"success": "ok"})

	case path == "/auth/avatar/upload" && r.Method == "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			errResp(w, "文件过大，最大 10MB", 400)
			return
		}
		_, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		file, header, err := r.FormFile("avatar")
		if err != nil {
			errResp(w, "文件读取失败", 400)
			return
		}
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		dst, _ := os.Create(filepath.Join("uploads", name))
		io.Copy(dst, file)
		dst.Close()
		jsonResp(w, map[string]interface{}{"success": true, "url": "/uploads/" + name})
	}
}

func handleNotes(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/notes" && r.Method == "GET":
		rows, err := db.Query("SELECT id, content, created_at, updated_at, pinned, tags, username, avatar, nickname FROM notes ORDER BY created_at DESC")
		if err != nil {
			errResp(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var notes []map[string]interface{}
		for rows.Next() {
			var id, content, username, tags, avatar, nickname string
			var createdAt, updatedAt int64
			var pinned int
			rows.Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &username, &avatar, &nickname)
			var tagList []string
			json.Unmarshal([]byte(tags), &tagList)
			notes = append(notes, map[string]interface{}{
				"id": id, "content": content, "createdAt": createdAt, "updatedAt": updatedAt,
				"pinned": pinned == 1, "tags": tagList, "username": username,
				"avatar": avatar, "nickname": nickname,"reactions": getReactions(id),
			})
		}
		if notes == nil {
			notes = []map[string]interface{}{}
		}
		jsonResp(w, notes)

	case path == "/notes" && r.Method == "POST":
		var n struct {
			Id, Content, Username, Avatar, Nickname string
			CreatedAt, UpdatedAt                    int64
			Tags                                    []string
		}
		json.NewDecoder(r.Body).Decode(&n)
		tagBytes, _ := json.Marshal(n.Tags)
		db.Exec("INSERT INTO notes (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname) VALUES (?,?,?,?,0,?,?,?,?)",
			n.Id, n.Content, n.CreatedAt, n.UpdatedAt, string(tagBytes), n.Username, n.Avatar, n.Nickname)
		jsonResp(w, map[string]string{"success": "ok"})

	case strings.Contains(path, "/upload") && r.Method == "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			errResp(w, "文件过大，最大 10MB", 400)
			return
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			errResp(w, "文件读取失败", 400)
			return
		}
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		dst, _ := os.Create(filepath.Join("uploads", name))
		io.Copy(dst, file)
		dst.Close()
		jsonResp(w, map[string]interface{}{"success": true, "url": "/uploads/" + name})

	case strings.HasSuffix(path, "/react") && (r.Method == "POST" || r.Method == "DELETE"):
		noteId := strings.TrimSuffix(strings.TrimPrefix(path, "/notes/"), "/react")
		var body struct{ Emoji, Username string }
		json.NewDecoder(r.Body).Decode(&body)
		if body.Emoji == "" || body.Username == "" { errResp(w, "emoji and username required", 400); return }
		if r.Method == "POST" {
			db.Exec("INSERT OR IGNORE INTO reactions (id, emoji, username) VALUES (?, ?, ?)", noteId, body.Emoji, body.Username)
		} else {
			db.Exec("DELETE FROM reactions WHERE id=? AND emoji=? AND username=?", noteId, body.Emoji, body.Username)
		}
		jsonResp(w, map[string]string{"success": "ok"})
	default:
		parts := strings.Split(strings.TrimPrefix(path, "/notes/"), "/")
		if len(parts) == 1 && r.Method == "PUT" {
			var body struct{ Content string; Tags []string; UpdatedAt int64; Username string }
			json.NewDecoder(r.Body).Decode(&body)
			if body.Username == "" { errResp(w, "username required", 400); return }
			// Verify token
			tokenUser, tokenValid := verifyToken(r)
			if !tokenValid || tokenUser != body.Username {
				errResp(w, "unauthorized", 401)
				return
			}
			var owner string
			db.QueryRow("SELECT username FROM notes WHERE id=?", parts[0]).Scan(&owner)
			if owner == "" { errResp(w, "note not found", 404); return }
			var callerRole string
			db.QueryRow("SELECT role FROM users WHERE username=?", body.Username).Scan(&callerRole)
			if body.Username != owner && callerRole != "admin" { errResp(w, "forbidden", 403); return }
			tagBytes, _ := json.Marshal(body.Tags)
			db.Exec("UPDATE notes SET content=?, tags=?, updated_at=? WHERE id=?", body.Content, string(tagBytes), body.UpdatedAt, parts[0])
			jsonResp(w, map[string]string{"success": "ok"})
		} else if len(parts) == 1 && r.Method == "DELETE" {
			username := r.URL.Query().Get("username")
			if username == "" { errResp(w, "username required", 400); return }
			// Verify token
			tokenUser, tokenValid := verifyToken(r)
			if !tokenValid || tokenUser != username {
				errResp(w, "unauthorized", 401)
				return
			}
			var owner string
			db.QueryRow("SELECT username FROM notes WHERE id=?", parts[0]).Scan(&owner)
			if owner == "" { errResp(w, "note not found", 404); return }
			var callerRole string
			db.QueryRow("SELECT role FROM users WHERE username=?", username).Scan(&callerRole)
			if username != owner && callerRole != "admin" { errResp(w, "forbidden", 403); return }
			var cont, ts, av, nk string
			var ct, ut int64
			var p int
			db.QueryRow("SELECT content, created_at, updated_at, pinned, tags, avatar, nickname FROM notes WHERE id=?", parts[0]).Scan(&cont, &ct, &ut, &p, &ts, &av, &nk)
			db.Exec("INSERT OR IGNORE INTO trash (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname, deleted_at) VALUES (?,?,?,?,?,?,?,?,?,?)", parts[0], cont, ct, ut, p, ts, owner, av, nk, time.Now().UnixMilli())
			db.Exec("DELETE FROM notes WHERE id=?", parts[0])
			jsonResp(w, map[string]string{"success": "ok"})
		} else if len(parts) == 2 && parts[1] == "pin" && r.Method == "PATCH" {
			_, tokenValid := verifyToken(r)
			if !tokenValid {
				errResp(w, "unauthorized", 401)
				return
			}
			db.Exec("UPDATE notes SET pinned = CASE WHEN pinned=0 THEN 1 ELSE 0 END WHERE id=?", parts[0])
			jsonResp(w, map[string]string{"success": "ok"})
		}
	}
}

func handleTrash(w http.ResponseWriter, r *http.Request, path string) {
	// GET /notes/trash?username=xxx
	if r.Method == "GET" && path == "/notes/trash" {
		username := r.URL.Query().Get("username")
		rows, err := db.Query("SELECT id, content, created_at, updated_at, pinned, tags, username, avatar, nickname, deleted_at FROM trash WHERE username=? ORDER BY deleted_at DESC", username)
		if err != nil { errResp(w, err.Error(), 500); return }
		defer rows.Close()
		var items []map[string]interface{}
		for rows.Next() {
			var id, content, uname, tags, avatar, nickname string
			var createdAt, updatedAt, deletedAt int64
			var pinned int
			rows.Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &uname, &avatar, &nickname, &deletedAt)
			var tagList []string
			json.Unmarshal([]byte(tags), &tagList)
			items = append(items, map[string]interface{}{
				"id": id, "content": content, "createdAt": createdAt, "updatedAt": updatedAt,
				"pinned": pinned == 1, "tags": tagList, "username": uname,
				"avatar": avatar, "nickname": nickname, "deletedAt": deletedAt,
			})
		}
		if items == nil { items = []map[string]interface{}{} }
		jsonResp(w, items)
		return
	}
	// PATCH /notes/:id/restore?username=xxx
	if r.Method == "PATCH" && strings.HasSuffix(path, "/restore") {
		noteId := strings.TrimSuffix(strings.TrimPrefix(path, "/notes/"), "/restore")
		username := r.URL.Query().Get("username")
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != username {
			errResp(w, "unauthorized", 401)
			return
		}
		var cont, ts, av, nk string
		var ct, ut int64
		var p int
		err := db.QueryRow("SELECT content, created_at, updated_at, pinned, tags, avatar, nickname FROM trash WHERE id=? AND username=?", noteId, username).Scan(&cont, &ct, &ut, &p, &ts, &av, &nk)
		if err != nil { errResp(w, "not found in trash", 404); return }
		db.Exec("INSERT OR IGNORE INTO notes (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname) VALUES (?,?,?,?,?,?,?,?,?)", noteId, cont, ct, ut, p, ts, username, av, nk)
		db.Exec("DELETE FROM trash WHERE id=?", noteId)
		jsonResp(w, map[string]string{"success": "ok"})
		return
	}
	// DELETE /notes/:id/hard-delete?username=xxx
	if r.Method == "DELETE" && strings.HasSuffix(path, "/hard-delete") {
		noteId := strings.TrimSuffix(strings.TrimPrefix(path, "/notes/"), "/hard-delete")
		username := r.URL.Query().Get("username")
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != username {
			errResp(w, "unauthorized", 401)
			return
		}
		result, _ := db.Exec("DELETE FROM trash WHERE id=? AND username=?", noteId, username)
		affected, _ := result.RowsAffected()
		if affected == 0 { errResp(w, "not found", 404); return }
		jsonResp(w, map[string]string{"success": "ok"})
		return
	}
	errResp(w, "not found", 404)
}


func getReactions(noteId string) map[string][]string {
	rows, err := db.Query("SELECT emoji, username FROM reactions WHERE id=?", noteId)
	if err != nil { return map[string][]string{} }
	defer rows.Close()
	reactions := map[string][]string{}
	for rows.Next() {
		var emoji, username string
		rows.Scan(&emoji, &username)
		reactions[emoji] = append(reactions[emoji], username)
	}
	if reactions == nil { reactions = map[string][]string{} }
	return reactions
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, _ := db.Query("SELECT key, value FROM settings")
		defer rows.Close()
		s := map[string]string{"site_title": "", "allow_register": "true", "site_favicon": ""}
		for rows.Next() {
			var k, v string
			rows.Scan(&k, &v)
			s[k] = v
		}
		jsonResp(w, s)
	} else if r.Method == "POST" {
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if callerRole != "admin" { errResp(w, "forbidden", 403); return }
		var body struct{ Key, Value string }
		json.NewDecoder(r.Body).Decode(&body)
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?,?)", body.Key, body.Value)
		jsonResp(w, map[string]string{"success": "ok"})
	}
}

func handleAdmin(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/admin/stats":
		_, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var users, notes int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users)
		db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&notes)
		jsonResp(w, map[string]int{"totalUsers": users, "totalNotes": notes})

	case path == "/admin/users":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if callerRole != "admin" { errResp(w, "forbidden", 403); return }
		page := 1
		perPage := 10
		p := r.URL.Query().Get("page")
		if p != "" { fmt.Sscanf(p, "%d", &page) }
		pp := r.URL.Query().Get("per_page")
		if pp != "" { fmt.Sscanf(pp, "%d", &perPage) }
		if page < 1 { page = 1 }
		if perPage < 1 { perPage = 10 }
		offset := (page - 1) * perPage
		var total int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
		rows, _ := db.Query("SELECT id, username, nickname, avatar, role, created_at FROM users ORDER BY id LIMIT ? OFFSET ?", perPage, offset)
		defer rows.Close()
		var users []map[string]interface{}
		for rows.Next() {
			var id int; var username, nickname, avatar, role string; var createdAt int64
			rows.Scan(&id, &username, &nickname, &avatar, &role, &createdAt)
			var memoCount int
			db.QueryRow("SELECT COUNT(*) FROM notes WHERE username=?", username).Scan(&memoCount)
			users = append(users, map[string]interface{}{
				"id": id, "username": username, "nickname": nickname, "avatar": avatar,
				"role": role, "createdAt": createdAt, "memoCount": memoCount,
			})
		}
		if users == nil { users = []map[string]interface{}{} }
		jsonResp(w, map[string]interface{}{"users": users, "total": total, "page": page, "perPage": perPage})

	default:
		parts := strings.Split(strings.TrimPrefix(path, "/admin/users/"), "/")
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if len(parts) == 1 && r.Method == "DELETE" {
			if callerRole != "admin" { errResp(w, "forbidden", 403); return }
			var username string
			db.QueryRow("SELECT username FROM users WHERE id=?", parts[0]).Scan(&username)
			db.Exec("DELETE FROM notes WHERE username=?", username)
			db.Exec("DELETE FROM users WHERE id=?", parts[0])
			jsonResp(w, map[string]string{"success": "ok"})
		}
	}
}

func handleUploads(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if strings.Contains(filePath, "..") || strings.Contains(filePath, "/") || strings.Contains(filePath, "\\") {
		errResp(w, "invalid path", 400)
		return
	}
	fullPath := filepath.Join("uploads", filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
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
		data, _ = staticFiles.ReadFile("dist/index.html")
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

