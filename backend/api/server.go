package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/Another0Noob/mangadex-import/internal/mangaparser"
	"github.com/Another0Noob/mangadex-import/internal/match"
	"github.com/google/uuid"
)

// ProgressUpdate represents a status update during processing
type ProgressUpdate struct {
	Type    string `json:"type"` // "info", "progress", "error", "complete"
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// FollowRequest holds the parameters for a follow operation
type FollowRequest struct {
	UserID       string
	Username     string // MangaDex username
	Password     string // MangaDex password
	ClientID     string // MangaDex OAuth client ID
	ClientSecret string // MangaDex OAuth client secret
	InputFile    []byte // Manga list file content
}

// UserSession manages resources for a single user's operation
type UserSession struct {
	ID        string
	Client    *mangadexapi.Client
	Progress  chan ProgressUpdate
	Ctx       context.Context
	CancelFn  context.CancelFunc
	CreatedAt time.Time
}

// SessionManager handles concurrent user sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*UserSession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*UserSession),
	}
}

func (sm *SessionManager) CreateSession(userID string) (*UserSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if user already has an active session
	if existing, ok := sm.sessions[userID]; ok {
		// Cancel the old session
		existing.CancelFn()
		close(existing.Progress)
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &UserSession{
		ID:        uuid.New().String(),
		Client:    mangadexapi.NewClient(),
		Progress:  make(chan ProgressUpdate, 100),
		Ctx:       ctx,
		CancelFn:  cancel,
		CreatedAt: time.Now(),
	}

	sm.sessions[userID] = session
	return session, nil
}

func (sm *SessionManager) GetSession(userID string) (*UserSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[userID]
	return session, ok
}

func (sm *SessionManager) RemoveSession(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if session, ok := sm.sessions[userID]; ok {
		session.CancelFn()
		close(session.Progress)
		delete(sm.sessions, userID)
	}
}

// CleanupStale removes sessions older than duration
func (sm *SessionManager) CleanupStale(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for userID, session := range sm.sessions {
		if now.Sub(session.CreatedAt) > maxAge {
			session.CancelFn()
			close(session.Progress)
			delete(sm.sessions, userID)
		}
	}
}

// API Handler
type MangaAPI struct {
	sessions *SessionManager
}

func NewMangaAPI() *MangaAPI {
	api := &MangaAPI{
		sessions: NewSessionManager(),
	}

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

// HandleFollow starts the follow operation for a user
func (api *MangaAPI) HandleFollow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	// Get MangaDex credentials from form
	username := r.FormValue("username")
	password := r.FormValue("password")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if username == "" || password == "" || clientID == "" || clientSecret == "" {
		http.Error(w, "All MangaDex credentials required", http.StatusBadRequest)
		return
	}

	// Read manga list file
	inputFile, _, err := r.FormFile("manga_list")
	if err != nil {
		http.Error(w, "manga_list file required", http.StatusBadRequest)
		return
	}
	defer inputFile.Close()
	inputData, err := io.ReadAll(inputFile)
	if err != nil {
		http.Error(w, "Failed to read manga list file", http.StatusInternalServerError)
		return
	}

	req := FollowRequest{
		UserID:       userID,
		Username:     username,
		Password:     password,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		InputFile:    inputData,
	}

	// Create a new session for this user
	session, err := api.sessions.CreateSession(req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	// Start the follow operation in a goroutine
	go api.runFollowAsync(session, req)

	// Return session ID for tracking
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"user_id":    req.UserID,
		"status":     "started",
	})
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

// runFollowAsync executes the follow operation with progress updates
func (api *MangaAPI) runFollowAsync(session *UserSession, req FollowRequest) {
	defer api.sessions.RemoveSession(req.UserID)

	// Use the session's cancellable context with a timeout
	ctx, cancel := context.WithTimeout(session.Ctx, 30*time.Minute)
	defer cancel()

	// Helper to send progress
	sendProgress := func(typ, msg string, data any) {
		select {
		case session.Progress <- ProgressUpdate{Type: typ, Message: msg, Data: data}:
		default:
			// Channel full, skip this update
		}
	}

	// Check if already cancelled
	if ctx.Err() != nil {
		sendProgress("error", "Operation cancelled", nil)
		return
	}

	// Parse manga list directly from memory
	sendProgress("info", "Reading manga list...", nil)
	inputManga, err := mangaparser.ParseFromBytes(req.InputFile, "manga-list.json")
	if err != nil {
		sendProgress("error", fmt.Sprintf("Failed to parse file: %v", err), nil)
		return
	}
	sendProgress("info", fmt.Sprintf("Got %d manga", len(inputManga)), map[string]int{"count": len(inputManga)})

	sendProgress("info", "Authenticating with MangaDex...", nil)
	authForm := mangadexapi.AuthForm{
		Username:     req.Username,
		Password:     req.Password,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	}
	if err := session.Client.LoadAuthFrom(authForm); err != nil {
		sendProgress("error", fmt.Sprintf("Failed to load auth: %v", err), nil)
		return
	}
	if err := session.Client.Authenticate(ctx); err != nil {
		sendProgress("error", fmt.Sprintf("Authentication failed: %v", err), nil)
		return
	}

	sendProgress("info", "Fetching followed manga from MangaDex...", nil)
	followedManga, err := session.Client.GetAllFollowed(ctx)
	if err != nil {
		sendProgress("error", fmt.Sprintf("Failed to get followed manga: %v", err), nil)
		return
	}
	sendProgress("info", fmt.Sprintf("Got %d MangaDex manga", len(followedManga)), map[string]int{"count": len(followedManga)})

	sendProgress("info", "Matching manga...", nil)
	matchResult := match.MatchDirect(followedManga, inputManga)
	countDirect := len(matchResult.Matches)
	sendProgress("progress", fmt.Sprintf("Matched %d manga directly", countDirect), map[string]int{"direct_matches": countDirect})

	matchResult = match.FuzzyMatch(matchResult)
	sendProgress("progress", fmt.Sprintf("Fuzzy matched %d manga", len(matchResult.Matches)-countDirect),
		map[string]int{"fuzzy_matches": len(matchResult.Matches) - countDirect})

	sendProgress("info", "Searching for unmatched manga...", nil)
	newMatches, stillUnmatched, err := match.SearchAndFollow(ctx, session.Client, matchResult.Unmatched.Import, true)
	if err != nil {
		// Check if error is due to cancellation
		if ctx.Err() == context.Canceled {
			sendProgress("error", "Operation cancelled by user", nil)
		} else {
			sendProgress("error", fmt.Sprintf("Search failed: %v", err), nil)
		}
		return
	}

	sendProgress("complete", "Operation completed", map[string]any{
		"direct_matches":  countDirect,
		"fuzzy_matches":   len(matchResult.Matches) - countDirect,
		"new_matches":     len(newMatches),
		"still_unmatched": len(stillUnmatched),
	})
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

	api.sessions.RemoveSession(userID)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func main() {
	api := NewMangaAPI()

	http.HandleFunc("/api/follow", api.HandleFollow)
	http.HandleFunc("/api/progress", api.HandleProgress)
	http.HandleFunc("/api/cancel", api.HandleCancel)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}
