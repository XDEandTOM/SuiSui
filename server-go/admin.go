package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT key, value FROM settings")
		if err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		defer rows.Close()
		s := map[string]string{"site_title": "", "allow_register": "true", "site_favicon": "", "site_icp": "", "live_stream_url": ""}
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				continue
			}
			s[k] = v
		}
		if err := rows.Err(); err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		jsonResp(w, s)
	} else if r.Method == "POST" {
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var callerRole string
		if err := db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole); err != nil {
			log.Printf("failed to query role: %v", err)
		}
		if callerRole != "admin" {
			errResp(w, "forbidden", 403)
			return
		}
		var body struct{ Key, Value string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errResp(w, "无效的请求数据", 400)
			return
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?,?)", body.Key, body.Value); err != nil {
			errResp(w, "设置保存失败", 500)
			return
		}
		jsonResp(w, successResponse{Success: "ok"})
	}
}

func handleAdmin(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/admin/stats":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var callerRole string
		if err := db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole); err != nil || callerRole != "admin" {
			errResp(w, "forbidden", 403)
			return
		}
		var users, notes int
		if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); err != nil {
			log.Printf("failed to count users: %v", err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&notes); err != nil {
			log.Printf("failed to count notes: %v", err)
		}
		jsonResp(w, adminStatsResponse{TotalUsers: users, TotalNotes: notes})

	case path == "/admin/users":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var callerRole string
		if err := db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole); err != nil {
			log.Printf("failed to query caller role: %v", err)
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		if callerRole != "admin" {
			errResp(w, "forbidden", 403)
			return
		}
		page := 1
		perPage := 10
		p := r.URL.Query().Get("page")
		if p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		pp := r.URL.Query().Get("per_page")
		if pp != "" {
			fmt.Sscanf(pp, "%d", &perPage)
		}
		if page < 1 {
			page = 1
		}
		if perPage < 1 {
			perPage = 10
		}
		offset := (page - 1) * perPage
		var total int
		if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total); err != nil {
			log.Printf("failed to count users for pagination: %v", err)
		}
		// Single query with LEFT JOIN — no N+1
		rows, err := db.Query(`
			SELECT u.id, u.username, u.nickname, u.avatar, u.role, u.created_at,
			       COUNT(n.id) AS memo_count
			FROM users u
			LEFT JOIN notes n ON n.username = u.username
			GROUP BY u.id
			ORDER BY u.id LIMIT ? OFFSET ?
		`, perPage, offset)
		if err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		defer rows.Close()
		var userList []adminUserResponse
		for rows.Next() {
			var id int
			var username, nickname, avatar, role string
			var createdAt int64
			var memoCount int
			if err := rows.Scan(&id, &username, &nickname, &avatar, &role, &createdAt, &memoCount); err != nil {
				continue
			}
			userList = append(userList, adminUserResponse{
				ID: id, Username: username, Nickname: nickname, Avatar: avatar,
				Role: role, CreatedAt: createdAt, MemoCount: memoCount,
			})
		}
		if err := rows.Err(); err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		if userList == nil {
			userList = []adminUserResponse{}
		}
		jsonResp(w, adminUserListResponse{Users: userList, Total: total, Page: page, PerPage: perPage})

	case path == "/admin/config/ssl" && r.Method == "POST":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if callerRole != "admin" { errResp(w, "forbidden", 403); return }

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			errResp(w, "文件过大", 400); return
		}
		certFile, _, err := r.FormFile("cert")
		if err != nil { errResp(w, "缺少证书文件", 400); return }
		defer certFile.Close()
		certData, _ := io.ReadAll(certFile)
		if err := os.WriteFile(filepath.Join(dataDir, "cert.pem"), certData, 0644); err != nil {
			errResp(w, "证书保存失败", 500); return
		}

		keyFile, _, err := r.FormFile("key")
		if err != nil { errResp(w, "缺少私钥文件", 400); return }
		defer keyFile.Close()
		keyData, _ := io.ReadAll(keyFile)
		if err := os.WriteFile(filepath.Join(dataDir, "key.pem"), keyData, 0644); err != nil {
			errResp(w, "私钥保存失败", 500); return
		}

		// Save config so server picks it up on restart
		cfg := map[string]string{"cert": "cert.pem", "key": "key.pem"}
		cfgData, _ := json.Marshal(cfg)
		os.WriteFile(filepath.Join(dataDir, "server.json"), cfgData, 0644)

		jsonResp(w, successResponse{Success: "ok"})

	case path == "/admin/config/ssl" && r.Method == "DELETE":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if callerRole != "admin" { errResp(w, "forbidden", 403); return }

		os.Remove(filepath.Join(dataDir, "server.json"))
		jsonResp(w, successResponse{Success: "ok"})

	case path == "/admin/restart" && r.Method == "POST":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid { errResp(w, "unauthorized", 401); return }
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
		if callerRole != "admin" { errResp(w, "forbidden", 403); return }

		jsonResp(w, successResponse{Success: "ok"})
		// Graceful shutdown — Docker will auto-restart
		go func() {
			time.Sleep(500 * time.Millisecond)
			os.Exit(0)
		}()



	default:
		parts := strings.Split(strings.TrimPrefix(path, "/admin/users/"), "/")
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var callerRole string
		if err := db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole); err != nil {
			log.Printf("failed to query caller role: %v", err)
		}
		if len(parts) == 1 && r.Method == "DELETE" {
			if callerRole != "admin" {
				errResp(w, "forbidden", 403)
				return
			}
			var username string
			if err := db.QueryRow("SELECT username FROM users WHERE id=?", parts[0]).Scan(&username); err != nil {
				errResp(w, "用户不存在", 404)
				return
			}
			execSQLLog("DELETE FROM notes WHERE username=?", username)
			if _, err := db.Exec("DELETE FROM users WHERE id=?", parts[0]); err != nil {
				errResp(w, "删除用户失败", 500)
				return
			}
			jsonResp(w, successResponse{Success: "ok"})
		}
	}
}
