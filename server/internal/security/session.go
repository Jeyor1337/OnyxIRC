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

type Session struct {
    SessionID    string
    UserID       int64
    User         *models.User
    IPAddress    string
    SessionKey   []byte 
    CreatedAt    time.Time
    LastActivity time.Time
    ExpiresAt    time.Time
}

type SessionManager struct {
    sessions       map[string]*Session 
    userSessions   map[int64][]string  
    mu             sync.RWMutex
    sessionTimeout time.Duration
}

func NewSessionManager(sessionTimeout time.Duration) *SessionManager {
    sm := &SessionManager{
        sessions:       make(map[string]*Session),
        userSessions:   make(map[int64][]string),
        sessionTimeout: sessionTimeout,
    }

    go sm.cleanupExpiredSessions()

    return sm
}

func (sm *SessionManager) CreateSession(user *models.User, ipAddress string, sessionKey []byte) (*Session, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    sessionID, err := generateSessionID()
    if err != nil {
        return nil, fmt.Errorf("failed to generate session ID: %w", err)
    }

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

    sm.sessions[sessionID] = session

    if _, exists := sm.userSessions[user.UserID]; !exists {
        sm.userSessions[user.UserID] = []string{}
    }
    sm.userSessions[user.UserID] = append(sm.userSessions[user.UserID], sessionID)

    return session, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return nil, fmt.Errorf("session not found")
    }

    if time.Now().After(session.ExpiresAt) {
        return nil, fmt.Errorf("session expired")
    }

    return session, nil
}

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

func (sm *SessionManager) DestroySession(sessionID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }

    delete(sm.sessions, sessionID)

    userSessions := sm.userSessions[session.UserID]
    for i, sid := range userSessions {
        if sid == sessionID {
            sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
            break
        }
    }

    if len(sm.userSessions[session.UserID]) == 0 {
        delete(sm.userSessions, session.UserID)
    }

    return nil
}

func (sm *SessionManager) DestroyUserSessions(userID int64) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    sessionIDs, exists := sm.userSessions[userID]
    if !exists {
        return nil 
    }

    for _, sessionID := range sessionIDs {
        delete(sm.sessions, sessionID)
    }

    delete(sm.userSessions, userID)

    return nil
}

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

func (sm *SessionManager) GetActiveSessionCount() int {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    return len(sm.sessions)
}

func (sm *SessionManager) cleanupExpiredSessions() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        sm.mu.Lock()

        now := time.Now()
        expiredSessions := []string{}

        for sessionID, session := range sm.sessions {
            if now.After(session.ExpiresAt) {
                expiredSessions = append(expiredSessions, sessionID)
            }
        }

        for _, sessionID := range expiredSessions {
            session := sm.sessions[sessionID]
            delete(sm.sessions, sessionID)

            userSessions := sm.userSessions[session.UserID]
            for i, sid := range userSessions {
                if sid == sessionID {
                    sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
                    break
                }
            }

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

func generateSessionID() (string, error) {
    bytes := make([]byte, 32) 
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func GetSessionHash(sessionID string) string {
    return auth.HashSHA256(sessionID)
}
