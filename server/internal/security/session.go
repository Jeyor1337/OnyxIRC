package security

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "sync"
    "time"

    "github.com/onyxirc/server/internal/auth"
    "github.com/onyxirc/server/internal/models"
)

// Session represents an active user session
type Session struct {
    SessionID    string
    UserID       int64
    User         *models.User
    IPAddress    string
    SessionKey   []byte // AES session key for encryption
    CreatedAt    time.Time
    LastActivity time.Time
    ExpiresAt    time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
    sessions       map[string]*Session // sessionID -> Session
    userSessions   map[int64][]string  // userID -> []sessionID
    mu             sync.RWMutex
    sessionTimeout time.Duration
}

// NewSessionManager creates a new SessionManager
func NewSessionManager(sessionTimeout time.Duration) *SessionManager {
    sm := &SessionManager{
        sessions:       make(map[string]*Session),
        userSessions:   make(map[int64][]string),
        sessionTimeout: sessionTimeout,
    }

    // Start cleanup goroutine
    go sm.cleanupExpiredSessions()

    return sm
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(user *models.User, ipAddress string, sessionKey []byte) (*Session, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Generate session ID
    sessionID, err := generateSessionID()
    if err != nil {
        return nil, fmt.Errorf("failed to generate session ID: %w", err)
    }

    // Create session
    now := time.Now()
    session := &Session{
        SessionID:    sessionID,
        UserID:       user.UserID,
        User:         user,
        IPAddress:    ipAddress,
        SessionKey:   sessionKey,
        CreatedAt:    now,
        LastActivity: now,
        ExpiresAt:    now.Add(sm.sessionTimeout),
    }

    // Store session
    sm.sessions[sessionID] = session

    // Track user sessions
    if _, exists := sm.userSessions[user.UserID]; !exists {
        sm.userSessions[user.UserID] = []string{}
    }
    sm.userSessions[user.UserID] = append(sm.userSessions[user.UserID], sessionID)

    return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return nil, fmt.Errorf("session not found")
    }

    // Check if session is expired
    if time.Now().After(session.ExpiresAt) {
        return nil, fmt.Errorf("session expired")
    }

    return session, nil
}

// UpdateActivity updates the last activity time for a session
func (sm *SessionManager) UpdateActivity(sessionID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }

    now := time.Now()
    session.LastActivity = now
    session.ExpiresAt = now.Add(sm.sessionTimeout)

    return nil
}

// DestroySession destroys a session
func (sm *SessionManager) DestroySession(sessionID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }

    // Remove from sessions map
    delete(sm.sessions, sessionID)

    // Remove from user sessions
    userSessions := sm.userSessions[session.UserID]
    for i, sid := range userSessions {
        if sid == sessionID {
            sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
            break
        }
    }

    // Clean up empty user session list
    if len(sm.userSessions[session.UserID]) == 0 {
        delete(sm.userSessions, session.UserID)
    }

    return nil
}

// DestroyUserSessions destroys all sessions for a user
func (sm *SessionManager) DestroyUserSessions(userID int64) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    sessionIDs, exists := sm.userSessions[userID]
    if !exists {
        return nil // No sessions to destroy
    }

    // Remove all sessions
    for _, sessionID := range sessionIDs {
        delete(sm.sessions, sessionID)
    }

    // Remove user sessions entry
    delete(sm.userSessions, userID)

    return nil
}

// GetUserSessions retrieves all active sessions for a user
func (sm *SessionManager) GetUserSessions(userID int64) []*Session {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    sessionIDs, exists := sm.userSessions[userID]
    if !exists {
        return []*Session{}
    }

    sessions := make([]*Session, 0, len(sessionIDs))
    for _, sessionID := range sessionIDs {
        if session, exists := sm.sessions[sessionID]; exists {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    return len(sm.sessions)
}

// cleanupExpiredSessions periodically removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        sm.mu.Lock()

        now := time.Now()
        expiredSessions := []string{}

        // Find expired sessions
        for sessionID, session := range sm.sessions {
            if now.After(session.ExpiresAt) {
                expiredSessions = append(expiredSessions, sessionID)
            }
        }

        // Remove expired sessions
        for _, sessionID := range expiredSessions {
            session := sm.sessions[sessionID]
            delete(sm.sessions, sessionID)

            // Remove from user sessions
            userSessions := sm.userSessions[session.UserID]
            for i, sid := range userSessions {
                if sid == sessionID {
                    sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
                    break
                }
            }

            // Clean up empty user session list
            if len(sm.userSessions[session.UserID]) == 0 {
                delete(sm.userSessions, session.UserID)
            }
        }

        if len(expiredSessions) > 0 {
            fmt.Printf("Cleaned up %d expired sessions\n", len(expiredSessions))
        }

        sm.mu.Unlock()
    }
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
    bytes := make([]byte, 32) // 256 bits
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// GetSessionHash returns a SHA-256 hash of the session ID for storage
func GetSessionHash(sessionID string) string {
    return auth.HashSHA256(sessionID)
}
