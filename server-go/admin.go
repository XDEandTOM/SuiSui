package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT key, value FROM settings")
		if err != nil {
			errResp(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		s := map[string]string{"site_title": "", "allow_register": "true", "site_favicon": ""}
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
		jsonResp(w, map[string]string{"success": "ok"})
	}
}

func handleAdmin(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/admin/stats":
		_, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var users, notes int
		if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); err != nil {
			log.Printf("failed to count users: %v", err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&notes); err != nil {
			log.Printf("failed to count notes: %v", err)
		}
		jsonResp(w, map[string]int{"totalUsers": users, "totalNotes": notes})

	case path == "/admin/users":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var callerRole string
		db.QueryRow("SELECT role FROM users WHERE username=?", tokenUser).Scan(&callerRole)
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
		rows, err := db.Query("SELECT id, username, nickname, avatar, role, created_at FROM users ORDER BY id LIMIT ? OFFSET ?", perPage, offset)
		if err != nil {
			errResp(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var users []map[string]interface{}
		for rows.Next() {
			var id int
			var username, nickname, avatar, role string
			var createdAt int64
			if err := rows.Scan(&id, &username, &nickname, &avatar, &role, &createdAt); err != nil {
				continue
			}
			var memoCount int
			db.QueryRow("SELECT COUNT(*) FROM notes WHERE username=?", username).Scan(&memoCount)
			users = append(users, map[string]interface{}{
				"id": id, "username": username, "nickname": nickname, "avatar": avatar,
				"role": role, "createdAt": createdAt, "memoCount": memoCount,
			})
		}
		if err := rows.Err(); err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		if users == nil {
			users = []map[string]interface{}{}
		}
		jsonResp(w, map[string]interface{}{"users": users, "total": total, "page": page, "perPage": perPage})

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
			execSQL("DELETE FROM notes WHERE username=?", username)
			if _, err := db.Exec("DELETE FROM users WHERE id=?", parts[0]); err != nil {
				errResp(w, "删除用户失败", 500)
				return
			}
			jsonResp(w, map[string]string{"success": "ok"})
		}
	}
}
