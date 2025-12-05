package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/Another0Noob/mangadex-import/internal/mangaparser"
	"github.com/Another0Noob/mangadex-import/internal/match"
)

type FollowJob struct {
	Req FollowRequest
}

func (fj FollowJob) Run(api *MangaAPI, session *UserSession) {
	// Reuse existing runFollowAsync which is synchronous and handles removal.
	api.runFollowAsync(session, fj.Req)
}

// FollowRequest holds the parameters for a follow operation
type FollowRequest struct {
	UserID        string
	Username      string // MangaDex username
	Password      string // MangaDex password
	ClientID      string // MangaDex OAuth client ID
	ClientSecret  string // MangaDex OAuth client secret
	InputFile     []byte // Manga list file content
	InputFilename string // original uploaded filename
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

	inputFile, fileHeader, err := r.FormFile("manga_list")
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

	// Sanitize uploaded filename (strip any path components)
	var filename string
	if fileHeader != nil && fileHeader.Filename != "" {
		filename = filepath.Base(fileHeader.Filename)
	}

	req := FollowRequest{
		UserID:        userID,
		Username:      username,
		Password:      password,
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		InputFile:     inputData,
		InputFilename: filename,
	}

	// Create a new session for this user
	session, err := api.sessions.CreateSession(req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	// Enqueue the job so only one will run at a time.
	api.queueMu.Lock()
	select {
	case api.jobQueue <- queuedJob{session: session, job: FollowJob{Req: req}}:
		// Enqueued successfully; track order by session.ID
		api.queueOrder = append(api.queueOrder, session.ID)
		api.queueMu.Unlock()
	default:
		// Queue full - inform client
		api.queueMu.Unlock()
		api.sessions.RemoveSession(req.UserID) // cleanup the session we created
		http.Error(w, "Server busy, try again later", http.StatusTooManyRequests)
		return
	}

	// Return session ID for tracking
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"user_id":    req.UserID,
		"status":     "queued",
	})
}

// runFollowAsync executes the follow operation with progress updates
// (existing implementation reused; no signature changes)
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
	inputManga, err := mangaparser.ParseFromBytes(req.InputFile, req.InputFilename)
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
	if err := session.Client.LoadAuthForm(authForm); err != nil {
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
