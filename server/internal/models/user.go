package models

import "time"

type User struct {
    UserID       int64     `json:"user_id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"-"` 
    PasswordSalt string    `json:"-"` 
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    IsActive     bool      `json:"is_active"`
    IsAdmin      bool      `json:"is_admin"`
    LastLoginTime *time.Time `json:"last_login_time,omitempty"`
}

type UserSecurityStatus struct {
    UserID            int64      `json:"user_id"`
    LastKnownIP       *string    `json:"last_known_ip,omitempty"`
    IPSuspicionCount  int        `json:"ip_suspicion_count"`
    AccountLocked     bool       `json:"account_locked"`
    LockReason        *string    `json:"lock_reason,omitempty"`
    LockedAt          *time.Time `json:"locked_at,omitempty"`
    LockedBy          *int64     `json:"locked_by,omitempty"`
}

type UserIPTracking struct {
    TrackingID      int64     `json:"tracking_id"`
    UserID          int64     `json:"user_id"`
    IPAddress       string    `json:"ip_address"`
    LoginTimestamp  time.Time `json:"login_timestamp"`
    IsSuccessful    bool      `json:"is_successful"`
    UserAgent       *string   `json:"user_agent,omitempty"`
}

type SessionToken struct {
    TokenID      int64     `json:"token_id"`
    UserID       int64     `json:"user_id"`
    TokenHash    string    `json:"-"` 
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    time.Time `json:"expires_at"`
    LastActivity time.Time `json:"last_activity"`
    IPAddress    string    `json:"ip_address"`
    IsValid      bool      `json:"is_valid"`
}

type UserBan struct {
    BanID     int64      `json:"ban_id"`
    UserID    int64      `json:"user_id"`
    BannedBy  int64      `json:"banned_by"`
    Reason    *string    `json:"reason,omitempty"`
    BannedAt  time.Time  `json:"banned_at"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    IsActive  bool       `json:"is_active"`
}
