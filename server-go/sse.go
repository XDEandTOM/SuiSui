package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSE event hub for note updates
var sseClients = make(map[chan string]bool)
var sseMu sync.Mutex

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
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	flusher, ok := w.(http.Flusher)
	if !ok { errResp(w, "streaming not supported", 500); return }
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

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
