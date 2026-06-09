package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database and initializes the schema.
func setupTestDB(t *testing.T) {
	t.Helper()
	var err error
	db, err = sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	// Create tables manually (initDB also does file I/O and PRAGMA).
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, nickname TEXT DEFAULT '', avatar TEXT DEFAULT '', role TEXT DEFAULT 'user', created_at INTEGER DEFAULT 0, theme_color TEXT DEFAULT '#1976D2', app_icon TEXT DEFAULT '', salt TEXT DEFAULT '')`,
		`CREATE TABLE IF NOT EXISTS notes (id TEXT PRIMARY KEY, content TEXT, created_at INTEGER, updated_at INTEGER, pinned INTEGER DEFAULT 0, tags TEXT DEFAULT '[]', username TEXT, avatar TEXT, nickname TEXT)`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE IF NOT EXISTS reactions (id TEXT, emoji TEXT, username TEXT, PRIMARY KEY (id, emoji, username))`,
		`CREATE TABLE IF NOT EXISTS trash (id TEXT PRIMARY KEY, content TEXT, created_at INTEGER, updated_at INTEGER, pinned INTEGER DEFAULT 0, tags TEXT DEFAULT '[]', username TEXT, avatar TEXT, nickname TEXT, deleted_at INTEGER)`,
		`CREATE TABLE IF NOT EXISTS auth_tokens (token TEXT PRIMARY KEY, username TEXT, created_at INTEGER)`,
	}
	for _, tbl := range tables {
		if _, err := db.Exec(tbl); err != nil {
			t.Fatal(err)
		}
	}
}

// --- Unit tests for utility functions ---

func TestGenerateSalt(t *testing.T) {
	s1 := generateSalt()
	s2 := generateSalt()
	if s1 == "" || s2 == "" {
		t.Fatal("salt should not be empty")
	}
	if s1 == s2 {
		t.Fatal("two salts should differ")
	}
	if len(s1) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32 hex chars, got %d", len(s1))
	}
}

func TestHashPassword(t *testing.T) {
	salt := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
	h := hashPassword("hello", salt)
	if h == "" {
		t.Fatal("hash should not be empty")
	}
	if !checkPassword("hello", h, salt) {
		t.Fatal("checkPassword should match correct password")
	}
	if checkPassword("wrong", h, salt) {
		t.Fatal("checkPassword should reject wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	t1 := generateToken()
	t2 := generateToken()
	if t1 == "" || t2 == "" {
		t.Fatal("token should not be empty")
	}
	if t1 == t2 {
		t.Fatal("two tokens should differ")
	}
	if len(t1) != 64 { // 32 bytes = 64 hex chars
		t.Fatalf("expected 64 hex chars, got %d", len(t1))
	}
}

// --- Integration tests for auth endpoints ---

func TestAuthRegisterAndLogin(t *testing.T) {
	setupTestDB(t)

	// Register
	regBody := `{"username":"testuser","password":"pass1234"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleAPI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("register returned %d: %s", w.Code, w.Body.String())
	}
	var regResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatal(err)
	}
	if regResp["token"].(string) == "" {
		t.Fatal("register should return a token")
	}

	// Login with correct credentials
	loginBody := `{"username":"testuser","password":"pass1234"}`
	req2 := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handleAPI(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("login returned %d: %s", w2.Code, w2.Body.String())
	}
	var loginResp map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}
	if loginResp["token"].(string) == "" {
		t.Fatal("login should return a token")
	}

	// Login with wrong password
	loginBody3 := `{"username":"testuser","password":"wrongpass"}`
	req3 := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handleAPI(w3, req3)

	if w3.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password should return 401, got %d", w3.Code)
	}

	// Duplicate register should fail
	w4 := httptest.NewRecorder()
	handleAPI(w4, httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody)))
	if w4.Code != http.StatusConflict {
		t.Fatalf("duplicate register should return 409, got %d", w4.Code)
	}
}

func TestAuthVerify(t *testing.T) {
	setupTestDB(t)

	// Register first
	handleAPI(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"username":"verifyuser","password":"pass1234"}`)))

	// Verify with valid token
	req := httptest.NewRequest("GET", "/api/auth/verify?username=verifyuser", nil)
	// Need to get a real token — login
	w := httptest.NewRecorder()
	handleAPI(w, httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"username":"verifyuser","password":"pass1234"}`)))
	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	req.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	handleAPI(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("verify returned %d", w2.Code)
	}
	var verifyResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &verifyResp)
	if verifyResp["valid"] != true {
		t.Fatal("token should be valid")
	}
}

// --- Integration tests for notes endpoints ---

func TestNotesCRUD(t *testing.T) {
	setupTestDB(t)

	// Register & login to get a working session
	handleAPI(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"username":"noter","password":"pass1234"}`)))
	w := httptest.NewRecorder()
	handleAPI(w, httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"username":"noter","password":"pass1234"}`)))
	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	authHeader := "Bearer " + token

	// Create a note
	noteBody := `{"id":"note1","content":"Hello World","tags":["test"],"username":"noter","createdAt":1700000000000,"updatedAt":1700000000000}`
	req := httptest.NewRequest("POST", "/api/notes", strings.NewReader(noteBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	w2 := httptest.NewRecorder()
	handleAPI(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("create note returned %d: %s", w2.Code, w2.Body.String())
	}

	// List notes
	req3 := httptest.NewRequest("GET", "/api/notes", nil)
	req3.Header.Set("Authorization", authHeader)
	w3 := httptest.NewRecorder()
	handleAPI(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("list notes returned %d", w3.Code)
	}
	var notes []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &notes)
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0]["content"] != "Hello World" {
		t.Fatalf("unexpected content: %v", notes[0]["content"])
	}
}

func TestNotesPagination(t *testing.T) {
	setupTestDB(t)

	// Register & login
	handleAPI(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"username":"pager","password":"pass1234"}`)))
	w := httptest.NewRecorder()
	handleAPI(w, httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"username":"pager","password":"pass1234"}`)))
	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)
	authHeader := "Bearer " + token

	// Create 3 notes
	for i := 0; i < 3; i++ {
		body := `{"id":"pnote` + string(rune('0'+i)) + `","content":"Note ` + string(rune('0'+i)) + `","tags":[],"username":"pager","createdAt":1700000000000,"updatedAt":1700000000000}`
		req := httptest.NewRequest("POST", "/api/notes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		handleAPI(httptest.NewRecorder(), req)
	}

	// Fetch with limit=1
	req := httptest.NewRequest("GET", "/api/notes?limit=1", nil)
	req.Header.Set("Authorization", authHeader)
	w2 := httptest.NewRecorder()
	handleAPI(w2, req)

	var notes []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &notes)
	if len(notes) != 1 {
		t.Fatalf("expected 1 note with limit=1, got %d", len(notes))
	}
}
