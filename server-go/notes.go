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

func buildNotesQuery(params map[string]string) (query string, countQuery string, args []interface{}) {
	baseWhere := "WHERE 1=1"
	var allArgs []interface{}

	if search := params["search"]; search != "" {
		baseWhere += " AND content LIKE ?"
		allArgs = append(allArgs, "%"+search+"%")
	}
	if tag := params["tag"]; tag != "" {
		baseWhere += " AND tags LIKE ?"
		allArgs = append(allArgs, `%"`+tag+`"%`)
	}
	if date := params["date"]; date != "" {
		baseWhere += " AND strftime('%Y-%m-%d', created_at / 1000, 'unixepoch') = ?"
		allArgs = append(allArgs, date)
	}
	if username := params["username"]; username != "" {
		baseWhere += " AND username = ?"
		allArgs = append(allArgs, username)
	}

	orderBy := "ORDER BY pinned DESC, CASE WHEN pinned=1 THEN pin_order ELSE 0 END ASC, created_at DESC"
	selectCols := "id, content, created_at, updated_at, pinned, tags, username, avatar, nickname"
	countSQL := "SELECT COUNT(*) FROM notes " + baseWhere
	dataSQL := "SELECT " + selectCols + " FROM notes " + baseWhere + " " + orderBy

	return dataSQL, countSQL, allArgs
}

