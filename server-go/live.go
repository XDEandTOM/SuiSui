package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Live chat
var liveChat []string
var liveChatMu sync.Mutex
var liveSSEClients = make(map[chan string]bool)
var liveSSEMu sync.Mutex

func liveChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var msg struct { Text string `json:"text"`; Author string `json:"author"` }
		if json.NewDecoder(r.Body).Decode(&msg) != nil || msg.Text == "" { errResp(w, "bad request", 400); return }
		payload := msg.Author + ": " + msg.Text
		liveChatMu.Lock()
		liveChat = append(liveChat, payload)
		if len(liveChat) > 100 { liveChat = liveChat[len(liveChat)-100:] }
		liveChatMu.Unlock()
		data, _ := json.Marshal(map[string]string{"text": msg.Text, "author": msg.Author})
		liveSSEMu.Lock()
		for ch := range liveSSEClients {
			select { case ch <- string(data): default: close(ch); delete(liveSSEClients, ch) }
		}
		liveSSEMu.Unlock()
		jsonResp(w, map[string]bool{"ok": true})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
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
	var url string
	db.QueryRow("SELECT value FROM settings WHERE key='live_stream_url'").Scan(&url)
	jsonResp(w, map[string]bool{"online": url != ""})
}

func handleLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(livePage))
}
