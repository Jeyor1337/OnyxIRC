package security

import (
    "fmt"
    "log"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

// IPTrackingService handles IP-based security and anti-dummy system
type IPTrackingService struct {
    securityRepo   *database.SecurityRepository
    maxSuspicion   int
    enableTracking bool
}

// NewIPTrackingService creates a new IPTrackingService
func NewIPTrackingService(securityRepo *database.SecurityRepository, maxSuspicion int, enableTracking bool) *IPTrackingService {
    return &IPTrackingService{
        securityRepo:   securityRepo,
        maxSuspicion:   maxSuspicion,
        enableTracking: enableTracking,
    }
}

// CheckIPAndTrack checks the user's IP address and updates suspicion counter
// Returns error if account should be blocked
func (s *IPTrackingService) CheckIPAndTrack(userID int64, currentIP string) error {
    if !s.enableTracking {
        return nil
    }

    // Get current security status
    status, err := s.securityRepo.GetSecurityStatus(userID)
    if err != nil {
        return fmt.Errorf("failed to get security status: %w", err)
    }

    // Check if account is locked
    if status.AccountLocked {
        reason := "Account locked"
        if status.LockReason != nil {
            reason = *status.LockReason
        }
        return fmt.Errorf("account is locked: %s", reason)
    }

    // If this is the first login (no last known IP), just update it
    if status.LastKnownIP == nil {
        if err := s.securityRepo.UpdateLastKnownIP(userID, currentIP); err != nil {
            log.Printf("Warning: failed to update last known IP: %v", err)
        }
        return nil
    }

    // Check if IP has changed
    if *status.LastKnownIP != currentIP {
        log.Printf("IP change detected for user %d: %s -> %s", userID, *status.LastKnownIP, currentIP)

        // Increment suspicion count
        newCount, err := s.securityRepo.IncrementSuspicionCount(userID)
        if err != nil {
            return fmt.Errorf("failed to increment suspicion count: %w", err)
        }

        log.Printf("IP suspicion count for user %d: %d/%d", userID, newCount, s.maxSuspicion)

        // Check if suspicion count exceeds threshold
        if newCount > s.maxSuspicion {
            // Lock the account
            reason := fmt.Sprintf("Too many IP address changes (%d)", newCount)
            if err := s.securityRepo.LockAccount(userID, reason, nil); err != nil {
                return fmt.Errorf("failed to lock account: %w", err)
            }

            log.Printf("Account locked for user %d due to IP suspicion", userID)
            return fmt.Errorf("account locked due to suspicious activity: too many IP address changes")
        }

        // Update last known IP
        if err := s.securityRepo.UpdateLastKnownIP(userID, currentIP); err != nil {
            log.Printf("Warning: failed to update last known IP: %v", err)
        }
    }

    return nil
}

// UnlockAccount unlocks a user account and resets suspicion counter
func (s *IPTrackingService) UnlockAccount(userID int64) error {
    if err := s.securityRepo.UnlockAccount(userID); err != nil {
        return fmt.Errorf("failed to unlock account: %w", err)
    }

    log.Printf("Account unlocked for user %d", userID)
    return nil
}

// ResetSuspicionCount resets the IP suspicion count for a user
func (s *IPTrackingService) ResetSuspicionCount(userID int64) error {
    if err := s.securityRepo.ResetSuspicionCount(userID); err != nil {
        return fmt.Errorf("failed to reset suspicion count: %w", err)
    }

    log.Printf("IP suspicion count reset for user %d", userID)
    return nil
}

// GetSecurityStatus retrieves the security status for a user
func (s *IPTrackingService) GetSecurityStatus(userID int64) (*models.UserSecurityStatus, error) {
    return s.securityRepo.GetSecurityStatus(userID)
}

// GetLoginHistory retrieves recent login attempts for a user
func (s *IPTrackingService) GetLoginHistory(userID int64, limit int) ([]*models.UserIPTracking, error) {
    return s.securityRepo.GetLoginHistory(userID, limit)
}

// IsAccountLocked checks if an account is locked
func (s *IPTrackingService) IsAccountLocked(userID int64) (bool, error) {
    return s.securityRepo.IsAccountLocked(userID)
}

// ManualLock manually locks an account (for admin use)
func (s *IPTrackingService) ManualLock(userID int64, reason string, adminID int64) error {
    if err := s.securityRepo.LockAccount(userID, reason, &adminID); err != nil {
        return fmt.Errorf("failed to lock account: %w", err)
    }

    log.Printf("Account manually locked for user %d by admin %d: %s", userID, adminID, reason)
    return nil
}
