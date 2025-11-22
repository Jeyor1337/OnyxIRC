package database

import (
    "database/sql"
    "fmt"

    "github.com/onyxirc/server/internal/models"
)

// ChannelRepository handles channel-related database operations
type ChannelRepository struct {
    db *DB
}

// NewChannelRepository creates a new ChannelRepository
func NewChannelRepository(db *DB) *ChannelRepository {
    return &ChannelRepository{db: db}
}

// Create creates a new channel
func (r *ChannelRepository) Create(channelName string, createdBy int64, isPrivate bool) (*models.Channel, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO channels (channel_name, created_by, is_private)
        VALUES (?, ?, ?)
    `

    result, err := r.db.ExecContext(ctx, query, channelName, createdBy, isPrivate)
    if err != nil {
        return nil, fmt.Errorf("failed to create channel: %w", err)
    }

    channelID, err := result.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("failed to get channel ID: %w", err)
    }

    // Add creator as owner
    if err := r.AddMember(channelID, createdBy, "owner"); err != nil {
        return nil, fmt.Errorf("failed to add creator as owner: %w", err)
    }

    return r.GetByID(channelID)
}

// GetByID retrieves a channel by ID
func (r *ChannelRepository) GetByID(channelID int64) (*models.Channel, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT channel_id, channel_name, created_by, created_at, topic, is_private, max_members
        FROM channels
        WHERE channel_id = ?
    `

    channel := &models.Channel{}
    err := r.db.QueryRowContext(ctx, query, channelID).Scan(
        &channel.ChannelID,
        &channel.ChannelName,
        &channel.CreatedBy,
        &channel.CreatedAt,
        &channel.Topic,
        &channel.IsPrivate,
        &channel.MaxMembers,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("channel not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get channel: %w", err)
    }

    return channel, nil
}

// GetByName retrieves a channel by name
func (r *ChannelRepository) GetByName(channelName string) (*models.Channel, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT channel_id, channel_name, created_by, created_at, topic, is_private, max_members
        FROM channels
        WHERE channel_name = ?
    `

    channel := &models.Channel{}
    err := r.db.QueryRowContext(ctx, query, channelName).Scan(
        &channel.ChannelID,
        &channel.ChannelName,
        &channel.CreatedBy,
        &channel.CreatedAt,
        &channel.Topic,
        &channel.IsPrivate,
        &channel.MaxMembers,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("channel not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get channel: %w", err)
    }

    return channel, nil
}

// List retrieves all public channels
func (r *ChannelRepository) List() ([]*models.Channel, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT channel_id, channel_name, created_by, created_at, topic, is_private, max_members
        FROM channels
        WHERE is_private = FALSE
        ORDER BY channel_name
    `

    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to list channels: %w", err)
    }
    defer rows.Close()

    var channels []*models.Channel
    for rows.Next() {
        channel := &models.Channel{}
        err := rows.Scan(
            &channel.ChannelID,
            &channel.ChannelName,
            &channel.CreatedBy,
            &channel.CreatedAt,
            &channel.Topic,
            &channel.IsPrivate,
            &channel.MaxMembers,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan channel: %w", err)
        }
        channels = append(channels, channel)
    }

    return channels, nil
}

// AddMember adds a user to a channel
func (r *ChannelRepository) AddMember(channelID, userID int64, role string) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO channel_members (channel_id, user_id, role)
        VALUES (?, ?, ?)
    `

    _, err := r.db.ExecContext(ctx, query, channelID, userID, role)
    if err != nil {
        return fmt.Errorf("failed to add member: %w", err)
    }

    return nil
}

// RemoveMember removes a user from a channel
func (r *ChannelRepository) RemoveMember(channelID, userID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `DELETE FROM channel_members WHERE channel_id = ? AND user_id = ?`
    _, err := r.db.ExecContext(ctx, query, channelID, userID)
    if err != nil {
        return fmt.Errorf("failed to remove member: %w", err)
    }

    return nil
}

// GetMembers retrieves all members of a channel
func (r *ChannelRepository) GetMembers(channelID int64) ([]*models.ChannelMember, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT membership_id, channel_id, user_id, joined_at, role, is_muted
        FROM channel_members
        WHERE channel_id = ?
        ORDER BY joined_at
    `

    rows, err := r.db.QueryContext(ctx, query, channelID)
    if err != nil {
        return nil, fmt.Errorf("failed to get members: %w", err)
    }
    defer rows.Close()

    var members []*models.ChannelMember
    for rows.Next() {
        member := &models.ChannelMember{}
        err := rows.Scan(
            &member.MembershipID,
            &member.ChannelID,
            &member.UserID,
            &member.JoinedAt,
            &member.Role,
            &member.IsMuted,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan member: %w", err)
        }
        members = append(members, member)
    }

    return members, nil
}

// IsMember checks if a user is a member of a channel
func (r *ChannelRepository) IsMember(channelID, userID int64) (bool, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `SELECT COUNT(*) FROM channel_members WHERE channel_id = ? AND user_id = ?`
    var count int
    err := r.db.QueryRowContext(ctx, query, channelID, userID).Scan(&count)
    if err != nil {
        return false, fmt.Errorf("failed to check membership: %w", err)
    }

    return count > 0, nil
}

// GetMemberRole retrieves the role of a user in a channel
func (r *ChannelRepository) GetMemberRole(channelID, userID int64) (string, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `SELECT role FROM channel_members WHERE channel_id = ? AND user_id = ?`
    var role string
    err := r.db.QueryRowContext(ctx, query, channelID, userID).Scan(&role)
    if err == sql.ErrNoRows {
        return "", fmt.Errorf("not a member")
    }
    if err != nil {
        return "", fmt.Errorf("failed to get role: %w", err)
    }

    return role, nil
}

// UpdateTopic updates the channel topic
func (r *ChannelRepository) UpdateTopic(channelID int64, topic string) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `UPDATE channels SET topic = ? WHERE channel_id = ?`
    _, err := r.db.ExecContext(ctx, query, topic, channelID)
    if err != nil {
        return fmt.Errorf("failed to update topic: %w", err)
    }

    return nil
}

// Delete deletes a channel
func (r *ChannelRepository) Delete(channelID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `DELETE FROM channels WHERE channel_id = ?`
    _, err := r.db.ExecContext(ctx, query, channelID)
    if err != nil {
        return fmt.Errorf("failed to delete channel: %w", err)
    }

    return nil
}
