package database

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/onyxirc/server/internal/models"
)

// AdminRepository handles admin-related database operations
type AdminRepository struct {
    db *DB
}

// NewAdminRepository creates a new AdminRepository
func NewAdminRepository(db *DB) *AdminRepository {
    return &AdminRepository{db: db}
}

// LogAction logs an admin action
func (r *AdminRepository) LogAction(adminID int64, actionType string, targetUserID, targetChannelID *int64, details string) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO admin_action_log (admin_id, action_type, target_user_id, target_channel_id, action_details)
        VALUES (?, ?, ?, ?, ?)
    `

    _, err := r.db.ExecContext(ctx, query, adminID, actionType, targetUserID, targetChannelID, details)
    if err != nil {
        return fmt.Errorf("failed to log admin action: %w", err)
    }

    return nil
}

// GetAdminActionLog retrieves admin actions with pagination
func (r *AdminRepository) GetAdminActionLog(limit, offset int) ([]*models.AdminActionLog, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT log_id, admin_id, action_type, target_user_id, target_channel_id, action_details, performed_at
        FROM admin_action_log
        ORDER BY performed_at DESC
        LIMIT ? OFFSET ?
    `

    rows, err := r.db.QueryContext(ctx, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get admin action log: %w", err)
    }
    defer rows.Close()

    var logs []*models.AdminActionLog
    for rows.Next() {
        log := &models.AdminActionLog{}
        err := rows.Scan(
            &log.LogID,
            &log.AdminID,
            &log.ActionType,
            &log.TargetUserID,
            &log.TargetChannelID,
            &log.ActionDetails,
            &log.PerformedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan admin log: %w", err)
        }
        logs = append(logs, log)
    }

    return logs, nil
}

// BanUser creates a user ban
func (r *AdminRepository) BanUser(userID, bannedBy int64, reason string, duration *time.Duration) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    var expiresAt *time.Time
    if duration != nil {
        expiry := time.Now().Add(*duration)
        expiresAt = &expiry
    }

    query := `
        INSERT INTO user_bans (user_id, banned_by, reason, expires_at)
        VALUES (?, ?, ?, ?)
    `

    _, err := r.db.ExecContext(ctx, query, userID, bannedBy, reason, expiresAt)
    if err != nil {
        return fmt.Errorf("failed to ban user: %w", err)
    }

    return nil
}

// UnbanUser removes active bans for a user
func (r *AdminRepository) UnbanUser(userID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `UPDATE user_bans SET is_active = FALSE WHERE user_id = ? AND is_active = TRUE`

    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("failed to unban user: %w", err)
    }

    return nil
}

// IsUserBanned checks if a user is currently banned
func (r *AdminRepository) IsUserBanned(userID int64) (bool, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT COUNT(*) FROM user_bans
        WHERE user_id = ?
          AND is_active = TRUE
          AND (expires_at IS NULL OR expires_at > NOW())
    `

    var count int
    err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
    if err != nil {
        return false, fmt.Errorf("failed to check ban status: %w", err)
    }

    return count > 0, nil
}

// GetActiveBans retrieves all active bans
func (r *AdminRepository) GetActiveBans() ([]*models.UserBan, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT ban_id, user_id, banned_by, reason, banned_at, expires_at, is_active
        FROM user_bans
        WHERE is_active = TRUE
        ORDER BY banned_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to get active bans: %w", err)
    }
    defer rows.Close()

    var bans []*models.UserBan
    for rows.Next() {
        ban := &models.UserBan{}
        err := rows.Scan(
            &ban.BanID,
            &ban.UserID,
            &ban.BannedBy,
            &ban.Reason,
            &ban.BannedAt,
            &ban.ExpiresAt,
            &ban.IsActive,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan ban: %w", err)
        }
        bans = append(bans, ban)
    }

    return bans, nil
}

// GetServerConfig retrieves a server configuration value
func (r *AdminRepository) GetServerConfig(key string) (string, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `SELECT config_value FROM server_config WHERE config_key = ?`

    var value string
    err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
    if err == sql.ErrNoRows {
        return "", fmt.Errorf("config key not found: %s", key)
    }
    if err != nil {
        return "", fmt.Errorf("failed to get config: %w", err)
    }

    return value, nil
}

// SetServerConfig sets a server configuration value
func (r *AdminRepository) SetServerConfig(key, value, description string, updatedBy *int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO server_config (config_key, config_value, description, updated_by)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            config_value = VALUES(config_value),
            description = VALUES(description),
            updated_by = VALUES(updated_by),
            updated_at = CURRENT_TIMESTAMP
    `

    _, err := r.db.ExecContext(ctx, query, key, value, description, updatedBy)
    if err != nil {
        return fmt.Errorf("failed to set config: %w", err)
    }

    return nil
}
