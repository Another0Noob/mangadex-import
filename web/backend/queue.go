package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (api *MangaAPI) HandleQueue(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessionID := r.URL.Query().Get("session_id")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 1)

	api.queueSubsMu.Lock()
	api.queueSubs[ch] = struct{}{}
	api.queueSubsMu.Unlock()

	defer func() {
		api.queueSubsMu.Lock()
		delete(api.queueSubs, ch)
		api.queueSubsMu.Unlock()
	}()

	sendUpdate := func() {
		api.queueMu.Lock()
		pos := queuePosition(api.queueOrder, sessionID)
		total := len(api.queueOrder)
		api.queueMu.Unlock()

		data, _ := json.Marshal(map[string]int{
			"position": pos,
			"queued":   total,
		})

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Initial event
	sendUpdate()

	ctx := r.Context()
	for {
		select {
		case <-ch:
			sendUpdate()
		case <-ctx.Done():
			return
		}
	}
}

func (api *MangaAPI) broadcastQueueLocked() {
	// caller must hold queueMu
	api.queueSubsMu.Lock()
	defer api.queueSubsMu.Unlock()

	for ch := range api.queueSubs {
		select {
		case ch <- struct{}{}:
		default:
			// drop update for slow subscriber
		}
	}
}

// removeQueuedUser removes a user from the queueOrder slice (if present).
func (api *MangaAPI) removeQueuedSession(sessionID string) {
	api.queueMu.Lock()
	defer api.queueMu.Unlock()

	for i, id := range api.queueOrder {
		if id == sessionID {
			api.queueOrder = append(api.queueOrder[:i], api.queueOrder[i+1:]...)
			api.broadcastQueueLocked()
			return
		}
	}
}

func queuePosition(queue []string, sessionID string) int {
	if sessionID == "" {
		return 0
	}
	for i, id := range queue {
		if id == sessionID {
			return i + 1
		}
	}
	return 0
}
