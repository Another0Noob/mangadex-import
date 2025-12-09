package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandleQueue returns queue information for a given user.
// GET /api/queue?clientID=...
// Response: {"position": <int>, "queued": <int>}
func (api *MangaAPI) HandleQueue(w http.ResponseWriter, r *http.Request) {
	// Prefer explicit session_id
	sessionID := r.URL.Query().Get("session_id")

	api.queueMu.Lock()
	defer api.queueMu.Unlock()

	total := len(api.queueOrder)
	position := 0
	if sessionID != "" {
		for i, id := range api.queueOrder {
			if id == sessionID {
				position = i + 1 // 1-based position
				break
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]int{
		"position": position,
		"queued":   total,
	})
}

// HandleQueueSubscribe streams queue changes to subscribers via SSE.
func (api *MangaAPI) HandleQueueSubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []string, 2)
	api.queueSubsMu.Lock()
	api.queueSubs[ch] = struct{}{}
	api.queueSubsMu.Unlock()

	// Send initial snapshot
	api.queueMu.Lock()
	initial := make([]string, len(api.queueOrder))
	copy(initial, api.queueOrder)
	api.queueMu.Unlock()

	// Write initial event
	data, _ := json.Marshal(map[string]any{"queue_order": initial, "queued": len(initial)})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Use request context to watch for client disconnects c
	notify := r.Context().Done()
	for {
		select {
		case q := <-ch:
			data, _ := json.Marshal(map[string]any{"queue_order": q, "queued": len(q)})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-notify:
			// client disconnected
			api.queueSubsMu.Lock()
			delete(api.queueSubs, ch)
			api.queueSubsMu.Unlock()
			return
		}
	}
}

func (api *MangaAPI) broadcastQueueLocked() {
	// caller must hold queueMu
	api.queueSubsMu.Lock()
	defer api.queueSubsMu.Unlock()
	snapshot := make([]string, len(api.queueOrder))
	copy(snapshot, api.queueOrder)
	for ch := range api.queueSubs {
		// non-blocking send: spawn goroutine to avoid blocking
		go func(c chan []string, s []string) {
			select {
			case c <- s:
			default:
				// if subscriber is slow / full, drop this update
			}
		}(ch, snapshot)
	}
}

// removeQueuedUser removes a user from the queueOrder slice (if present).
func (api *MangaAPI) removeQueuedSession(sessionID string) {
	api.queueMu.Lock()
	defer api.queueMu.Unlock()
	for i, id := range api.queueOrder {
		if id == sessionID {
			api.queueOrder = append(api.queueOrder[:i], api.queueOrder[i+1:]...)
			// notify subscribers of the changed queue
			api.broadcastQueueLocked()
			return
		}
	}
}
