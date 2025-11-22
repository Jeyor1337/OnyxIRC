package database

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/onyxirc/server/internal/models"
)

// UserRepository handles user-related database operations
type UserRepository struct {
    db *DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *DB) *UserRepository {
    return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(username, passwordHash, passwordSalt string) (*models.User, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        INSERT INTO users (username, password_hash, password_salt, is_active, is_admin)
        VALUES (?, ?, ?, TRUE, FALSE)
    `

    result, err := r.db.ExecContext(ctx, query, username, passwordHash, passwordSalt)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    userID, err := result.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("failed to get user ID: %w", err)
    }

    return r.GetByID(userID)
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(userID int64) (*models.User, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT user_id, username, password_hash, password_salt, created_at, updated_at,
               is_active, is_admin, last_login_time
        FROM users
        WHERE user_id = ?
    `

    user := &models.User{}
    err := r.db.QueryRowContext(ctx, query, userID).Scan(
        &user.UserID,
        &user.Username,
        &user.PasswordHash,
        &user.PasswordSalt,
        &user.CreatedAt,
        &user.UpdatedAt,
        &user.IsActive,
        &user.IsAdmin,
        &user.LastLoginTime,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT user_id, username, password_hash, password_salt, created_at, updated_at,
               is_active, is_admin, last_login_time
        FROM users
        WHERE username = ?
    `

    user := &models.User{}
    err := r.db.QueryRowContext(ctx, query, username).Scan(
        &user.UserID,
        &user.Username,
        &user.PasswordHash,
        &user.PasswordSalt,
        &user.CreatedAt,
        &user.UpdatedAt,
        &user.IsActive,
        &user.IsAdmin,
        &user.LastLoginTime,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// UpdateLastLogin updates the user's last login time
func (r *UserRepository) UpdateLastLogin(userID int64) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `UPDATE users SET last_login_time = ? WHERE user_id = ?`
    _, err := r.db.ExecContext(ctx, query, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to update last login: %w", err)
    }

    return nil
}

// SetAdminStatus sets the admin status for a user
func (r *UserRepository) SetAdminStatus(userID int64, isAdmin bool) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `UPDATE users SET is_admin = ? WHERE user_id = ?`
    _, err := r.db.ExecContext(ctx, query, isAdmin, userID)
    if err != nil {
        return fmt.Errorf("failed to set admin status: %w", err)
    }

    return nil
}

// SetActiveStatus sets the active status for a user
func (r *UserRepository) SetActiveStatus(userID int64, isActive bool) error {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `UPDATE users SET is_active = ? WHERE user_id = ?`
    _, err := r.db.ExecContext(ctx, query, isActive, userID)
    if err != nil {
        return fmt.Errorf("failed to set active status: %w", err)
    }

    return nil
}

// List retrieves all users with pagination
func (r *UserRepository) List(limit, offset int) ([]*models.User, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `
        SELECT user_id, username, password_hash, password_salt, created_at, updated_at,
               is_active, is_admin, last_login_time
        FROM users
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `

    rows, err := r.db.QueryContext(ctx, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list users: %w", err)
    }
    defer rows.Close()

    var users []*models.User
    for rows.Next() {
        user := &models.User{}
        err := rows.Scan(
            &user.UserID,
            &user.Username,
            &user.PasswordHash,
            &user.PasswordSalt,
            &user.CreatedAt,
            &user.UpdatedAt,
            &user.IsActive,
            &user.IsAdmin,
            &user.LastLoginTime,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }

    return users, nil
}

// Delete deletes a user (soft delete by setting inactive)
func (r *UserRepository) Delete(userID int64) error {
    return r.SetActiveStatus(userID, false)
}

// UsernameExists checks if a username already exists
func (r *UserRepository) UsernameExists(username string) (bool, error) {
    ctx, cancel := contextWithTimeout(defaultTimeout)
    defer cancel()

    query := `SELECT COUNT(*) FROM users WHERE username = ?`
    var count int
    err := r.db.QueryRowContext(ctx, query, username).Scan(&count)
    if err != nil {
        return false, fmt.Errorf("failed to check username: %w", err)
    }

    return count > 0, nil
}
