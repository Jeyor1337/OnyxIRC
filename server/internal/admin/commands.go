package admin

import (
    "fmt"
    "strconv"
    "time"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

type AdminService struct {
    userRepo     *database.UserRepository
    adminRepo    *database.AdminRepository
    securityRepo *database.SecurityRepository
}

func NewAdminService(userRepo *database.UserRepository, adminRepo *database.AdminRepository, securityRepo *database.SecurityRepository) *AdminService {
    return &AdminService{
        userRepo:     userRepo,
        adminRepo:    adminRepo,
        securityRepo: securityRepo,
    }
}

func (s *AdminService) IsAdmin(userID int64) (bool, error) {
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return false, err
    }
    return user.IsAdmin, nil
}

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

func (s *AdminService) MakeAdmin(adminID, targetUserID int64) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    if err := s.userRepo.SetAdminStatus(targetUserID, true); err != nil {
        return fmt.Errorf("failed to grant admin privileges: %w", err)
    }

    details := fmt.Sprintf("Granted admin privileges to user ID %d", targetUserID)
    s.adminRepo.LogAction(adminID, "makeadmin", &targetUserID, nil, details)

    return nil
}

func (s *AdminService) RemoveAdmin(adminID, targetUserID int64) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    if adminID == targetUserID {
        return fmt.Errorf("cannot remove your own admin privileges")
    }

    if err := s.userRepo.SetAdminStatus(targetUserID, false); err != nil {
        return fmt.Errorf("failed to revoke admin privileges: %w", err)
    }

    details := fmt.Sprintf("Revoked admin privileges from user ID %d", targetUserID)
    s.adminRepo.LogAction(adminID, "removeadmin", &targetUserID, nil, details)

    return nil
}

func (s *AdminService) BanUser(adminID int64, username, reason string, durationSeconds int) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if targetUser.IsAdmin {
        return fmt.Errorf("cannot ban admin users")
    }

    var duration *time.Duration
    if durationSeconds > 0 {
        d := time.Duration(durationSeconds) * time.Second
        duration = &d
    }

    if err := s.adminRepo.BanUser(targetUser.UserID, adminID, reason, duration); err != nil {
        return fmt.Errorf("failed to ban user: %w", err)
    }

    if err := s.userRepo.SetActiveStatus(targetUser.UserID, false); err != nil {
        return fmt.Errorf("failed to deactivate user: %w", err)
    }

    details := fmt.Sprintf("Banned user %s (ID %d): %s", username, targetUser.UserID, reason)
    s.adminRepo.LogAction(adminID, "ban", &targetUser.UserID, nil, details)

    return nil
}

func (s *AdminService) UnbanUser(adminID int64, username string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if err := s.adminRepo.UnbanUser(targetUser.UserID); err != nil {
        return fmt.Errorf("failed to unban user: %w", err)
    }

    if err := s.userRepo.SetActiveStatus(targetUser.UserID, true); err != nil {
        return fmt.Errorf("failed to reactivate user: %w", err)
    }

    details := fmt.Sprintf("Unbanned user %s (ID %d)", username, targetUser.UserID)
    s.adminRepo.LogAction(adminID, "unban", &targetUser.UserID, nil, details)

    return nil
}

func (s *AdminService) UnlockAccount(adminID int64, username string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if err := s.securityRepo.UnlockAccount(targetUser.UserID); err != nil {
        return fmt.Errorf("failed to unlock account: %w", err)
    }

    details := fmt.Sprintf("Unlocked account for user %s (ID %d)", username, targetUser.UserID)
    s.adminRepo.LogAction(adminID, "unlock", &targetUser.UserID, nil, details)

    return nil
}

func (s *AdminService) KickUser(adminID int64, username, reason string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    targetUser, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if targetUser.IsAdmin {
        return fmt.Errorf("cannot kick admin users")
    }

    details := fmt.Sprintf("Kicked user %s (ID %d): %s", username, targetUser.UserID, reason)
    s.adminRepo.LogAction(adminID, "kick", &targetUser.UserID, nil, details)

    return nil
}

func (s *AdminService) GetServerStats(adminID int64) (map[string]interface{}, error) {
    if err := s.RequireAdmin(adminID); err != nil {
        return nil, err
    }

    stats := make(map[string]interface{})

    users, err := s.userRepo.List(10000, 0) 
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

    bans, err := s.adminRepo.GetActiveBans()
    if err == nil {
        stats["active_bans"] = len(bans)
    }

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

func (s *AdminService) GetAdminLog(adminID int64, limit, offset int) ([]*models.AdminActionLog, error) {
    if err := s.RequireAdmin(adminID); err != nil {
        return nil, err
    }

    return s.adminRepo.GetAdminActionLog(limit, offset)
}

func (s *AdminService) BroadcastMessage(adminID int64, message string) error {
    if err := s.RequireAdmin(adminID); err != nil {
        return err
    }

    details := fmt.Sprintf("Broadcast message: %s", message)
    s.adminRepo.LogAction(adminID, "broadcast", nil, nil, details)

    return nil
}

func ParseDuration(durationStr string) (int, error) {
    if durationStr == "" || durationStr == "0" {
        return 0, nil 
    }

    seconds, err := strconv.Atoi(durationStr)
    if err == nil {
        return seconds, nil
    }

    duration, err := time.ParseDuration(durationStr)
    if err != nil {
        return 0, fmt.Errorf("invalid duration format: %s", durationStr)
    }

    return int(duration.Seconds()), nil
}