func handleNotes(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/notes" && r.Method == "GET":
		q := r.URL.Query()
		limit := 0
		offset := 0
		if l := q.Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		if o := q.Get("offset"); o != "" {
			fmt.Sscanf(o, "%d", &offset)
		}

		params := map[string]string{
			"search":   q.Get("search"),
			"tag":      q.Get("tag"),
			"date":     q.Get("date"),
			"username": q.Get("username"),
		}
		dataSQL, countSQL, args := buildNotesQuery(params)

		// Get total count
		var total int
		if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
			log.Printf("count query failed: %v", err)
			total = 0
		}

		// Apply pagination
		if limit > 0 {
			dataSQL += " LIMIT ?"
			args = append(args, limit)
			if offset > 0 {
				dataSQL += " OFFSET ?"
				args = append(args, offset)
			}
		}

		rows, err := db.Query(dataSQL, args...)
		if err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		defer rows.Close()
		var notes []noteResponse
		var allIds []string
		for rows.Next() {
			var id, content, username, tags, avatar, nickname string
			var createdAt, updatedAt int64
			var pinned int
			if err := rows.Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &username, &avatar, &nickname); err != nil {
				continue
			}
			allIds = append(allIds, id)
			var tagList []string
			if err := json.Unmarshal([]byte(tags), &tagList); err != nil {
				log.Printf("failed to parse tags from note %s: %v", id, err)
			}
			notes = append(notes, noteResponse{
				ID: id, Content: content, CreatedAt: createdAt, UpdatedAt: updatedAt,
				Pinned: pinned == 1, Tags: tagList, Username: username,
				Avatar: avatar, Nickname: nickname,
			})
		}
		if err := rows.Err(); err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		// Batch-load reactions for all notes
		reactionsMap := batchGetReactions(allIds)
		for i := range notes {
			if r, ok := reactionsMap[notes[i].ID]; ok {
				notes[i].Reactions = r
			} else {
				notes[i].Reactions = map[string][]string{}
			}
		}
		if notes == nil {
			notes = []noteResponse{}
		}
		// When paginating, return object with total; otherwise return array (backward compat)
		if limit > 0 {
			jsonResp(w, paginatedNotesResponse{Notes: notes, Total: total, Limit: limit, Offset: offset})
		} else {
			jsonResp(w, notes)
		}

	case path == "/notes" && r.Method == "POST":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var n struct {
			Id, Content, Username, Avatar, Nickname string
			CreatedAt, UpdatedAt                    int64
			Tags                                    []string
		}
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			errResp(w, "无效的请求数据", 400)
			return
		}
		if len(n.Content) > 50000 {
			errResp(w, "内容过长（上限 50000 字符）", 400)
			return
		}
		if len(n.Username) > 50 {
			errResp(w, "用户名过长", 400)
			return
		}
		if n.Username == "" || n.Username != tokenUser {
			errResp(w, "unauthorized", 401)
			return
		}
		tagBytes, err := json.Marshal(n.Tags)
		if err != nil {
			errResp(w, "无效的标签", 400)
			return
		}
		if _, err := db.Exec("INSERT INTO notes (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname) VALUES (?,?,?,?,0,?,?,?,?)",
			n.Id, n.Content, n.CreatedAt, n.UpdatedAt, string(tagBytes), n.Username, n.Avatar, n.Nickname); err != nil {
			errResp(w, "碎片笔记创建失败", 500)
			return
		}
		sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})

	case strings.Contains(path, "/upload") && r.Method == "POST":
		if _, tokenValid := verifyToken(r); !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			errResp(w, "文件过大，最大 10MB", 400)
			return
		}
		defer r.MultipartForm.RemoveAll()
		file, header, err := r.FormFile("image")
		if err != nil {
			errResp(w, "文件读取失败", 400)
			return
		}
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		if ext != "" {
			ext = strings.ToLower(ext)
		}
		if !allowedUploadExts[ext] {
			errResp(w, "不支持的文件格式", 400)
			return
		}
		name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		dir := uploadsDir()
		os.MkdirAll(dir, 0755)
		dst, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			errResp(w, "文件写入失败", 500)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			errResp(w, "文件写入失败", 500)
			return
		}
		jsonResp(w, uploadResponse{Success: true, URL: "/uploads/" + name})

	case strings.HasSuffix(path, "/react") && (r.Method == "POST" || r.Method == "DELETE"):
		noteId := strings.TrimSuffix(strings.TrimPrefix(path, "/notes/"), "/react")
		var body struct{ Emoji, Username string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errResp(w, "无效的请求数据", 400)
			return
		}
		if body.Emoji == "" || body.Username == "" {
			errResp(w, "invalid request", 400)
			return
		}
		// Allow guest reactions (guest_ prefix) without token, or verify token for logged-in users
		if strings.HasPrefix(body.Username, "guest_") {
			// Guest reaction — no auth required
		} else {
			uid, tokenValid := verifyToken(r)
			if !tokenValid || uid == "" || body.Username != uid {
				errResp(w, "unauthorized", 401)
				return
			}
		}
		if r.Method == "POST" {
			if err := execSQL("INSERT OR IGNORE INTO reactions (id, emoji, username) VALUES (?, ?, ?)", noteId, body.Emoji, body.Username); err != nil {
				errResp(w, "操作失败", 500)
				return
			}
		} else {
			if err := execSQL("DELETE FROM reactions WHERE id=? AND emoji=? AND username=?", noteId, body.Emoji, body.Username); err != nil {
				errResp(w, "操作失败", 500)
				return
			}
		}
		sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
	case path == "/notes/reorder" && r.Method == "PATCH":
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid {
			errResp(w, "unauthorized", 401)
			return
		}
		var body struct{ Order []string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errResp(w, "无效的请求数据", 400)
			return
		}
		for i, id := range body.Order {
			if err := execSQL("UPDATE notes SET pin_order=? WHERE id=? AND username=?", i, id, tokenUser); err != nil {
				errResp(w, "排序失败", 500)
				return
			}
		}
		sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
	default:
		parts := strings.Split(strings.TrimPrefix(path, "/notes/"), "/")
		if len(parts) == 1 && r.Method == "PUT" {
			var body struct{ Content string; Tags []string; UpdatedAt int64; Username string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				errResp(w, "无效的请求数据", 400)
				return
			}
			if body.Username == "" {
				errResp(w, "username required", 400)
				return
			}
			// Verify token
			tokenUser, tokenValid := verifyToken(r)
			if !tokenValid || tokenUser != body.Username {
				errResp(w, "unauthorized", 401)
				return
			}
			var owner string
			if err := db.QueryRow("SELECT username FROM notes WHERE id=?", parts[0]).Scan(&owner); err != nil {
				errResp(w, "note not found", 404)
				return
			}
			var callerRole string
			if err := db.QueryRow("SELECT role FROM users WHERE username=?", body.Username).Scan(&callerRole); err != nil {
				log.Printf("failed to query caller role: %v", err)
			}
			if body.Username != owner && callerRole != "admin" {
				errResp(w, "forbidden", 403)
				return
			}
			tagBytes, err := json.Marshal(body.Tags)
			if err != nil {
				errResp(w, "无效的标签", 400)
				return
			}
			if _, err := db.Exec("UPDATE notes SET content=?, tags=?, updated_at=? WHERE id=?", body.Content, string(tagBytes), body.UpdatedAt, parts[0]); err != nil {
				errResp(w, "碎片笔记更新失败", 500)
				return
			}
			sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
		} else if len(parts) == 1 && r.Method == "DELETE" {
			username := r.URL.Query().Get("username")
			if username == "" {
				errResp(w, "username required", 400)
				return
			}
			// Verify token
			tokenUser, tokenValid := verifyToken(r)
			if !tokenValid || tokenUser != username {
				errResp(w, "unauthorized", 401)
				return
			}
			var owner string
			if err := db.QueryRow("SELECT username FROM notes WHERE id=?", parts[0]).Scan(&owner); err != nil {
				errResp(w, "note not found", 404)
				return
			}
			var callerRole string
			if err := db.QueryRow("SELECT role FROM users WHERE username=?", username).Scan(&callerRole); err != nil {
				log.Printf("failed to query caller role: %v", err)
			}
			if username != owner && callerRole != "admin" {
				errResp(w, "forbidden", 403)
				return
			}
			var cont, ts, av, nk string
			var ct, ut int64
			var p int
			if err := db.QueryRow("SELECT content, created_at, updated_at, pinned, tags, avatar, nickname FROM notes WHERE id=?", parts[0]).Scan(&cont, &ct, &ut, &p, &ts, &av, &nk); err != nil {
				errResp(w, "碎片笔记不存在", 404)
				return
			}
			if _, err := db.Exec("INSERT OR IGNORE INTO trash (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname, deleted_at) VALUES (?,?,?,?,?,?,?,?,?,?)", parts[0], cont, ct, ut, p, ts, owner, av, nk, time.Now().UnixMilli()); err != nil {
				errResp(w, "删除失败", 500)
				return
			}
			if err := execSQL("DELETE FROM notes WHERE id=?", parts[0]); err != nil {
				errResp(w, "删除失败", 500)
				return
			}
			sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
		} else if len(parts) == 2 && parts[1] == "pin" && r.Method == "PATCH" {
			_, tokenValid := verifyToken(r)
			if !tokenValid {
				errResp(w, "unauthorized", 401)
				return
			}
			if _, err := db.Exec("UPDATE notes SET pinned = CASE WHEN pinned=0 THEN 1 ELSE 0 END WHERE id=?", parts[0]); err != nil {
				errResp(w, "操作失败", 500)
				return
			}
			sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
		}
	}
}

