package models

import "time"

// Channel represents a chat channel
type Channel struct {
    ChannelID   int64     `json:"channel_id"`
    ChannelName string    `json:"channel_name"`
    CreatedBy   int64     `json:"created_by"`
    CreatedAt   time.Time `json:"created_at"`
    Topic       *string   `json:"topic,omitempty"`
    IsPrivate   bool      `json:"is_private"`
    MaxMembers  int       `json:"max_members"`
}

// ChannelMember represents a user's membership in a channel
type ChannelMember struct {
    MembershipID int64     `json:"membership_id"`
    ChannelID    int64     `json:"channel_id"`
    UserID       int64     `json:"user_id"`
    JoinedAt     time.Time `json:"joined_at"`
    Role         string    `json:"role"` // member, moderator, owner
    IsMuted      bool      `json:"is_muted"`
}

// Message represents a channel message
type Message struct {
    MessageID      int64     `json:"message_id"`
    ChannelID      int64     `json:"channel_id"`
    UserID         int64     `json:"user_id"`
    MessageContent string    `json:"message_content"` // Encrypted
    MessageHash    *string   `json:"message_hash,omitempty"`
    SentAt         time.Time `json:"sent_at"`
    IsDeleted      bool      `json:"is_deleted"`
}

// DirectMessage represents a private message between users
type DirectMessage struct {
    DMID           int64     `json:"dm_id"`
    SenderID       int64     `json:"sender_id"`
    RecipientID    int64     `json:"recipient_id"`
    MessageContent string    `json:"message_content"` // Encrypted
    MessageHash    *string   `json:"message_hash,omitempty"`
    SentAt         time.Time `json:"sent_at"`
    IsRead         bool      `json:"is_read"`
    IsDeleted      bool      `json:"is_deleted"`
}
