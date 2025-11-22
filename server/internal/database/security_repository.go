package database

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/onyxirc/server/internal/models"
)

type SecurityRepository struct {
    db *DB
}

func NewSecurityRepository(db *DB) *SecurityRepository {
    return &SecurityRepository{db: db}
}

func (r *SecurityRepository) RecordLoginAttempt(userID int64, ipAddress string, isSuccessful bool, userAgent *string) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO user_ip_tracking (user_id, ip_address, is_successful, user_agent)
        VALUES (?, ?, ?, ?)
    `

    _, err := r.db.ExecContext(ctx, query, userID, ipAddress, isSuccessful, userAgent)
    if err != nil {
        return fmt.Errorf("failed to record login attempt: %w", err)
    }

    return nil
}

func (r *SecurityRepository) GetSecurityStatus(userID int64) (*models.UserSecurityStatus, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT user_id, last_known_ip, ip_suspicion_count, account_locked,
               lock_reason, locked_at, locked_by
        FROM user_security_status
        WHERE user_id = ?
    `

    status := &models.UserSecurityStatus{}
    err := r.db.QueryRowContext(ctx, query, userID).Scan(
        &status.UserID,
        &status.LastKnownIP,
        &status.IPSuspicionCount,
        &status.AccountLocked,
        &status.LockReason,
        &status.LockedAt,
        &status.LockedBy,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("security status not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get security status: %w", err)
    }

    return status, nil
}

func (r *SecurityRepository) UpdateLastKnownIP(userID int64, ipAddress string) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET last_known_ip = ?
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, ipAddress, userID)
    if err != nil {
        return fmt.Errorf("failed to update last known IP: %w", err)
    }

    return nil
}

func (r *SecurityRepository) IncrementSuspicionCount(userID int64) (int, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET ip_suspicion_count = ip_suspicion_count + 1
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return 0, fmt.Errorf("failed to increment suspicion count: %w", err)
    }

    status, err := r.GetSecurityStatus(userID)
    if err != nil {
        return 0, err
    }

    return status.IPSuspicionCount, nil
}

func (r *SecurityRepository) ResetSuspicionCount(userID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET ip_suspicion_count = 0
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("failed to reset suspicion count: %w", err)
    }

    return nil
}

func (r *SecurityRepository) DecrementSuspicionCount(userID int64) (int, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET ip_suspicion_count = CASE
            WHEN ip_suspicion_count > 0 THEN ip_suspicion_count - 1
            ELSE 0
        END
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return 0, fmt.Errorf("failed to decrement suspicion count: %w", err)
    }

    status, err := r.GetSecurityStatus(userID)
    if err != nil {
        return 0, err
    }

    return status.IPSuspicionCount, nil
}

func (r *SecurityRepository) LockAccount(userID int64, reason string, lockedBy *int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET account_locked = TRUE,
            lock_reason = ?,
            locked_at = ?,
            locked_by = ?
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, reason, time.Now(), lockedBy, userID)
    if err != nil {
        return fmt.Errorf("failed to lock account: %w", err)
    }

    return nil
}

func (r *SecurityRepository) UnlockAccount(userID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        UPDATE user_security_status
        SET account_locked = FALSE,
            ip_suspicion_count = 0,
            lock_reason = NULL,
            locked_at = NULL,
            locked_by = NULL
        WHERE user_id = ?
    `

    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("failed to unlock account: %w", err)
    }

    return nil
}

func (r *SecurityRepository) GetLoginHistory(userID int64, limit int) ([]*models.UserIPTracking, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT tracking_id, user_id, ip_address, login_timestamp, is_successful, user_agent
        FROM user_ip_tracking
        WHERE user_id = ?
        ORDER BY login_timestamp DESC
        LIMIT ?
    `

    rows, err := r.db.QueryContext(ctx, query, userID, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to get login history: %w", err)
    }
    defer rows.Close()

    var history []*models.UserIPTracking
    for rows.Next() {
        record := &models.UserIPTracking{}
        err := rows.Scan(
            &record.TrackingID,
            &record.UserID,
            &record.IPAddress,
            &record.LoginTimestamp,
            &record.IsSuccessful,
            &record.UserAgent,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan login record: %w", err)
        }
        history = append(history, record)
    }

    return history, nil
}

func (r *SecurityRepository) IsAccountLocked(userID int64) (bool, error) {
    status, err := r.GetSecurityStatus(userID)
    if err != nil {
        return false, err
    }
    return status.AccountLocked, nil
}

func (r *SecurityRepository) GetRecentSuccessfulLogins(userID int64, limit int) ([]*models.UserIPTracking, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT tracking_id, user_id, ip_address, login_timestamp, is_successful, user_agent
        FROM user_ip_tracking
        WHERE user_id = ? AND is_successful = TRUE
        ORDER BY login_timestamp DESC
        LIMIT ?
    `

    rows, err := r.db.QueryContext(ctx, query, userID, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to get recent successful logins: %w", err)
    }
    defer rows.Close()

    var history []*models.UserIPTracking
    for rows.Next() {
        record := &models.UserIPTracking{}
        err := rows.Scan(
            &record.TrackingID,
            &record.UserID,
            &record.IPAddress,
            &record.LoginTimestamp,
            &record.IsSuccessful,
            &record.UserAgent,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan login record: %w", err)
        }
        history = append(history, record)
    }

    return history, nil
}
