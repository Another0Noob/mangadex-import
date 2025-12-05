package api

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
}

func NewMangaAPI() *MangaAPI {
	api := &MangaAPI{
		sessions:   NewSessionManager(),
		queueSize:  size,                       // tune as you like
		jobQueue:   make(chan queuedJob, size), // buffered queue
		queueOrder: make([]string, 0, size),
	}

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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	session, ok := api.sessions.GetSession(userID)
	if !ok {
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

	// Stream progress updates
	for update := range session.Progress {
		data, _ := json.Marshal(update)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if update.Type == "complete" || update.Type == "error" {
			break
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

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	// If there's a session, remove its queued entry (tracked by session.ID)
	if session, ok := api.sessions.GetSession(userID); ok {
		api.removeQueuedSession(session.ID)
	}

	api.sessions.RemoveSession(userID)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// HandleQueue returns queue information for a given user.
// GET /api/queue?user_id=...
// Response: {"position": <int>, "queued": <int>}
func (api *MangaAPI) HandleQueue(w http.ResponseWriter, r *http.Request) {
	// Prefer explicit session_id
	sessionID := r.URL.Query().Get("session_id")
	userID := r.URL.Query().Get("user_id")

	// If session_id is not provided but user_id is, try to find session and use its ID
	if sessionID == "" && userID != "" {
		if session, ok := api.sessions.GetSession(userID); ok {
			sessionID = session.ID
		}
	}

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

func RunApi() {
	api := NewMangaAPI()

	http.HandleFunc("/api/follow", api.HandleFollow)
	http.HandleFunc("/api/progress", api.HandleProgress)
	http.HandleFunc("/api/cancel", api.HandleCancel)
	http.HandleFunc("/api/queue", api.HandleQueue)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}
