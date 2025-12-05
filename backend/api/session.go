package api

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
