package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const size = 100

// Job is a unit of work that can be executed by the worker.
type Job interface {
	// Run executes the job. It receives the API and the session for progress/cancellation.
	Run(api *MangaAPI, session *UserSession)
}

// queuedJob is an item in the processing queue
type queuedJob struct {
	session *UserSession
	job     Job
}

// API Handler
type MangaAPI struct {
	sessions *SessionManager

	// queue so that only one follow job runs at a time
	jobQueue  chan queuedJob
	queueSize int

	// queueOrder tracks session IDs in enqueue order (protected by queueMu)
	queueMu    sync.Mutex
	queueOrder []string

	// queue SSE subscribers
	queueSubs   map[chan struct{}]struct{}
	queueSubsMu sync.Mutex
}

func NewMangaAPI() *MangaAPI {
	api := &MangaAPI{
		sessions:   NewSessionManager(),
		queueSize:  size,                       // tune as you like
		jobQueue:   make(chan queuedJob, size), // buffered queue
		queueOrder: make([]string, 0, size),
	}

	api.queueSubs = make(map[chan struct{}]struct{})

	// Start single worker to process jobs sequentially
	go func() {
		for job := range api.jobQueue {
			// Remove job from queueOrder (if present) before processing.
			api.queueMu.Lock()
			if len(api.queueOrder) > 0 && api.queueOrder[0] == job.session.ID {
				// common fast path: pop front
				api.queueOrder = api.queueOrder[1:]
			} else {
				// fallback: find and remove by session ID
				for i, id := range api.queueOrder {
					if id == job.session.ID {
						api.queueOrder = append(api.queueOrder[:i], api.queueOrder[i+1:]...)
						break
					}
				}
			}
			api.queueMu.Unlock()

			// If the session was already removed/cancelled, skip
			if job.session == nil || job.session.Ctx == nil {
				continue
			}
			// Dispatch the job
			job.job.Run(api, job.session)
		}
	}()

	// Cleanup stale sessions every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			api.sessions.CleanupStale(24 * time.Hour)
		}
	}()

	return api
}

// HandleProgress streams progress updates via SSE
func (api *MangaAPI) HandleProgress(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	userID := r.URL.Query().Get("user_id")

	var session *UserSession
	var ok bool
	if sessionID != "" {
		session, ok = api.sessions.GetSessionByID(sessionID) // you might need to add this helper
	} else if userID != "" {
		session, ok = api.sessions.GetSession(userID)
	}

	if !ok || session == nil {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Listen for client disconnect via request context
	notify := r.Context().Done()

	// Stream progress updates
	for {
		select {
		case update, okCh := <-session.Progress:
			if !okCh {
				// session progress channel closed, finish
				return
			}
			data, _ := json.Marshal(update)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			if update.Type == "complete" || update.Type == "error" {
				return
			}
		case <-notify:
			// client disconnected
			return
		}
	}
}

// HandleCancel allows users to cancel their operation
func (api *MangaAPI) HandleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	userID := r.URL.Query().Get("user_id")

	// Prefer session_id (server-generated session identifier)
	if sessionID != "" {
		if session, ok := api.sessions.GetSessionByID(sessionID); ok {
			api.removeQueuedSession(session.ID)
			// Remove session from sessions map by finding its user key.
			api.sessions.RemoveBySessionID(sessionID)
			json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
			return
		}
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	// legacy path: user_id
	if userID == "" {
		http.Error(w, "user_id or session_id required", http.StatusBadRequest)
		return
	}

	if session, ok := api.sessions.GetSession(userID); ok {
		api.removeQueuedSession(session.ID)
	}
	api.sessions.RemoveSession(userID)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}
