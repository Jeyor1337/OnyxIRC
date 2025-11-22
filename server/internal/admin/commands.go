package admin

import (
    "fmt"
    "strconv"
    "time"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

// AdminService handles admin operations
type AdminService struct {
    userRepo     *database.UserRepository
    adminRepo    *database.AdminRepository
    securityRepo *database.SecurityRepository
}

// NewAdminService creates a new AdminService
func NewAdminService(userRepo *database.UserRepository, adminRepo *database.AdminRepository, securityRepo *database.SecurityRepository) *AdminService {
    return &AdminService{
        userRepo:     userRepo,
        adminRepo:    adminRepo,
        securityRepo: securityRepo,
    }
}

// IsAdmin checks if a user is an admin
func (s *AdminService) IsAdmin(userID int64) (bool, error) {
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return false, err
    }
    return user.IsAdmin, nil
}

// RequireAdmin checks if a user is an admin and returns an error if not
func (s *AdminService) RequireAdmin(userID int64) error {
    isAdmin, err := s.IsAdmin(userID)
    if err != nil {
        return err
    }
    if !isAdmin {
        return fmt.Errorf("permission denied: admin privileges required")
    }
    return nil
}

// MakeAdmin grants admin privileges to a user
func (s *AdminService) MakeAdmin(adminID, targetUserID int64) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    if err := s.userRepo.SetAdminStatus(targetUserID, true); err != nil {
        return fmt.Errorf("failed to grant admin privileges: %w", err)
    }

    // Log action
    details := fmt.Sprintf("Granted admin privileges to user ID %d", targetUserID)
    s.adminRepo.LogAction(adminID, "makeadmin", &targetUserID, nil, details)

    return nil
}

// RemoveAdmin revokes admin privileges from a user
func (s *AdminService) RemoveAdmin(adminID, targetUserID int64) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Prevent removing yourself
    if adminID == targetUserID {
        return fmt.Errorf("cannot remove your own admin privileges")
    }

    if err := s.userRepo.SetAdminStatus(targetUserID, false); err != nil {
        return fmt.Errorf("failed to revoke admin privileges: %w", err)
    }

    // Log action
    details := fmt.Sprintf("Revoked admin privileges from user ID %d", targetUserID)
    s.adminRepo.LogAction(adminID, "removeadmin", &targetUserID, nil, details)

    return nil
}

// BanUser bans a user
func (s *AdminService) BanUser(adminID int64, username, reason string, durationSeconds int) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Get target user
    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // Prevent banning admins
    if targetUser.IsAdmin {
        return fmt.Errorf("cannot ban admin users")
    }

    // Calculate duration
    var duration *time.Duration
    if durationSeconds > 0 {
        d := time.Duration(durationSeconds) * time.Second
        duration = &d
    }

    // Create ban
    if err := s.adminRepo.BanUser(targetUser.UserID, adminID, reason, duration); err != nil {
        return fmt.Errorf("failed to ban user: %w", err)
    }

    // Deactivate user account
    if err := s.userRepo.SetActiveStatus(targetUser.UserID, false); err != nil {
        return fmt.Errorf("failed to deactivate user: %w", err)
    }

    // Log action
    details := fmt.Sprintf("Banned user %s (ID %d): %s", username, targetUser.UserID, reason)
    s.adminRepo.LogAction(adminID, "ban", &targetUser.UserID, nil, details)

    return nil
}

// UnbanUser unbans a user
func (s *AdminService) UnbanUser(adminID int64, username string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Get target user
    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // Remove ban
    if err := s.adminRepo.UnbanUser(targetUser.UserID); err != nil {
        return fmt.Errorf("failed to unban user: %w", err)
    }

    // Reactivate user account
    if err := s.userRepo.SetActiveStatus(targetUser.UserID, true); err != nil {
        return fmt.Errorf("failed to reactivate user: %w", err)
    }

    // Log action
    details := fmt.Sprintf("Unbanned user %s (ID %d)", username, targetUser.UserID)
    s.adminRepo.LogAction(adminID, "unban", &targetUser.UserID, nil, details)

    return nil
}

// UnlockAccount unlocks a user account (resets IP suspicion)
func (s *AdminService) UnlockAccount(adminID int64, username string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Get target user
    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // Unlock account
    if err := s.securityRepo.UnlockAccount(targetUser.UserID); err != nil {
        return fmt.Errorf("failed to unlock account: %w", err)
    }

    // Log action
    details := fmt.Sprintf("Unlocked account for user %s (ID %d)", username, targetUser.UserID)
    s.adminRepo.LogAction(adminID, "unlock", &targetUser.UserID, nil, details)

    return nil
}

// KickUser forcibly disconnects a user (handled by server)
func (s *AdminService) KickUser(adminID int64, username, reason string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Get target user
    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // Prevent kicking admins
    if targetUser.IsAdmin {
        return fmt.Errorf("cannot kick admin users")
    }

    // Log action (actual kick is handled by server)
    details := fmt.Sprintf("Kicked user %s (ID %d): %s", username, targetUser.UserID, reason)
    s.adminRepo.LogAction(adminID, "kick", &targetUser.UserID, nil, details)

    return nil
}

// GetServerStats retrieves server statistics
func (s *AdminService) GetServerStats(adminID int64) (map[string]interface{}, error) {
    if err := s.RequireAdmin(adminID); err != nil {
        return nil, err
    }

    stats := make(map[string]interface{})

    // Get total users
    users, err := s.userRepo.List(10000, 0) // Get all users
    if err == nil {
        stats["total_users"] = len(users)

        activeUsers := 0
        adminUsers := 0
        for _, user := range users {
            if user.IsActive {
                activeUsers++
            }
            if user.IsAdmin {
                adminUsers++
            }
        }
        stats["active_users"] = activeUsers
        stats["admin_users"] = adminUsers
    }

    // Get active bans
    bans, err := s.adminRepo.GetActiveBans()
    if err == nil {
        stats["active_bans"] = len(bans)
    }

    // Get server config
    version, err := s.adminRepo.GetServerConfig("server.version")
    if err == nil {
        stats["server_version"] = version
    }

    maxSuspicion, err := s.adminRepo.GetServerConfig("security.max_ip_suspicion")
    if err == nil {
        stats["max_ip_suspicion"] = maxSuspicion
    }

    return stats, nil
}

// GetAdminLog retrieves admin action log
func (s *AdminService) GetAdminLog(adminID int64, limit, offset int) ([]*models.AdminActionLog, error) {
    if err := s.RequireAdmin(adminID); err != nil {
        return nil, err
    }

    return s.adminRepo.GetAdminActionLog(limit, offset)
}

// BroadcastMessage sends a message to all connected users (handled by server)
func (s *AdminService) BroadcastMessage(adminID int64, message string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    // Log action
    details := fmt.Sprintf("Broadcast message: %s", message)
    s.adminRepo.LogAction(adminID, "broadcast", nil, nil, details)

    return nil
}

// ParseDuration parses a duration string like "1h", "30m", "7d"
func ParseDuration(durationStr string) (int, error) {
    if durationStr == "" || durationStr == "0" {
        return 0, nil // Permanent ban
    }

    // Parse as integer (seconds)
    seconds, err := strconv.Atoi(durationStr)
    if err == nil {
        return seconds, nil
    }

    // Parse as duration string
    duration, err := time.ParseDuration(durationStr)
    if err != nil {
        return 0, fmt.Errorf("invalid duration format: %s", durationStr)
    }

    return int(duration.Seconds()), nil
}