func handleTrash(w http.ResponseWriter, r *http.Request, path string) {
	// GET /notes/trash?username=xxx
	if r.Method == "GET" && path == "/notes/trash" {
		username := r.URL.Query().Get("username")
		if username == "" {
			errResp(w, "username required", 400)
			return
		}
		tokenUser, tokenValid := verifyToken(r)
		if !tokenValid || tokenUser != username {
			errResp(w, "unauthorized", 401)
			return
		}
		limit := 50
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" { fmt.Sscanf(l, "%d", &limit) }
		if o := r.URL.Query().Get("offset"); o != "" { fmt.Sscanf(o, "%d", &offset) }
		rows, err := db.Query("SELECT id, content, created_at, updated_at, pinned, tags, username, avatar, nickname, deleted_at FROM trash WHERE username=? ORDER BY deleted_at DESC LIMIT ? OFFSET ?", username, limit, offset)
		if err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		defer rows.Close()
		var items []trashItemResponse
		for rows.Next() {
			var id, content, uname, tags, avatar, nickname string
			var createdAt, updatedAt, deletedAt int64
			var pinned int
			if err := rows.Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &uname, &avatar, &nickname, &deletedAt); err != nil {
				continue
			}
			var tagList []string
			if err := json.Unmarshal([]byte(tags), &tagList); err != nil {
				log.Printf("failed to parse tags from trash item %s: %v", id, err)
			}
			items = append(items, trashItemResponse{
				ID: id, Content: content, CreatedAt: createdAt, UpdatedAt: updatedAt,
				Pinned: pinned == 1, Tags: tagList, Username: uname,
				Avatar: avatar, Nickname: nickname, DeletedAt: deletedAt,
			})
		}
		if err := rows.Err(); err != nil {
			errResp(w, "查询数据时发生错误", 500)
			return
		}
		if items == nil {
			items = []trashItemResponse{}
		}
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
		if err != nil {
			errResp(w, "not found in trash", 404)
			return
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO notes (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname) VALUES (?,?,?,?,?,?,?,?,?)", noteId, cont, ct, ut, p, ts, username, av, nk); err != nil {
			errResp(w, "恢复失败", 500)
			return
		}
		if err := execSQL("DELETE FROM trash WHERE id=?", noteId); err != nil {
			errResp(w, "恢复失败", 500)
			return
		}
		sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
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
		result, err := db.Exec("DELETE FROM trash WHERE id=? AND username=?", noteId, username)
		if err != nil {
			errResp(w, "删除失败", 500)
			return
		}
		affected, err := result.RowsAffected()
		if err != nil || affected == 0 {
			errResp(w, "not found", 404)
			return
		}
		sseBroadcast("note", "updated")
		jsonResp(w, successResponse{Success: "ok"})
		return
	}
	errResp(w, "not found", 404)
}

