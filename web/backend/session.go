package backend

import (
	"context"
	"sync"
	"time"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/google/uuid"
)

// ProgressUpdate represents a status update during processing
type ProgressUpdate struct {
	Type    string `json:"type"` // "info", "progress", "error", "complete"
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
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

func (sm *SessionManager) CreateSession(clientID string) (*UserSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if user already has an active session
	if existing, ok := sm.sessions[clientID]; ok {
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

	sm.sessions[clientID] = session
	return session, nil
}

func (sm *SessionManager) GetSession(clientID string) (*UserSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[clientID]
	return session, ok
}

// GetSessionByID finds a session by its internal session ID (not the clientID).
// Returns the session and true if found.
func (sm *SessionManager) GetSessionByID(sessionID string) (*UserSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, session := range sm.sessions {
		if session != nil && session.ID == sessionID {
			return session, true
		}
	}
	return nil, false
}

func (sm *SessionManager) RemoveSession(clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if session, ok := sm.sessions[clientID]; ok {
		session.CancelFn()
		close(session.Progress)
		delete(sm.sessions, clientID)
	}
}

// CleanupStale removes sessions older than duration
func (sm *SessionManager) CleanupStale(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for clientID, session := range sm.sessions {
		if now.Sub(session.CreatedAt) > maxAge {
			session.CancelFn()
			close(session.Progress)
			delete(sm.sessions, clientID)
		}
	}
}

// RemoveBySessionID removes session by internal session.ID (searches the map).
func (sm *SessionManager) RemoveBySessionID(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for userKey, session := range sm.sessions {
		if session != nil && session.ID == sessionID {
			session.CancelFn()
			close(session.Progress)
			delete(sm.sessions, userKey)
			return
		}
	}
}