func batchGetReactions(ids []string) map[string]map[string][]string {
	if len(ids) == 0 {
		return map[string]map[string][]string{}
	}
	// Build placeholder string: ?,?,?,...
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.Query("SELECT id, emoji, username FROM reactions WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		log.Printf("batchGetReactions query failed: %v", err)
		return map[string]map[string][]string{}
	}
	defer rows.Close()
	result := make(map[string]map[string][]string)
	for rows.Next() {
		var id, emoji, username string
		if err := rows.Scan(&id, &emoji, &username); err != nil {
			continue
		}
		if result[id] == nil {
			result[id] = map[string][]string{}
		}
		result[id][emoji] = append(result[id][emoji], username)
	}
	if err := rows.Err(); err != nil {
		log.Printf("batchGetReactions iteration error: %v", err)
	}
	return result
}

// handleNotesExport exports all notes for the authenticated user as JSON.
// GET /api/notes/export?username=xxx
func handleNotesExport(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		errResp(w, "username required", 400)
		return
	}
	tokenUser, tokenValid := verifyToken(r)
	if !tokenValid || tokenUser != username {
		errResp(w, "unauthorized", 401)
		return
	}
	rows, err := db.Query("SELECT id, content, created_at, updated_at, pinned, tags, username, avatar, nickname FROM notes WHERE username=? ORDER BY created_at DESC", username)
	if err != nil {
		errResp(w, "查询数据时发生错误", 500)
		return
	}
	defer rows.Close()
	var notes []noteResponse
	for rows.Next() {
		var id, content, uname, tags, avatar, nickname string
		var createdAt, updatedAt int64
		var pinned int
		if err := rows.Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &uname, &avatar, &nickname); err != nil {
			continue
		}
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)
		notes = append(notes, noteResponse{
			ID: id, Content: content, CreatedAt: createdAt, UpdatedAt: updatedAt,
			Pinned: pinned == 1, Tags: tagList, Username: uname,
			Avatar: avatar, Nickname: nickname,
		})
	}
	if err := rows.Err(); err != nil {
		errResp(w, "查询数据时发生错误", 500)
		return
	}
	if notes == nil {
		notes = []noteResponse{}
	}
	data, _ := json.MarshalIndent(notes, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"suisui-notes-"+username+".json\"")
	w.Write(data)
}

// handleNotesImport imports notes from a JSON array.
// POST /api/notes/import
func handleNotesImport(w http.ResponseWriter, r *http.Request) {
	tokenUser, tokenValid := verifyToken(r)
	if !tokenValid {
		errResp(w, "unauthorized", 401)
		return
	}
	var notes []struct {
		Id        string   `json:"id"`
		Content   string   `json:"content"`
		CreatedAt int64    `json:"createdAt"`
		UpdatedAt int64    `json:"updatedAt"`
		Pinned    bool     `json:"pinned"`
		Tags      []string `json:"tags"`
		Username  string   `json:"username"`
		Avatar    string   `json:"avatar"`
		Nickname  string   `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&notes); err != nil {
		errResp(w, "无效的请求数据", 400)
		return
	}
	imported := 0
	for _, n := range notes {
		// Only import notes belonging to the authenticated user
		if n.Username != tokenUser {
			continue
		}
		tagBytes, _ := json.Marshal(n.Tags)
		pinned := 0
		if n.Pinned {
			pinned = 1
		}
		_, err := db.Exec("INSERT OR IGNORE INTO notes (id, content, created_at, updated_at, pinned, tags, username, avatar, nickname) VALUES (?,?,?,?,?,?,?,?,?)",
			n.Id, n.Content, n.CreatedAt, n.UpdatedAt, pinned, string(tagBytes), n.Username, n.Avatar, n.Nickname)
		if err == nil {
			imported++
		}
	}
	jsonResp(w, importResponse{Imported: imported})
}

// handleShareView returns a note by its ID (public, no auth).
// GET /api/share/:id
func handleShareView(w http.ResponseWriter, r *http.Request) {
	noteId := strings.TrimPrefix(r.URL.Path, "/api/share/")
	if noteId == "" {
		errResp(w, "missing note id", 400)
		return
	}

	var id, content, username, tags, avatar, nickname string
	var createdAt, updatedAt int64
	var pinned int
	err := db.QueryRow("SELECT id, content, created_at, updated_at, pinned, tags, username, avatar, nickname FROM notes WHERE id=?", noteId).
		Scan(&id, &content, &createdAt, &updatedAt, &pinned, &tags, &username, &avatar, &nickname)
	if err != nil {
		errResp(w, "笔记不存在", 404)
		return
	}

	var tagList []string
	json.Unmarshal([]byte(tags), &tagList)

	// Load reactions
	reactions := batchGetReactions([]string{noteId})
	rMap := reactions[noteId]
	if rMap == nil {
		rMap = map[string][]string{}
	}

	jsonResp(w, noteResponse{
		ID: id, Content: content, CreatedAt: createdAt, UpdatedAt: updatedAt,
		Pinned: pinned == 1, Tags: tagList, Username: username,
		Avatar: avatar, Nickname: nickname, Reactions: rMap,
	})
}
